## 1. Project Scaffolding

- [ ] 1.1 Initialize Go module: `go mod init github.com/gjed/renovate-exporter`
- [ ] 1.2 Create directory structure: `cmd/exporter/`, `internal/config/`, `internal/github/`, `internal/discovery/`, `internal/semconv/` (placeholder), `internal/exporter/`
- [ ] 1.3 Add dependencies: `google/go-github/v62`, `shurcooL/githubv4`, `golang-jwt/jwt/v5`, `spf13/cobra`, `spf13/viper`
- [ ] 1.4 Create `Makefile` with `build`, `test`, `lint` targets (no generate yet — that's observability-contract)
- [ ] 1.5 Create minimal `cmd/exporter/main.go` with cobra root command and `--config` flag

## 2. Configuration Structs

- [ ] 2.1 Define `Config`, `Target`, `Auth`, `AppAuth`, `OrgConfig` structs in `internal/config/`
- [ ] 2.2 Implement YAML config loading with `viper`
- [ ] 2.3 Implement env var overrides for `auth.token_env` and `auth.app.private_key_env`
- [ ] 2.4 Add config validation: mutual exclusion of PAT vs App auth, required fields
- [ ] 2.5 Write unit tests for config loading (table-driven, using fixture YAML files)

## 3. GitHub Auth

- [ ] 3.1 Implement `internal/github/auth.go`: `Authenticator` interface with `Token(ctx) (string, error)` method
- [ ] 3.2 Implement `PATAuthenticator`: returns the configured token directly
- [ ] 3.3 Implement `AppAuthenticator`: JWT generation (RS256 with `golang-jwt/jwt`), REST call to exchange for installation token, token caching with TTL
- [ ] 3.4 Implement auto-refresh: if cached token expires within 5 minutes, refresh before returning
- [ ] 3.5 Implement credential validation: `Ping(ctx) error` calls `GET /user` (PAT) or `GET /app` (App) and returns a descriptive error on failure
- [ ] 3.6 Wire `Ping()` into application startup; fatal exit with clear message on failure
- [ ] 3.7 Write unit tests for both authenticators using mock HTTP server

## 4. GitHub Client

- [ ] 4.1 Implement `internal/github/client.go`: wraps REST (`go-github`) and GraphQL (`githubv4`) clients; accepts an `Authenticator`
- [ ] 4.2 Implement rate limit tracking: read `X-RateLimit-Remaining` from REST responses; parse GraphQL cost fields
- [ ] 4.3 Implement pause-on-threshold logic: block new requests when remaining < 200, resume after reset time
- [ ] 4.4 Expose rate limit state via hooks for self-metrics (callback or channel — wired up in observability-contract)
- [ ] 4.5 Write unit tests for rate limit tracking and pause logic using mock HTTP server

## 5. Repository Discovery

- [ ] 5.1 Implement `internal/discovery/discoverer.go`: `Discoverer` with `Repos(ctx) ([]Repo, error)` method
- [ ] 5.2 Implement org repo listing: paginated `GET /orgs/{org}/repos?type=all`, filter archived/fork by default
- [ ] 5.3 Implement glob include filter using `filepath.Match`
- [ ] 5.4 Implement glob exclude filter; apply after include
- [ ] 5.5 Implement explicit repo list mode (bypasses API listing)
- [ ] 5.6 Implement multi-org: iterate multiple orgs, deduplicate results
- [ ] 5.7 Implement incremental refresh: background goroutine updates repo list every `refresh_interval`
- [ ] 5.8 Log discovered repos at `info` on startup and each refresh
- [ ] 5.9 Write unit tests: autodiscovery, glob filters, explicit list, multi-org dedup (mock API server)
