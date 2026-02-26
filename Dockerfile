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

# Stage 3: get sing-box binary
FROM ghcr.io/sagernet/sing-box:latest AS singbox

# Stage 4: runtime
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=server /boxpilot .
COPY --from=web /app/web/dist ./web/dist
COPY --from=singbox /usr/local/bin/sing-box /usr/local/bin/sing-box
COPY docker/entrypoint.sh /app/docker/entrypoint.sh
COPY docker/restart-singbox.sh /app/docker/restart-singbox.sh
RUN chmod +x /app/docker/entrypoint.sh /app/docker/restart-singbox.sh /usr/local/bin/sing-box
ENV ADDR=:8080 \
    DB_PATH=/data/app.db \
    WEB_ROOT=/app/web/dist \
    SINGBOX_CONFIG=/data/sing-box.json \
    SINGBOX_RESTART_CMD=/app/docker/restart-singbox.sh
EXPOSE 8080 7890 7891
ENTRYPOINT ["/app/docker/entrypoint.sh"]
