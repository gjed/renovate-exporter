//go:build integration

// Package integration contains live-check tests that validate the exporter's
// emitted OTLP stream against the Weaver registry schema.
//
// Prerequisites:
//   - `weaver` binary on PATH (run scripts/install-weaver.sh in CI)
//   - No live OTLP collector needed; weaver live-check starts its own receiver
//
// Run:
//
//	go test -tags integration -timeout 120s ./tests/integration/...
package integration

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/gjed/renovate-exporter/internal/exporter"
	sc "github.com/gjed/renovate-exporter/internal/semconv"
)

// TestLiveCheck_ConformsToSchema starts the exporter with stub metric
// emissions and runs `weaver registry live-check` to assert schema compliance.
func TestLiveCheck_ConformsToSchema(t *testing.T) {
	weaverBin := requireWeaver(t)

	// Pick a free OTLP port for the live-check receiver.
	otlpPort := freePort(t)
	otlpEndpoint := fmt.Sprintf("http://localhost:%d", otlpPort)

	// Start the MeterProvider pushing to the live-check receiver endpoint.
	cfg := exporter.Config{
		OTLPEndpoint:       otlpEndpoint,
		Protocol:           exporter.ProtocolHTTP,
		CollectionInterval: 500 * time.Millisecond, // fast cycles for the test
	}
	p, err := exporter.New(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("exporter.New: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	})

	// Emit conforming metrics using generated semconv constants.
	emitConformingMetrics(t, p)

	// ForceFlush pushes pending metric data immediately rather than waiting
	// for the next PeriodicReader cycle, eliminating timing flakiness.
	flushCtx, flushCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer flushCancel()
	if err := p.ForceFlush(flushCtx); err != nil {
		t.Logf("ForceFlush warning: %v", err) // non-fatal; live-check will still receive data
	}

	// Run `weaver registry live-check` against our registry.
	// It connects to the OTLP endpoint, receives metrics, and validates them.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx,
		weaverBin,
		"registry", "live-check",
		"-r", "../../registry/",
		"--otlp-endpoint", fmt.Sprintf("localhost:%d", otlpPort),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("weaver registry live-check failed (schema violation): %v", err)
	}
}

// TestLiveCheck_SchemaViolationDetected emits a metric with a wrong attribute
// type (integer instead of string for github.org) and asserts live-check
// returns non-zero.
func TestLiveCheck_SchemaViolationDetected(t *testing.T) {
	weaverBin := requireWeaver(t)

	otlpPort := freePort(t)
	otlpEndpoint := fmt.Sprintf("http://localhost:%d", otlpPort)

	cfg := exporter.Config{
		OTLPEndpoint:       otlpEndpoint,
		Protocol:           exporter.ProtocolHTTP,
		CollectionInterval: 500 * time.Millisecond,
	}
	p, err := exporter.New(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("exporter.New: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	})

	// Deliberately violate the schema: use an integer for github.org (must be string).
	emitViolatingMetrics(t, p)

	flushCtx, flushCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer flushCancel()
	if err := p.ForceFlush(flushCtx); err != nil {
		t.Logf("ForceFlush warning: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx,
		weaverBin,
		"registry", "live-check",
		"-r", "../../registry/",
		"--otlp-endpoint", fmt.Sprintf("localhost:%d", otlpPort),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err == nil {
		t.Fatal("expected weaver live-check to return non-zero for schema violation, but it exited 0")
	}
	t.Logf("weaver live-check correctly detected violation: %v", err)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func requireWeaver(t *testing.T) string {
	t.Helper()
	bin, err := exec.LookPath("weaver")
	if err != nil {
		t.Skip("weaver binary not found on PATH — run scripts/install-weaver.sh")
	}
	return bin
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

// emitConformingMetrics records metrics using the correct attribute types
// as defined in the Weaver registry.
func emitConformingMetrics(t *testing.T, p *exporter.Provider) {
	t.Helper()
	meter := p.MeterProvider().Meter("integration-test")

	prCount, err := meter.Int64UpDownCounter(sc.MetricGitHubPrCount)
	if err != nil {
		t.Fatalf("create pr.count: %v", err)
	}
	prCount.Add(context.Background(), 5,
		metric.WithAttributes(
			attribute.String(sc.AttrGitHubOrg, "test-org"),
			attribute.String(sc.AttrGitHubRepo, "test-repo"),
			attribute.String(sc.AttrGitHubPrState, sc.AttrGitHubPrStateOpen),
		),
	)

	scrapeDur, err := meter.Float64Histogram(sc.MetricGitHubExporterScrapeDuration)
	if err != nil {
		t.Fatalf("create scrape.duration: %v", err)
	}
	scrapeDur.Record(context.Background(), 1.23,
		metric.WithAttributes(
			attribute.String(sc.AttrExporterTarget, "test-org/test-repo"),
		),
	)
}

// emitViolatingMetrics records github.pr.count with github.org set to an
// integer, which violates the registry schema (type: string).
func emitViolatingMetrics(t *testing.T, p *exporter.Provider) {
	t.Helper()
	meter := p.MeterProvider().Meter("integration-test-bad")

	prCount, err := meter.Int64UpDownCounter(sc.MetricGitHubPrCount)
	if err != nil {
		t.Fatalf("create pr.count: %v", err)
	}
	// VIOLATION: github.org must be string, not int.
	prCount.Add(context.Background(), 3,
		metric.WithAttributes(
			attribute.Int(sc.AttrGitHubOrg, 42), // wrong type: int instead of string
			attribute.String(sc.AttrGitHubRepo, "test-repo"),
			attribute.String(sc.AttrGitHubPrState, sc.AttrGitHubPrStateOpen),
		),
	)
}
