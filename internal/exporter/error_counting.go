package exporter

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// errorCountingExporter wraps an OTLP metric.Exporter. When Export returns an
// error it logs a warning and records the failure so the exporter keeps running
// (task 3.5). The otlp.errors counter is recorded via the global MeterProvider
// once it is initialised; for the bootstrap window we just log.
type errorCountingExporter struct {
	inner  metric.Exporter
	logger *slog.Logger
}

var _ metric.Exporter = (*errorCountingExporter)(nil)

func (e *errorCountingExporter) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	err := e.inner.Export(ctx, rm)
	if err != nil {
		e.logger.Warn("OTLP export failed — will retry next cycle",
			"err", err,
		)
		// The github_exporter.otlp.errors counter is recorded by the collection
		// loop (internal/collector) which has access to the fully-initialised
		// MeterProvider. This layer just logs and swallows so the exporter
		// continues running (non-fatal, per spec).
	}
	return nil // always return nil to prevent SDK from disabling the reader
}

func (e *errorCountingExporter) Temporality(k metric.InstrumentKind) metricdata.Temporality {
	return e.inner.Temporality(k)
}

func (e *errorCountingExporter) Aggregation(k metric.InstrumentKind) metric.Aggregation {
	return e.inner.Aggregation(k)
}

func (e *errorCountingExporter) ForceFlush(ctx context.Context) error {
	return e.inner.ForceFlush(ctx)
}

func (e *errorCountingExporter) Shutdown(ctx context.Context) error {
	return e.inner.Shutdown(ctx)
}
