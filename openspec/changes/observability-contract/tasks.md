## 1. Weaver Registry Bootstrap

- [ ] 1.1 Install `weaver` CLI (add to `Makefile` via `go install` or download binary in CI)
- [ ] 1.2 Create `registry/registry_manifest.yaml` with project metadata and OTel semconv dependency (pinned version)
- [ ] 1.3 Create `registry/attributes.yaml` defining all custom attributes: `github.org`, `github.repo`, `github.pr.state`, `github.pr.label`, `github.pr.review_status`, `github.issue.label`, `renovate.dashboard.section`, `exporter.target`
- [ ] 1.4 Create `registry/signals.yaml` defining all metrics: `github.pr.count` (updowncounter), `github.pr.age` (gauge, seconds), `github.pr.close.duration` (histogram, seconds), `github.pr.automerged` (counter), `github.issue.count` (updowncounter), `github.issue.age` (gauge, seconds), `renovate.dashboard.queue` (gauge), `github_exporter.scrape.duration` (histogram, seconds), `github_exporter.api.errors` (counter), `github_exporter.otlp.errors` (counter)
- [ ] 1.5 Run `weaver registry check -r registry/` locally and fix any errors
- [ ] 1.6 Run `weaver registry resolve -r registry/` and review resolved output

## 2. Weaver Codegen Templates

- [ ] 2.1 Create `templates/go/` directory with Jinja2 templates for Go constants
- [ ] 2.2 Write `templates/go/metrics.go.j2` — generates typed metric name constants
- [ ] 2.3 Write `templates/go/attributes.go.j2` — generates attribute key constants
- [ ] 2.4 Run `weaver registry generate -r registry/ --templates templates/go/ go internal/semconv/` and verify output
- [ ] 2.5 Create `templates/grafana/` directory with Jinja2 template for Grafana dashboard JSON
- [ ] 2.6 Write `templates/grafana/dashboard.json.j2` — generates panels for all metric groups
- [ ] 2.7 Run `weaver registry generate -r registry/ --templates templates/grafana/ grafana dashboards/` and verify output
- [ ] 2.8 Add `make generate` target that runs both codegen commands
- [ ] 2.9 Add `make check-schema` target that runs `weaver registry check`

## 3. OTel MeterProvider Wiring

- [ ] 3.1 Add OTel Go SDK dependencies: `go.opentelemetry.io/otel`, `go.opentelemetry.io/otel/sdk/metric`, `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp`, `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc`, `go.opentelemetry.io/otel/exporters/prometheus`
- [ ] 3.2 Implement `internal/exporter/provider.go`: build `MeterProvider` with `PeriodicReader` + OTLP exporter (HTTP or gRPC based on config)
- [ ] 3.3 Implement optional `ManualReader` + Prometheus bridge exporter wiring (enabled via config)
- [ ] 3.4 Implement `/metrics` HTTP handler using `promhttp.HandlerFor` with the bridge registry
- [ ] 3.5 Implement OTLP error handling: catch push errors, increment `github_exporter.otlp.errors`, log warning, continue
- [ ] 3.6 Add OTLP header injection and TLS config support
- [ ] 3.7 Wire `MeterProvider` into application startup and graceful shutdown (flush on SIGTERM)

## 4. Exporter Self-Metrics

- [ ] 4.1 Implement `github_exporter.scrape.duration` histogram recording in the collection loop
- [ ] 4.2 Implement `github_exporter.api.errors` counter (wired into GitHub client error paths — stub hook for now, collectors fill it in)
- [ ] 4.3 Implement `github_exporter.otlp.errors` counter in OTLP error handler
- [ ] 4.4 Write unit tests for MeterProvider setup and exporter config paths

## 5. Live-Check Integration Test

- [ ] 5.1 Add `weaver` binary to integration test environment (download in CI, use in Docker Compose locally)
- [ ] 5.2 Write `tests/integration/live_check_test.go`: start exporter with stub collectors, run `weaver registry live-check`, assert exit code 0
- [ ] 5.3 Add a deliberate schema violation test: emit a wrong attribute type, assert `live-check` returns non-zero
