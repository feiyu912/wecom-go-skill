VERSION := $(shell cat VERSION 2>/dev/null || echo "v0.1.0")

.PHONY: build test release clean

build:
	go build -o wecom-go ./cmd/wecom-go

test:
	go test ./...

release:
	VERSION=$(VERSION) bash scripts/release.sh

clean:
	rm -rf dist wecom-go wecom-go-*
