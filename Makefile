BINARY=nexus
MODULE=github.com/kivervinicius/ai-cli
LOCAL_BIN ?= $(HOME)/.local/bin
HOST_LOCAL_BIN = /home/desenvolvedor/.local/bin

.PHONY: all build web restart dev test race vet lint install install-local release-local bump clean

all: build

bump:
	@echo "Version bumps are managed by: make release-local"

# Rebuild frontend bundle and copy into Go embed path.
# Runs automatically before `build`; also available standalone: make web
web:
	@echo "Building frontend..."
	cd web && node scripts/build.mjs
	@echo "✓ Frontend bundle embedded into internal/control/web/dist/"

# Full build: frontend → embed → Go binary → install → restart live server.
build: web
	@VERSION=$$(cat VERSION 2>/dev/null || echo "dev"); \
	COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	BUILDDATE=$$(date -u +%Y-%m-%dT%H:%M:%SZ); \
	LDFLAGS="-s -w -X $(MODULE)/internal/buildinfo.Version=$$VERSION -X $(MODULE)/internal/buildinfo.Commit=$$COMMIT -X $(MODULE)/internal/buildinfo.BuildDate=$$BUILDDATE"; \
	echo "Building $(BINARY) v$$VERSION (commit: $$COMMIT)..."; \
	go build -ldflags="$$LDFLAGS" -o $(BINARY) ./cmd/nexus; \
	mkdir -p $(LOCAL_BIN); \
	rm -f $(LOCAL_BIN)/$(BINARY).tmp; \
	cp -f $(BINARY) $(LOCAL_BIN)/$(BINARY).tmp; \
	chmod +x $(LOCAL_BIN)/$(BINARY).tmp; \
	mv -f $(LOCAL_BIN)/$(BINARY).tmp $(LOCAL_BIN)/$(BINARY); \
	ln -sf $(LOCAL_BIN)/$(BINARY) $(LOCAL_BIN)/ai; \
	echo "✓ Built and installed $(BINARY) v$$VERSION to $(LOCAL_BIN)/$(BINARY) (alias: ai)"
	@$(MAKE) --no-print-directory restart

# Find any running `nexus web` process, capture its flags, kill it, and
# relaunch it fully detached so it survives after make exits.
# Safe when no server is running (skips gracefully).
# Delegates to scripts/restart-web.sh to avoid make sub-shell signal issues.
restart:
	@bash scripts/restart-web.sh "$(LOCAL_BIN)/$(BINARY)"

# Full dev loop alias: build everything and restart the live server.
dev: build

release-local:
	go run ./cmd/nexus release

install: build
	@if [ -n "$(DESTDIR)" ]; then \
		install -d $(DESTDIR)/usr/local/bin; \
		install -m 755 $(BINARY) $(DESTDIR)/usr/local/bin/$(BINARY); \
	fi

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
