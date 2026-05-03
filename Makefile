.PHONY: build build-nocgo

# Production build: requires CGO_ENABLED=1 and libvips-dev.
# On Ubuntu VPS: sudo apt-get install -y libvips-dev pkg-config
build:
#	go run ./cmd/syncpermissions # sẽ chạy trong tương lai
	CGO_ENABLED=1 go build -trimpath -buildvcs=false -o /dev/null .

# Pure-Go build for environments without libvips (local code check, review).
# Image uploads at runtime will return errcode 9017 (ImageEncodeBusy); all other features unaffected.
build-nocgo:
	CGO_ENABLED=0 go build -trimpath -buildvcs=false -o /dev/null .
