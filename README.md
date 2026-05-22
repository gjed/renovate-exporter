# renovate-exporter

[![CI](https://github.com/gjed/renovate-exporter/actions/workflows/ci.yml/badge.svg)](https://github.com/gjed/renovate-exporter/actions/workflows/ci.yml)
[![Registry Validate](https://github.com/gjed/renovate-exporter/actions/workflows/registry-validate.yml/badge.svg)](https://github.com/gjed/renovate-exporter/actions/workflows/registry-validate.yml)
[![Release](https://github.com/gjed/renovate-exporter/actions/workflows/release.yml/badge.svg)](https://github.com/gjed/renovate-exporter/actions/workflows/release.yml)
[![Latest Release](https://img.shields.io/github/v/release/gjed/renovate-exporter)](https://github.com/gjed/renovate-exporter/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/gjed/renovate-exporter)](go.mod)
[![License](https://img.shields.io/github/license/gjed/renovate-exporter)](LICENSE)

GitHub metrics exporter for [Renovate](https://docs.renovatebot.com/) — collects PR and issue metrics from the GitHub API and pushes them via OTLP to any OpenTelemetry-compatible backend (Grafana Cloud, Prometheus with OTLP receiver, Datadog, etc.).

## Features

- Pushes PR and issue metrics via **OTLP** (HTTP or gRPC) as primary output
- Optional Prometheus `/metrics` bridge for scrape-based setups (disabled by default)
- Metrics conform to the [OTel semantic convention registry](registry/) and are validated by [Weaver](https://github.com/open-telemetry/weaver) in CI
- Multi-arch Docker image published to [GHCR](https://ghcr.io/gjed/renovate-exporter)
- Fully automated releases via [semantic-release](https://semantic-release.gitbook.io/) — no manual tagging

## Quick Start

```bash
docker run --rm \
  -e GITHUB_TOKEN=ghp_... \
  ghcr.io/gjed/renovate-exporter:latest \
  --orgs my-org \
  --otlp-endpoint http://my-otel-collector:4318
```

Metrics are pushed to the OTLP endpoint after each collection cycle (default: every 5 minutes).

To also expose a Prometheus scrape endpoint, add `--prometheus-enabled --prometheus-listen :9090`.

## Installation

### Docker

```bash
docker pull ghcr.io/gjed/renovate-exporter:latest
```

### Binary

Download the latest release for your platform from the [releases page](https://github.com/gjed/renovate-exporter/releases/latest), then:

```bash
tar -xzf renovate-exporter_*_linux_amd64.tar.gz
chmod +x renovate-exporter
./renovate-exporter --help
```

### From source

```bash
go install github.com/gjed/renovate-exporter/cmd/renovate-exporter@latest
```

## Configuration

| Flag                    | Env                   | Default                  | Description                                                     |
| ----------------------- | --------------------- | ------------------------ | --------------------------------------------------------------- |
| `--orgs`                | `RENOVATE_ORGS`       | —                        | Comma-separated GitHub orgs to collect                          |
| `--repos`               | `RENOVATE_REPOS`      | —                        | Comma-separated `owner/repo` pairs to collect                   |
| `--otlp-endpoint`       | `OTLP_ENDPOINT`       | **required**             | OTLP endpoint URL (e.g. `http://localhost:4318`)                |
| `--otlp-protocol`       | `OTLP_PROTOCOL`       | `http`                   | OTLP protocol: `http` or `grpc`                                 |
| `--collection-interval` | `COLLECTION_INTERVAL` | `5m`                     | How often to collect and push metrics                           |
| `--github-api-base-url` | `GITHUB_API_BASE_URL` | `https://api.github.com` | GitHub API base URL (useful for GHES)                           |
| `--prometheus-enabled`  | `PROMETHEUS_ENABLED`  | `false`                  | Enable optional Prometheus `/metrics` bridge                    |
| `--prometheus-listen`   | `PROMETHEUS_LISTEN`   | `:9090`                  | Address for Prometheus bridge (requires `--prometheus-enabled`) |

Set `GITHUB_TOKEN` (or pass `--github-token`) with a token that has `repo` read access.

## Metrics

All metric names and attributes are defined in the OTel registry at [`registry/`](registry/).
When the Prometheus bridge is enabled, OTel metric names are translated to Prometheus naming conventions automatically.

| Metric                            | Instrument    | Description                                              |
| --------------------------------- | ------------- | -------------------------------------------------------- |
| `github.pr.count`                 | UpDownCounter | PR count by repo, state, and label                       |
| `github.pr.age`                   | Gauge         | Age in seconds of the oldest open PR per repo            |
| `github.pr.close.duration`        | Histogram     | Time from open to close/merge for PRs in lookback window |
| `github.pr.automerged`            | Counter       | PRs merged with no human approval                        |
| `github.pr.review_status`         | Gauge         | Open PR count by review decision state                   |
| `github.issue.count`              | UpDownCounter | Issue count by repo, state, and label                    |
| `github.issue.age`                | Gauge         | Age in seconds of the oldest open issue per repo         |
| `renovate.dashboard.queue`        | Gauge         | Renovate dashboard PRs by queue state                    |
| `github_exporter.scrape.duration` | Histogram     | Collection cycle duration in seconds                     |
| `github_exporter.api.errors`      | Counter       | GitHub API errors by target and endpoint                 |
| `github_exporter.otlp.errors`     | Counter       | OTLP push failures                                       |

## Development

### Prerequisites

- Go 1.22+
- [golangci-lint](https://golangci-lint.run/usage/install/)
- [OTel Weaver](https://github.com/open-telemetry/weaver/releases) (for schema validation)
- [GoReleaser](https://goreleaser.com/install/) (for release builds)

### Build

```bash
go build ./...
```

### Test

```bash
go test -race ./...
```

### Lint

```bash
golangci-lint run
```

### Validate OTel registry

```bash
weaver registry check -r registry/
```

### Generate code from registry

```bash
make generate
```

## CI

| Job                 | Trigger                   | Description                                               |
| ------------------- | ------------------------- | --------------------------------------------------------- |
| `lint`              | every PR + push to `main` | golangci-lint                                             |
| `test`              | every PR + push to `main` | `go test -race`, JUnit XML upload, coverage PR comment    |
| `build`             | every PR + push to `main` | `go build ./...` for `linux/amd64`                        |
| `check-generated`   | every PR + push to `main` | Fails if `make generate` produces uncommitted changes     |
| `registry-validate` | every PR + push to `main` | Weaver schema check + emit smoke-test (separate workflow) |
| `integration-test`  | PRs targeting `main`      | Starts exporter + mock GitHub API, runs Weaver live-check |
| `release`           | push to `main`            | semantic-release → GoReleaser → GHCR                      |

## Release

Releases are fully automated. Merge to `main` and semantic-release determines the next version from [conventional commits](https://www.conventionalcommits.org/):

| Commit prefix                 | Version bump |
| ----------------------------- | ------------ |
| `feat:`                       | minor        |
| `fix:`, `perf:`               | patch        |
| `feat!:` / `BREAKING CHANGE:` | major        |
| `chore:`, `docs:`, `test:`    | no release   |

## License

[Apache 2.0](LICENSE)
