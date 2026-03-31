.PHONY: build
build:
#	go run ./cmd/syncpermissions # sẽ chạy trong tương lai
	go build -trimpath -buildvcs=false -o /dev/null .
