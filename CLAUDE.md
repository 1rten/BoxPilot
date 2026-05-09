# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
# Build everything (web → server binary)
make build

# Build web only (outputs to web/dist)
make web

# Build server only (requires web/dist for embed)
make server

# Run tests
make test
# Or directly:
cd server && go test ./...

# Run a single test
cd server && go test ./internal/api/handlers/ -run TestProbeNode_Hysteria2UDP_LocalEcho

# Run server locally (for dev, run web separately with `cd web && npm run dev`)
make run

# Generate/update frontend API types from OpenAPI spec
make migrate-gen

# Build Docker image (prebuilt flow)
make image-prebuilt
# Run with compose
make up-prebuilt
```

## Architecture

**Stack**: Go (Gin + SQLite via modernc) backend, React 18 + Vite + Ant Design frontend.

The Go module is at `server/`. The entry point is `server/main.go`, which opens SQLite, starts the subscription auto-refresh scheduler, and runs the Gin HTTP server on `:8080`.

### Internal package structure

- **`server/internal/api/`** — Gin router (`router.go`) and HTTP handlers split by domain (subscriptions, nodes, runtime, settings). Middleware: CORS, request ID, panic recovery. When `WEB_ROOT` is set, serves the SPA from disk with SPA-route fallback.
- **`server/internal/service/`** — All business logic:
  - `config_build.go` — Reads DB state and calls the generator to produce a `sing-box` JSON config.
  - `runtime_control.go` — `Reload()` orchestrates the full apply flow: load settings → build config → preflight check → atomic write → restart → rollback on failure.
  - `runtime_apply.go` — The apply flow: write candidate config → run `SINGBOX_CHECK_CMD` → atomic write → run `SINGBOX_RESTART_CMD` → verify listeners → rollback with last-good backup on failure.
  - `runtime_auto_reload.go` — Debounced (1.2s) auto-reload coalesces multiple state changes when forwarding is active.
  - `subscription_refresh.go` — Fetches subscription URL (with ETag/If-Modified-Since), parses the body, ingests nodes + routing metadata, and triggers auto-reload.
  - `node_ingest.go` — Ingests parsed outbounds into the DB, resolving tag conflicts.
  - `forwarding_policy.go` — Health-based filtering (latency threshold, untested-node policy).
  - `scheduler.go` — Periodic subscription auto-refresh running every 30s.
  - `runtime_health.go` / `routing_settings.go` — Health checks and routing settings loading.
- **`server/internal/generator/singbox.go`** — Generates the complete `sing-box` JSON config struct from typed inputs (proxy inbounds, node outbounds, routing settings, business groups). Injects optimizations (tcp_fast_open, domain_strategy), builds selector/urltest outbounds for `manual` and `biz-*` groups, wires up Clash API, DNS, and route rules.
- **`server/internal/parser/subscription.go`** — Auto-detects subscription format from raw bytes and parses into sing-box compatible outbounds + routing metadata. Supports: sing-box JSON arrays/objects, Clash YAML, traditional URI lists (vmess/vless/trojan/ss/hysteria2/http/socks), and base64-encoded variants of all formats.
- **`server/internal/store/`** — SQLite via `modernc.org/sqlite` with `MaxOpenConns(1)`. Embedded migrations in `store/migrations/` auto-apply on startup. `repo/` sub-package contains per-table data access functions (raw `*sql.DB` queries, no ORM).
- **`server/internal/runtime/`** — Process control: validates `SINGBOX_CONFIG` / `SINGBOX_RESTART_CMD` env contract, runs `sing-box check` preflight, executes restart via `sh -lc`.
- **`server/internal/util/`** — Shared utilities: structured error codes (`errorx/`), atomic file write, JSON hashing, UUID generation, time helpers.

### Runtime model (process mode only)

BoxPilot generates a `sing-box` config, writes it atomically, then runs external commands:
1. Build config from DB state (subscriptions, nodes, routing, policy)
2. Write candidate to `{config}.candidate`
3. Run `SINGBOX_CHECK_CMD` (default: `sing-box check -c "$SINGBOX_CONFIG"`)
4. Atomic-write to `SINGBOX_CONFIG`
5. Run `SINGBOX_RESTART_CMD`
6. Wait for HTTP/SOCKS listeners to become reachable
7. On failure: rollback to previous config or `.last-good` backup

Environment variables control the runtime contract — see README.md for the full table.

### Frontend

React 18 + Vite + Ant Design in `web/`. Pages: Dashboard, Subscriptions, Nodes, Settings. API types are generated from `docs/api.openapi.yaml` into `web/src/api/types.gen.ts`, with a compatibility layer in `types.compat.ts` and a re-export in `types.ts`. The API client lives in `web/src/api/client.ts`.

### Error codes

Stable string error codes (e.g., `CFG_NO_ENABLED_NODES`, `SUB_FETCH_FAILED`) defined in `server/internal/util/errorx/codes.go`. The `errorx.AppError` struct carries code, message, and arbitrary details map — used throughout the service and handler layers.
