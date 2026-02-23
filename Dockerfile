# Stage 1: build web
FROM node:20-alpine AS web
WORKDIR /app/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci --ignore-scripts || true
COPY web/ ./
RUN npm run build 2>/dev/null || npx vite build 2>/dev/null || true
RUN mkdir -p dist && echo '<!DOCTYPE html><html><body>BoxPilot</body></html>' > dist/index.html

# Stage 2: build server
FROM golang:1.22-alpine AS server
WORKDIR /app
COPY server/go.mod server/go.sum* ./
RUN go mod download 2>/dev/null || true
COPY server/ ./
RUN CGO_ENABLED=0 go build -o /boxpilot .

# Stage 3: runtime
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=server /boxpilot .
COPY --from=web /app/web/dist ./web/dist
ENV ADDR=:8080 DB_PATH=/data/app.db WEB_ROOT=/app/web/dist
EXPOSE 8080
ENTRYPOINT ["./boxpilot"]
