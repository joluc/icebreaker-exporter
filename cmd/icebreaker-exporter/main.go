package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/joluc/icebreaker-exporter/pkg/config"
	"github.com/joluc/icebreaker-exporter/pkg/exporter"
)

func main() {
	// Configure slog
	logger := slog.Default()
	slog.SetDefault(logger)

	cfg, err := config.ParseFlags()
	if err != nil {
		slog.Error("failed to parse flags", "error", err)
		os.Exit(1)
	}

	if cfg.RefreshInterval <= 0 {
		slog.Error("refresh-interval must be > 0")
		os.Exit(1)
	}
	if cfg.RequestTimeout <= 0 {
		slog.Error("request-timeout must be > 0")
		os.Exit(1)
	}

	if len(cfg.TargetNames) == 0 {
		slog.Error("at least one vessel name must be configured")
		os.Exit(1)
	}

	exp := exporter.New(cfg)

	ctx := context.Background()
	go exp.RefreshLoop(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc(cfg.MetricsPath, exp.MetricsHandler)
	mux.HandleFunc("/healthz", exp.HealthHandler)
	mux.HandleFunc("/", exp.RootHandler())

	srv := &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info("starting nordic icebreaker exporter", "address", cfg.ListenAddress, "metrics_path", cfg.MetricsPath)
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
