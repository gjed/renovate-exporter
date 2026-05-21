package exporter

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	sc "github.com/gjed/renovate-exporter/internal/semconv"
)

// SelfMetrics holds the instruments for exporter self-monitoring.
// Obtain one via NewSelfMetrics after the MeterProvider is initialised.
type SelfMetrics struct {
	scrapeDuration metric.Float64Histogram
	apiErrors      metric.Int64Counter
	otlpErrors     metric.Int64Counter
}

// NewSelfMetrics creates and registers all exporter self-metric instruments
// against the global MeterProvider (set by New).
func NewSelfMetrics() (*SelfMetrics, error) {
	meter := otel.GetMeterProvider().Meter("github.com/gjed/renovate-exporter")

	scrapeDuration, err := meter.Float64Histogram(
		sc.MetricGithubexporterScrapeDuration,
		metric.WithDescription("Duration of a complete collection cycle in seconds."),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			0.1, 0.5, 1, 5, 10, 30, 60, 120, 300,
		),
	)
	if err != nil {
		return nil, err
	}

	apiErrors, err := meter.Int64Counter(
		sc.MetricGithubexporterApiErrors,
		metric.WithDescription("Number of GitHub API errors encountered during collection."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, err
	}

	otlpErrors, err := meter.Int64Counter(
		sc.MetricGithubexporterOtlpErrors,
		metric.WithDescription("Number of OTLP push failures."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, err
	}

	return &SelfMetrics{
		scrapeDuration: scrapeDuration,
		apiErrors:      apiErrors,
		otlpErrors:     otlpErrors,
	}, nil
}

// RecordScrapeDuration records the wall-clock duration of a collection cycle.
// Call with the time the cycle started; it computes the elapsed duration.
func (s *SelfMetrics) RecordScrapeDuration(ctx context.Context, target string, start time.Time) {
	s.scrapeDuration.Record(ctx,
		time.Since(start).Seconds(),
		metric.WithAttributes(
			attribute.String(sc.AttrExporterTarget, target),
		),
	)
}

// RecordAPIError increments the API error counter.
// target is the org/repo slug; endpoint is the GitHub API path that failed.
func (s *SelfMetrics) RecordAPIError(ctx context.Context, target string) {
	s.apiErrors.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String(sc.AttrExporterTarget, target),
		),
	)
}

// RecordOTLPError increments the OTLP push error counter.
func (s *SelfMetrics) RecordOTLPError(ctx context.Context, target string) {
	s.otlpErrors.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String(sc.AttrExporterTarget, target),
		),
	)
}
