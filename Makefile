BINARY  := bin/server
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build test vet fmt run clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/server

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

run: build
	$(BINARY)

clean:
	rm -rf bin
