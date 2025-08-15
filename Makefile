SHELL := /bin/bash
APP := sonic-dpmon
IMG := ghcr.io/platformbuilds/sonic-dpmon:latest

.PHONY: all build bpf build-agent build-cli docker lint fmt test

all: build

bpf:
	$(MAKE) -C bpf

build-agent:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/agent ./cmd/agent

build-cli:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/sonic-dpmon ./cmd/cli

build: bpf build-agent build-cli

docker:
	docker build -t $(IMG) -f deploy/Dockerfile .

fmt:
	gofmt -s -w .

lint:
	@echo "Run your linter of choice (golangci-lint) here."

test:
	go test ./...
