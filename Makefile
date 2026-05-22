.PHONY: build test lint

BINARY := renovate-exporter
CMD     := ./cmd/exporter

build:
	go build -o $(BINARY) $(CMD)

test:
	go test ./...

lint:
	golangci-lint run ./...
WEAVER_VERSION ?= 0.23.0
WEAVER ?= weaver

.PHONY: install-weaver
install-weaver: ## Install OTel Weaver CLI (requires cargo or downloads binary)
	@if ! command -v weaver >/dev/null 2>&1; then \
		echo "Installing weaver v$(WEAVER_VERSION) via cargo..."; \
		cargo install weaver-toolchain --version $(WEAVER_VERSION); \
	else \
		echo "weaver already installed: $$(weaver --version)"; \
	fi

.PHONY: check-schema
check-schema: ## Validate OTel Weaver registry
	$(WEAVER) registry check -r registry/

.PHONY: generate
generate: ## Generate Go constants and Grafana dashboard from Weaver registry
	$(WEAVER) registry generate \
		-r registry/ \
		--templates templates/go/ \
		go \
		internal/semconv/
	$(WEAVER) registry generate \
		-r registry/ \
		--templates templates/grafana/ \
		grafana \
		dashboards/

.PHONY: test
test: ## Run unit tests
	go test ./...

.PHONY: test-integration
test-integration: ## Run integration tests (requires live OTLP receiver)
	go test -tags integration -timeout 120s ./tests/integration/...

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
