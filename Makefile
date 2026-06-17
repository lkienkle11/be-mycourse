.PHONY: build build-nocgo check-all test-all check-architecture check-layout check-dupl

# Cross-package clone detection (golangci dupl is per-package only). Threshold 60 matches .golangci.yml settings.dupl.
DUPL_THRESHOLD ?= 60
DUPL_PATHS ?= internal

# DDD layer graph: reads .go-arch-lint.yml, scans internal/ packages, compares import graph to rules.
# Exit 0 = OK. Exit 1 = layer violation (e.g. application importing infra).
GO_ARCH_LINT_VERSION ?= v1.15.0
DUPL_VERSION ?= v0.0.0-20260401084720-c99c5cf5c202

check-architecture:
	@export PATH="$$(go env GOPATH)/bin:$$PATH"; \
	if command -v go-arch-lint >/dev/null 2>&1 && go-arch-lint version 2>/dev/null | grep -Fq "$(GO_ARCH_LINT_VERSION)"; then \
	  echo "go-arch-lint $(GO_ARCH_LINT_VERSION) already present, skipping go install"; \
	else \
	  echo "Installing go-arch-lint $(GO_ARCH_LINT_VERSION)..."; \
	  go install "github.com/fe3dback/go-arch-lint@$(GO_ARCH_LINT_VERSION)"; \
	fi; \
	go-arch-lint check --project-path .

# Project layout: Go files only under allowed top-level dirs (plus root main.go). See tools/layoutguard.
check-layout:
	@command -v go >/dev/null 2>&1 || { echo "go not found in PATH"; exit 1; }; \
	go run ./tools/layoutguard ./...

check-dupl:
	@export PATH="$$(go env GOPATH)/bin:$$PATH"; \
	if command -v dupl >/dev/null 2>&1 && go version -m "$$(command -v dupl)" 2>/dev/null | grep -Fq "$(DUPL_VERSION)"; then \
	  echo "dupl $(DUPL_VERSION) already present, skipping go install"; \
	else \
	  echo "Installing dupl $(DUPL_VERSION)..."; \
	  go install "github.com/golangci/dupl@$(DUPL_VERSION)"; \
	fi; \
	out=$$(dupl -t $(DUPL_THRESHOLD) $(DUPL_PATHS) 2>&1); \
	printf '%s\n' "$$out"; \
	if printf '%s' "$$out" | grep -q 'Found total 0 clone groups'; then \
	  exit 0; \
	fi; \
	exit 1

check-all:
	go fmt ./...
	$(MAKE) test-all
	$(MAKE) build-nocgo
	$(MAKE) build

test-all:
	go test ./...
	CGO_ENABLED=1 go test ./...
	go vet ./...
	golangci-lint cache clean && golangci-lint run
	$(MAKE) check-layout
	$(MAKE) check-architecture
	$(MAKE) check-dupl

# Production build: requires CGO_ENABLED=1 and libvips-dev.
# On Ubuntu VPS: sudo apt-get install -y libvips-dev pkg-config
build:
#	go run ./cmd/syncpermissions # sẽ chạy trong tương lai
	CGO_ENABLED=1 go build -trimpath -buildvcs=false -o /dev/null .

# Pure-Go build for environments without libvips (local code check, review).
# Image uploads at runtime will return errcode 9017 (ImageEncodeBusy); all other features unaffected.
build-nocgo:
	CGO_ENABLED=0 go build -trimpath -buildvcs=false -o /dev/null .
