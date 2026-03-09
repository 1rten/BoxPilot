# BoxPilot

[中文文档](./README.zh-CN.md)

BoxPilot is a self-hosted control plane for `sing-box`. It manages subscriptions, nodes, forwarding policy, and runtime config generation in one repo.

Current stack:

- Backend: Go + Gin + SQLite
- Frontend: React 18 + Vite + Ant Design
- Runtime model: process mode only; BoxPilot generates `sing-box.json` and applies it through `SINGBOX_RESTART_CMD`

## Features

- Subscription management: create, update, delete, manual refresh, auto refresh
- Subscription parsing: URI lists, sing-box JSON, Clash YAML, and base64 variants
- Node management: enable/disable, forwarding toggle, batch actions, HTTP/PING tests
- Runtime observability: status, traffic, connections, logs, proxy chain check
- Proxy settings: HTTP / SOCKS5 listen address, port, auth
- Routing settings: private bypass, custom domain/CIDR bypass
- Forwarding policy: health filter, latency threshold, untested-node policy, test concurrency
- Business groups: derive `biz-*` runtime groups from subscription rules and rule sets
- Safe apply flow: preflight check, atomic write, rollback, debounced auto reload

## Pages

- `Dashboard`: runtime overview, traffic, connections, logs, routing summary
- `Subscriptions`: subscription list, auto refresh settings, sync health
- `Nodes`: node filtering, batch forwarding, connectivity tests, detail drawer
- `Settings`
  - `Access`: global HTTP / SOCKS5 inbound settings
  - `Routing`: bypass rules and forwarding policy
  - `Runtime`: `manual` and `biz-*` runtime group selection

## Quick Start

### Option 1: prebuilt flow

```bash
git clone <repo-url>
cd BoxPilot
make up-prebuilt
```

Open `http://localhost:8080`.

If needed:

```bash
make PREBUILT_GOARCH=amd64 up-prebuilt
```

### Option 2: standard Docker build

```bash
docker compose up --build
```

Bind to localhost only:

```bash
export BIND_IP=127.0.0.1
docker compose up --build
```

Default exposed ports:

- `8080`: Web UI + API
- `7890`: HTTP proxy
- `7891`: SOCKS5 proxy

## Local Development

### Backend

```bash
cd server
ADDR=:8080 \
DB_PATH=../data/app.db \
SINGBOX_CONFIG=../data/sing-box.json \
SINGBOX_RESTART_CMD='pkill -HUP sing-box' \
go run .
```

### Frontend

```bash
cd web
npm ci
npm run dev
```

To serve the built SPA from the backend, build the frontend first and set:

```bash
WEB_ROOT=../web/dist
```

## Commands

```bash
make build
make build-prebuilt
make image-prebuilt
make up-prebuilt
make test
make migrate-gen
make diagnose
```

## Runtime Model

BoxPilot uses process mode only:

1. Load subscriptions, nodes, routing, and policy from SQLite
2. Generate runtime `sing-box` config
3. Write to `SINGBOX_CONFIG`
4. Run `SINGBOX_CHECK_CMD`
5. Run `SINGBOX_RESTART_CMD`
6. Roll back on restart failure

When forwarding is already running, these changes trigger debounced auto reload:

- node forwarding state changes
- subscription refresh changing node sets
- runtime group selection changes

## Recommended Workflow

Under the default strict policy, `Start Forwarding` needs eligible nodes.

Typical flow:

1. import or refresh subscriptions
2. enable forwarding on selected nodes
3. run node tests
4. start forwarding from the Proxy control or Settings page

If no node passes the policy filter, the common error is `CFG_NO_ENABLED_NODES`.

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `ADDR` | `:8080` | HTTP listen address |
| `DB_PATH` | auto-detected | SQLite file path |
| `WEB_ROOT` | unset | frontend static asset directory |
| `SINGBOX_CONFIG` | auto-detected | runtime config path |
| `SINGBOX_RESTART_CMD` | unset | restart/reload command |
| `SINGBOX_CHECK_CMD` | `sing-box check -c "$SINGBOX_CONFIG"` | preflight check command |
| `SINGBOX_CLASH_API_ADDR` | `127.0.0.1:9090` | runtime traffic / probe source |
| `SINGBOX_CLASH_API_SECRET` | unset | Clash API secret |
| `HTTP_PROXY_PORT` | compose-provided in container mode | bootstrap HTTP port hint |
| `SOCKS_PROXY_PORT` | compose-provided in container mode | bootstrap SOCKS port hint |
| `BACKUP_KEEP` | reserved | reserved backup retention setting |

Auto-detection:

- `DB_PATH`: `/data/app.db` if `/data` exists, otherwise `data/app.db`
- `SINGBOX_CONFIG`: `/data/sing-box.json` if `/data` exists, otherwise `data/sing-box.json`

## API Overview

Base path: `/api/v1`

- `subscriptions`: list, create, update, delete, refresh
- `nodes`: list, update, test, batch forwarding, restart forwarding
- `runtime`: status, traffic, connections, logs, proxy check, reload, groups
- `settings`: proxy settings, routing settings, forwarding policy, start/stop forwarding

Reference: [docs/api.openapi.yaml](/Users/1rten/Documents/workspace/BoxPilot/docs/api.openapi.yaml)

## Frontend Types

- generated: `web/src/api/types.gen.ts`
- compatibility layer: `web/src/api/types.compat.ts`
- app import target: `web/src/api/types.ts`

Update with:

```bash
make migrate-gen
```

## Troubleshooting

### Direct SPA routes return 404

- set `WEB_ROOT`
- make sure `web/dist/index.html` exists
- make sure built assets are included in the runtime image

### `CFG_NO_ENABLED_NODES`

Check:

```bash
curl --noproxy '*' http://127.0.0.1:8080/api/v1/settings/forwarding/summary
curl --noproxy '*' http://127.0.0.1:8080/api/v1/settings/forwarding/policy
```

### Quick proxy checks

```bash
curl --noproxy '' --proxy http://127.0.0.1:7890 https://ipinfo.io/json
curl --noproxy '' --socks5-hostname 127.0.0.1:7891 https://ipinfo.io/json
```

### Built-in diagnostics

```bash
make diagnose
```

## Security Notes

- Do not expose proxy ports directly to the public Internet.
- If listening on `0.0.0.0`, enable auth and use firewall restrictions.
- Do not commit subscription URLs or tokens.
- Restrict `SINGBOX_RESTART_CMD` to trusted scripts or fixed commands.

## Docs

- [Chinese README](./README.zh-CN.md)
- [Architecture](/Users/1rten/Documents/workspace/BoxPilot/docs/architecture.md)
- [Frontend Architecture](/Users/1rten/Documents/workspace/BoxPilot/docs/frontend-architecture.md)
- [Error Codes](/Users/1rten/Documents/workspace/BoxPilot/docs/error-codes.md)
- [Migration Guide](/Users/1rten/Documents/workspace/BoxPilot/docs/migrations.md)
- [UI Notes](/Users/1rten/Documents/workspace/BoxPilot/docs/ui-design.md)
- [中文架构文档](/Users/1rten/Documents/workspace/BoxPilot/docs/zh-CN/architecture.md)

## License

MIT
