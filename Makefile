.PHONY: build web server run test migrate-gen image-prebuilt up-prebuilt

# Build web then server binary (embedding web/dist)
build: web server

# Build frontend to web/dist
web:
	cd web && npm ci && npm run build

# Build Go server (expects web/dist to exist for embed)
server:
	cd server && go build -o ../bin/boxpilot .

# Run server locally (dev: run web separately with npm run dev)
run: server
	./bin/boxpilot

# Run tests
test:
	cd server && go test ./...

# Build docker image from prebuilt artifacts (bin/boxpilot + web/dist)
image-prebuilt: build
	docker build -f docker/Dockerfile.prebuilt -t boxpilot:latest .

# Run compose with prebuilt image flow
up-prebuilt: build
	docker compose -f docker-compose.yml -f docker-compose.prebuilt.yml up --build

# Generate OpenAPI types for frontend
migrate-gen:
	cd web && npx openapi-typescript ../docs/api.openapi.yaml -o src/api/types.ts
