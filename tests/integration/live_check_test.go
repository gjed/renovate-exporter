//go:build integration

// Package integration contains live-check tests that validate the exporter's
// emitted OTLP stream against the Weaver registry schema.
//
// Prerequisites:
//   - Docker (preferred) or `weaver` binary on PATH
//   - If using the binary: run scripts/install-weaver.sh
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
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/gjed/renovate-exporter/internal/exporter"
	sc "github.com/gjed/renovate-exporter/internal/semconv"
)

const weaverVersion = "0.23.0"
const weaverImage = "otel/weaver:v" + weaverVersion

// TestLiveCheck_ConformsToSchema starts the exporter with stub metric
// emissions and runs `weaver registry live-check` to assert schema compliance.
func TestLiveCheck_ConformsToSchema(t *testing.T) {
	runner := requireWeaverRunner(t)

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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := runner.liveCheckCmd(ctx, otlpPort)
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
	runner := requireWeaverRunner(t)

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

	cmd := runner.liveCheckCmd(ctx, otlpPort)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err == nil {
		t.Fatal("expected weaver live-check to return non-zero for schema violation, but it exited 0")
	}
	t.Logf("weaver live-check correctly detected violation: %v", err)
}

// ── weaver runner ─────────────────────────────────────────────────────────────

// weaverRunner abstracts running weaver via Docker or a local binary.
type weaverRunner struct {
	useDocker   bool
	registryDir string // absolute path to registry/ on the host
}

// requireWeaverRunner returns a weaverRunner, preferring Docker when available.
// Skips the test if neither Docker nor the weaver binary is found.
func requireWeaverRunner(t *testing.T) weaverRunner {
	t.Helper()

	// Locate the registry/ directory relative to this test file.
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	registryDir, err := filepath.Abs(filepath.Join(repoRoot, "registry"))
	if err != nil {
		t.Fatalf("resolve registry path: %v", err)
	}

	// Prefer Docker.
	if dockerBin, err := exec.LookPath("docker"); err == nil {
		if out, err := exec.Command(dockerBin, "info").CombinedOutput(); err == nil && !strings.Contains(string(out), "ERROR") {
			t.Logf("using Docker to run weaver (%s)", weaverImage)
			return weaverRunner{useDocker: true, registryDir: registryDir}
		}
	}

	// Fall back to local binary.
	if _, err := exec.LookPath("weaver"); err == nil {
		t.Logf("using local weaver binary")
		return weaverRunner{useDocker: false, registryDir: registryDir}
	}

	t.Skip("neither Docker nor weaver binary found — run scripts/install-weaver.sh or ensure Docker is running")
	return weaverRunner{}
}

// liveCheckCmd builds the exec.Cmd for `weaver registry live-check`.
// When running via Docker, --network host is used so the container can reach
// the OTLP port opened by the test process on localhost.
func (r weaverRunner) liveCheckCmd(ctx context.Context, otlpPort int) *exec.Cmd {
	if r.useDocker {
		args := []string{
			"run", "--rm",
			"--network", "host", // reach localhost OTLP port from inside container
			"--mount", fmt.Sprintf("type=bind,source=%s,target=/workspace/registry,readonly", r.registryDir),
			"--env", "HOME=/tmp/weaver",
			weaverImage,
			"registry", "live-check",
			"-r", "/workspace/registry/",
			"--otlp-endpoint", fmt.Sprintf("localhost:%d", otlpPort),
		}
		return exec.CommandContext(ctx, "docker", args...)
	}

	return exec.CommandContext(ctx,
		"weaver",
		"registry", "live-check",
		"-r", r.registryDir+"/",
		"--otlp-endpoint", fmt.Sprintf("localhost:%d", otlpPort),
	)
}

// ── helpers ───────────────────────────────────────────────────────────────────

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
