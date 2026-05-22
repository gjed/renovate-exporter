# renovate-exporter

[![CI](https://github.com/gjed/renovate-exporter/actions/workflows/ci.yml/badge.svg)](https://github.com/gjed/renovate-exporter/actions/workflows/ci.yml)
[![Registry Validate](https://github.com/gjed/renovate-exporter/actions/workflows/registry-validate.yml/badge.svg)](https://github.com/gjed/renovate-exporter/actions/workflows/registry-validate.yml)
[![Release](https://github.com/gjed/renovate-exporter/actions/workflows/release.yml/badge.svg)](https://github.com/gjed/renovate-exporter/actions/workflows/release.yml)
[![Latest Release](https://img.shields.io/github/v/release/gjed/renovate-exporter)](https://github.com/gjed/renovate-exporter/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/gjed/renovate-exporter)](go.mod)
[![License](https://img.shields.io/github/license/gjed/renovate-exporter)](LICENSE)

Prometheus exporter for [Renovate](https://docs.renovatebot.com/) — exposes dependency update metrics from the GitHub API so you can alert on stale dependencies and track update adoption across repositories.

## Features

- Exports Renovate PR metrics (open, merged, closed, age) per repository and dependency type
- Metrics conform to the [OTel semantic convention registry](registry/) and are validated by [Weaver](https://github.com/open-telemetry/weaver) in CI
- Multi-arch Docker image published to [GHCR](https://ghcr.io/gjed/renovate-exporter)
- Fully automated releases via [semantic-release](https://semantic-release.gitbook.io/) — no manual tagging

## Quick Start

```bash
docker run --rm \
  -e GITHUB_TOKEN=ghp_... \
  -p 9090:9090 \
  ghcr.io/gjed/renovate-exporter:latest \
  --orgs my-org
```

Metrics are available at `http://localhost:9090/metrics`.

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

| Flag                    | Env                   | Default                  | Description                                  |
| ----------------------- | --------------------- | ------------------------ | -------------------------------------------- |
| `--orgs`                | `RENOVATE_ORGS`       | —                        | Comma-separated GitHub orgs to scrape        |
| `--repos`               | `RENOVATE_REPOS`      | —                        | Comma-separated `owner/repo` pairs to scrape |
| `--listen-address`      | `LISTEN_ADDRESS`      | `:9090`                  | Address to expose metrics on                 |
| `--github-api-base-url` | `GITHUB_API_BASE_URL` | `https://api.github.com` | GitHub API base URL (useful for GHES)        |
| `--interval`            | `SCRAPE_INTERVAL`     | `5m`                     | How often to poll the GitHub API             |

Set `GITHUB_TOKEN` (or pass `--github-token`) with a token that has `repo` read access.

## Metrics

All metrics are prefixed with `renovate_` and conform to the OTel registry schema in [`registry/`](registry/).

| Metric                      | Type      | Description                                    |
| --------------------------- | --------- | ---------------------------------------------- |
| `renovate_prs_open`         | Gauge     | Open Renovate PRs per repo and dependency type |
| `renovate_prs_merged_total` | Counter   | Merged Renovate PRs                            |
| `renovate_prs_closed_total` | Counter   | Closed (rejected) Renovate PRs                 |
| `renovate_pr_age_seconds`   | Histogram | Age of open Renovate PRs                       |

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
