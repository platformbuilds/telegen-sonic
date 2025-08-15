SHELL := /bin/bash
APP := telegen-sonic
IMG := ghcr.io/platformbuilds/telegen-sonic:latest

.PHONY: all build bpf build-agent build-cli docker lint fmt test

all: build

bpf:
	$(MAKE) -C bpf

build-agent:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/agent ./cmd/agent

build-cli:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/telegen-sonic ./cmd/cli

build: bpf build-agent build-cli

docker:
	docker build -t $(IMG) -f deploy/Dockerfile .

fmt:
	gofmt -s -w .

lint:
	@echo "Run your linter of choice (golangci-lint) here."

test:
	go test ./...
