package exporter

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joluc/icebreaker-exporter/pkg/config"
	"github.com/joluc/icebreaker-exporter/pkg/models"
)

type Exporter struct {
	client *http.Client
	cfg    config.Config

	mu       sync.RWMutex
	snapshot models.Snapshot

	scrapeCount uint64
}

func New(cfg config.Config) *Exporter {
	return &Exporter{
		client: &http.Client{},
		cfg:    cfg,
	}
}

func (e *Exporter) RootHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = fmt.Fprintf(w, "Nordic icebreaker exporter\nMetrics: %s\nHealth: /healthz\n", e.cfg.MetricsPath)
	}
}

func (e *Exporter) HealthHandler(w http.ResponseWriter, _ *http.Request) {
	s := e.GetSnapshot()
	if s.LastRefresh.IsZero() || s.LastRefreshError != "" {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, "not ready\n")
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ok\n")
}

func (e *Exporter) MetricsHandler(w http.ResponseWriter, _ *http.Request) {
	atomic.AddUint64(&e.scrapeCount, 1)
	s := e.GetSnapshot()
	now := float64(time.Now().Unix())

	up := 1.0
	if s.LastRefreshError != "" {
		up = 0
	}

	var b strings.Builder
	b.Grow(4096)

	writeMetricHeader(&b, "icebreaker_up", "Whether the latest Digitraffic refresh succeeded", "gauge")
	fmt.Fprintf(&b, "icebreaker_up %.0f\n", up)

	writeMetricHeader(&b, "icebreaker_last_refresh_timestamp_seconds", "Unix timestamp of last refresh", "gauge")
	fmt.Fprintf(&b, "icebreaker_last_refresh_timestamp_seconds %.0f\n", float64(s.LastRefresh.Unix()))

	writeMetricHeader(&b, "icebreaker_refresh_duration_seconds", "Duration of latest refresh operation", "gauge")
	fmt.Fprintf(&b, "icebreaker_refresh_duration_seconds %.6f\n", s.RefreshDuration.Seconds())

	writeMetricHeader(&b, "icebreaker_scrapes_total", "Total number of /metrics scrapes", "counter")
	fmt.Fprintf(&b, "icebreaker_scrapes_total %d\n", atomic.LoadUint64(&e.scrapeCount))

	writeMetricHeader(&b, "icebreaker_positions", "Number of exported icebreaker positions", "gauge")
	fmt.Fprintf(&b, "icebreaker_positions %d\n", len(s.Positions))

	writeMetricHeader(&b, "icebreaker_latitude_degrees", "Current latitude of a Nordic icebreaker", "gauge")
	writeMetricHeader(&b, "icebreaker_longitude_degrees", "Current longitude of a Nordic icebreaker", "gauge")
	writeMetricHeader(&b, "icebreaker_last_report_timestamp_seconds", "Unix timestamp of the vessel position report", "gauge")
	writeMetricHeader(&b, "icebreaker_report_age_seconds", "Seconds since the latest vessel position report", "gauge")

	for _, pos := range s.Positions {
		labels := fmt.Sprintf(`vessel_name="%s",mmsi="%s",country="%s"`, EscapeLabel(pos.Name), EscapeLabel(pos.MMSI), EscapeLabel(pos.Country))
		fmt.Fprintf(&b, "icebreaker_latitude_degrees{%s} %.6f\n", labels, pos.Latitude)
		fmt.Fprintf(&b, "icebreaker_longitude_degrees{%s} %.6f\n", labels, pos.Longitude)
		if pos.Timestamp > 0 {
			fmt.Fprintf(&b, "icebreaker_last_report_timestamp_seconds{%s} %d\n", labels, pos.Timestamp)
			fmt.Fprintf(&b, "icebreaker_report_age_seconds{%s} %.0f\n", labels, math.Max(0, now-float64(pos.Timestamp)))
		}
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = io.WriteString(w, b.String())
}

func writeMetricHeader(b *strings.Builder, metric, help, metricType string) {
	fmt.Fprintf(b, "# HELP %s %s\n", metric, help)
	fmt.Fprintf(b, "# TYPE %s %s\n", metric, metricType)
}

func EscapeLabel(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}

func (e *Exporter) RefreshLoop(ctx context.Context) {
	e.Refresh(ctx)

	ticker := time.NewTicker(e.cfg.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.Refresh(ctx)
		}
	}
}

func (e *Exporter) Refresh(ctx context.Context) {
	start := time.Now()
	positions, err := fetchPositions(ctx, e.client, e.cfg)
	duration := time.Since(start)

	e.mu.Lock()
	defer e.mu.Unlock()

	s := models.Snapshot{
		LastRefresh:     time.Now(),
		RefreshDuration: duration,
	}
	if err != nil {
		slog.Error("refresh failed", "error", err)
		s.Positions = e.snapshot.Positions
		s.LastRefreshError = err.Error()
	} else {
		s.Positions = positions
		slog.Info("refreshed icebreaker positions", "count", len(positions), "durationMs", duration.Milliseconds())
	}

	e.snapshot = s
}

func (e *Exporter) GetSnapshot() models.Snapshot {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.snapshot
}
