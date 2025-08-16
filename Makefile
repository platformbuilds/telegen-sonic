SHELL := /bin/bash

# ---- Project ----
APP := telegen-sonic
# Default image tag is :latest; override on CLI if needed
IMG ?= ghcr.io/platformbuilds/$(APP):latest

# ---- Build params (override with: make GOOS=linux GOARCH=arm64) ----
GOOS        ?= linux
GOARCH      ?= amd64
CGO_ENABLED ?= 0

# ---- Version info (derived from git; safe fallbacks) ----
# RAW_TAG is the nearest tag (e.g., v1.2.3). Empty if not on/behind a tag.
RAW_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "0000000")
DATE    := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Turn v1.2.3 -> release/mark-v1-2-3 ; if not on a tag -> dev
ifeq ($(RAW_TAG),)
  VERSION := dev
else
  # enforce vMAJOR.MINOR.PATCH; if not matching, fall back to dev
  TAG_OK := $(shell echo "$(RAW_TAG)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$$' && echo yes || echo no)
  ifeq ($(TAG_OK),yes)
    VERSION := $(shell echo "$(RAW_TAG)" | sed -E 's/^v([0-9]+)\.([0-9]+)\.([0-9]+)$$/release\/mark-v\1-\2-\3/')
  else
    VERSION := dev
  endif
endif

# Inject into cmd/agent and cmd/cli main package variables:
#   var version, commit, date string
LDFLAGS := -s -w \
	-X 'main.version=$(VERSION)' \
	-X 'main.commit=$(COMMIT)' \
	-X 'main.date=$(DATE)'

# ---- Paths ----
BIN_DIR := bin

.PHONY: all build bpf build-agent build-cli docker lint fmt test clean

# Default target
all: build

# ---------- eBPF ----------
bpf:
	$(MAKE) -C bpf

# ---------- Go binaries ----------
build-agent:
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
	go build -trimpath -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/agent ./cmd/agent

build-cli:
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
	go build -trimpath -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(APP) ./cmd/cli

build: bpf build-agent build-cli

# ---------- Docker ----------
docker:
	docker build -t $(IMG) -f deploy/Dockerfile .

# ---------- Quality ----------
fmt:
	gofmt -s -w .

lint:
	@echo "Run your linter of choice (golangci-lint) here."

test:
	go test -race -count=1 ./...

clean:
	rm -rf $(BIN_DIR)/*
	-$(MAKE) -C bpf clean