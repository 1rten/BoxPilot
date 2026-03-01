UNAME_M := $(shell uname -m)
PREBUILT_GOOS ?= linux
ifeq ($(UNAME_M),x86_64)
PREBUILT_GOARCH_DEFAULT := amd64
else ifeq ($(UNAME_M),amd64)
PREBUILT_GOARCH_DEFAULT := amd64
else ifeq ($(UNAME_M),arm64)
PREBUILT_GOARCH_DEFAULT := arm64
else ifeq ($(UNAME_M),aarch64)
PREBUILT_GOARCH_DEFAULT := arm64
else
PREBUILT_GOARCH_DEFAULT := amd64
endif
PREBUILT_GOARCH ?= $(PREBUILT_GOARCH_DEFAULT)

.PHONY: build build-prebuilt web server server-prebuilt run test migrate-gen image-prebuilt up-prebuilt diagnose

# Build web then server binary (embedding web/dist)
build: web server

# Build web then Linux static server binary for prebuilt image.
build-prebuilt: web server-prebuilt

# Build frontend to web/dist
web:
	cd web && npm ci && npm run build

# Build Go server (expects web/dist to exist for embed)
server:
	mkdir -p bin
	cd server && go build -o ../bin/boxpilot .

# Build Linux static binary to avoid libc mismatch in Alpine runtime.
server-prebuilt:
	mkdir -p bin
	cd server && CGO_ENABLED=0 GOOS=$(PREBUILT_GOOS) GOARCH=$(PREBUILT_GOARCH) go build -o ../bin/boxpilot .

# Run server locally (dev: run web separately with npm run dev)
run: server
	./bin/boxpilot

# Run tests
test:
	cd server && go test ./...

# Build docker image from prebuilt artifacts (bin/boxpilot + web/dist)
image-prebuilt: build-prebuilt
	docker build -f docker/Dockerfile.prebuilt -t boxpilot:latest .

# Run compose with prebuilt image flow
up-prebuilt: build-prebuilt
	docker compose -f docker-compose.yml -f docker-compose.prebuilt.yml up --build

# Runtime diagnostics (API + proxy smoke)
diagnose:
	./scripts/diagnose-forwarding.sh

# Generate OpenAPI types for frontend
migrate-gen:
	cd web && npm run gen:types && npm run prepare:types
