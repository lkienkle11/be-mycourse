#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
#go run ./cmd/syncpermissions # sẽ chạy trong tương lai
go build -trimpath -buildvcs=false -o /dev/null .
