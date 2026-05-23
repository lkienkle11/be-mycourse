.PHONY: build build-nocgo check-architecture check-layout

# DDD layer graph: reads .go-arch-lint.yml, scans internal/ packages, compares import graph to rules.
# Exit 0 = OK. Exit 1 = layer violation (e.g. application importing infra).

# GO_ARCH_LINT_VERSION ?= v1.15.0

# @export PATH="$$(go env GOPATH)/bin:$$PATH"; \
# 	if ! command -v go-arch-lint >/dev/null 2>&1; then \
# 	  echo "Installing go-arch-lint $(GO_ARCH_LINT_VERSION)..."; \
# 	  go install "github.com/fe3dback/go-arch-lint@$(GO_ARCH_LINT_VERSION)"; \
# 	fi

check-architecture:
	go-arch-lint check --project-path .

# Project layout: Go files only under allowed top-level dirs (plus root main.go). See tools/layoutguard.
check-layout:
	go run ./tools/layoutguard ./...

# Production build: requires CGO_ENABLED=1 and libvips-dev.
# On Ubuntu VPS: sudo apt-get install -y libvips-dev pkg-config
build:
#	go run ./cmd/syncpermissions # sẽ chạy trong tương lai
	CGO_ENABLED=1 go build -trimpath -buildvcs=false -o /dev/null .

# Pure-Go build for environments without libvips (local code check, review).
# Image uploads at runtime will return errcode 9017 (ImageEncodeBusy); all other features unaffected.
build-nocgo:
	CGO_ENABLED=0 go build -trimpath -buildvcs=false -o /dev/null .
