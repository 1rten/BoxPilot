.PHONY: build web server run test migrate-gen

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

# Generate OpenAPI types for frontend
migrate-gen:
	cd web && npx openapi-typescript ../docs/api.openapi.yaml -o src/api/types.ts
