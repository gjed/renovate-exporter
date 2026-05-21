.PHONY: build test lint

BINARY := renovate-exporter
CMD     := ./cmd/exporter

build:
	go build -o $(BINARY) $(CMD)

test:
	go test ./...

lint:
	golangci-lint run ./...
