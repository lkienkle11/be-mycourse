# Sync permission catalog from constants, then compile without writing an .exe (discard output binary).
$ErrorActionPreference = 'Stop'
#go run ./cmd/syncpermissions # sẽ chạy trong tương lai
#if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE } # sẽ chạy trong tương lai
go build -trimpath -buildvcs=false -o NUL .
#exit $LASTEXITCODE # sẽ chạy trong tương lai
