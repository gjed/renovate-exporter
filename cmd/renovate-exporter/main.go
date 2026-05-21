// Command renovate-exporter collects GitHub/Renovate metrics and exports them
// via OTLP (primary) with an optional Prometheus /metrics bridge (secondary).
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gjed/renovate-exporter/internal/exporter"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	cfg := exporter.Config{
		OTLPEndpoint:       envOr("OTLP_ENDPOINT", "http://localhost:4318"),
		Protocol:           exporter.Protocol(envOr("OTLP_PROTOCOL", "http")),
		CollectionInterval: mustDuration(envOr("COLLECTION_INTERVAL", "5m")),
		PrometheusEnabled:  os.Getenv("PROMETHEUS_ENABLED") == "true",
		PrometheusAddr:     envOr("PROMETHEUS_ADDR", ":9090"),
		TLSInsecure:        os.Getenv("OTLP_TLS_INSECURE") == "true",
	}

	// Build auth header from env if present (OTLP_AUTH_HEADER=Bearer <token>)
	if authHeader := os.Getenv("OTLP_AUTH_HEADER"); authHeader != "" {
		cfg.Headers = map[string]string{"Authorization": authHeader}
	}

	provider, err := exporter.New(ctx, cfg, logger)
	if err != nil {
		logger.Error("failed to initialise metrics provider", "err", err)
		os.Exit(1)
	}

	selfMetrics, err := exporter.NewSelfMetrics()
	if err != nil {
		logger.Error("failed to register self-metrics", "err", err)
		os.Exit(1)
	}

	logger.Info("renovate-exporter started",
		"otlp_endpoint", cfg.OTLPEndpoint,
		"protocol", cfg.Protocol,
		"collection_interval", cfg.CollectionInterval,
		"prometheus_enabled", cfg.PrometheusEnabled,
	)

	// ── Collection loop ───────────────────────────────────────────────────────
	ticker := time.NewTicker(cfg.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("shutting down — flushing metrics")
			shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := provider.Shutdown(shutCtx); err != nil {
				logger.Error("shutdown error", "err", err)
			}
			return
		case <-ticker.C:
			runCollectionCycle(ctx, selfMetrics, logger)
		}
	}
}

// runCollectionCycle is a stub; the data-collectors change will fill this in.
func runCollectionCycle(ctx context.Context, sm *exporter.SelfMetrics, logger *slog.Logger) {
	const target = "stub" // replaced by real target from config
	start := time.Now()
	defer sm.RecordScrapeDuration(ctx, target, start)

	logger.Info("collection cycle started", "target", target)
	// TODO: invoke collectors (github-client change)
	logger.Info("collection cycle complete",
		"target", target,
		"duration", time.Since(start),
	)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		slog.Error("invalid duration", "value", s, "err", err)
		os.Exit(1)
	}
	return d
}
