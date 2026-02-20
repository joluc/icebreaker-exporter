package exporter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/joluc/icebreaker-exporter/pkg/config"
	"github.com/joluc/icebreaker-exporter/pkg/models"
)

func TestRootHandler(t *testing.T) {
	exp := New(config.Config{MetricsPath: "/metrics"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler := exp.RootHandler()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if !strings.Contains(rr.Body.String(), "/metrics") {
		t.Errorf("handler returned unexpected body: %v", rr.Body.String())
	}
}

func TestHealthHandler(t *testing.T) {
	exp := New(config.Config{})

	// Test case: Not ready (initially empty snapshot with no LastRefresh)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	exp.HealthHandler(rr, req)
	if status := rr.Code; status != http.StatusServiceUnavailable {
		t.Errorf("expected 503 Service Unavailable, got %v", status)
	}

	// Test case: Ready (mock a successful refresh)
	exp.snapshot = models.Snapshot{
		LastRefresh: time.Now(),
	}
	rr = httptest.NewRecorder()
	exp.HealthHandler(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected 200 OK, got %v", status)
	}

	// Test case: Not ready due to error
	exp.snapshot = models.Snapshot{
		LastRefresh:      time.Now(),
		LastRefreshError: "failed to fetch",
	}
	rr = httptest.NewRecorder()
	exp.HealthHandler(rr, req)
	if status := rr.Code; status != http.StatusServiceUnavailable {
		t.Errorf("expected 503 Service Unavailable on error, got %v", status)
	}
}

func TestMetricsHandler(t *testing.T) {
	exp := New(config.Config{})

	exp.snapshot = models.Snapshot{
		LastRefresh:     time.Now(),
		RefreshDuration: 100 * time.Millisecond,
		Positions: []models.IcebreakerPosition{
			{
				Name:      "OTSO",
				MMSI:      "123456",
				Country:   "FI",
				Latitude:  60.1,
				Longitude: 24.9,
				Timestamp: time.Now().Unix(),
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()

	exp.MetricsHandler(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected 200 OK, got %v", status)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `icebreaker_latitude_degrees{vessel_name="OTSO",mmsi="123456",country="FI"} 60.1`) {
		t.Errorf("missing or incorrect metric output for latitude:\n%s", body)
	}
	if !strings.Contains(body, `icebreaker_up 1`) {
		t.Errorf("missing or incorrect metric output for up state:\n%s", body)
	}
}
