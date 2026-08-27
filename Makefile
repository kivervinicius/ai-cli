BINARY=ai
MODULE=github.com/kivervinicius/ai-cli

.PHONY: all build test race vet lint clean install

all: build

build:
	go build -ldflags="-s -w" -o $(BINARY) ./cmd/ai

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
