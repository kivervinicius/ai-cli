BINARY=ai
MODULE=github.com/kivervinicius/ai-cli

VERSION   ?= $(shell cat VERSION 2>/dev/null || echo dev)
COMMIT    ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILDDATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS = -s -w \
	-X $(MODULE)/internal/buildinfo.Version=$(VERSION) \
	-X $(MODULE)/internal/buildinfo.Commit=$(COMMIT) \
	-X $(MODULE)/internal/buildinfo.BuildDate=$(BUILDDATE)

.PHONY: all build test race vet lint install clean

all: build

build:
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/ai

test:
	go test -v ./...

race:
	go test -race ./...

vet:
	go vet ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY)

install: build
	install -d $(DESTDIR)/usr/local/bin
	install -m 755 $(BINARY) $(DESTDIR)/usr/local/bin/$(BINARY)
