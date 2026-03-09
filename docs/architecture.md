# BoxPilot Architecture

[中文](./zh-CN/architecture.md)

This document describes the current implemented architecture, not an early design draft.

## System Boundary

BoxPilot is the control plane for `sing-box`. It does not implement the proxy data plane itself.

BoxPilot is responsible for:

- subscriptions and nodes
- parsing outbounds, rule sets, and business routing metadata
- storing state in SQLite
- generating runtime `sing-box` config
- running preflight and restart commands
- serving the Web UI and REST API

## High-Level Layout

```text
Browser
  -> React SPA
  -> /api/v1/*

BoxPilot
  -> Gin router
  -> handlers
  -> services
  -> SQLite repos
  -> config generator
  -> runtime check / restart

sing-box
  -> reads generated config
  -> exposes HTTP / SOCKS5
  -> optionally exposes Clash API for metrics and probes
```

## Backend Layers

`server/internal`:

- `api/`: router, handlers, DTOs, middleware
- `service/`: refresh flow, settings, runtime apply, scheduler, auto reload
- `store/`: SQLite open, migrator, repositories
- `parser/`: sing-box / Clash / URI subscription parsing
- `generator/`: final `sing-box` config generation
- `runtime/`: validate restart contract, run check/restart
- `util/`: atomic write, ids, time, error codes

## Frontend Structure

`web/src`:

- `App.tsx`: shell, navigation, locale switch, proxy control
- `pages/`: `Dashboard`, `Subscriptions`, `Nodes`, `Settings`
- `hooks/`: React Query wrappers for API state
- `api/`: Axios client, API wrappers, generated types
- `components/common/`: toast, empty state, error state

## API Groups

Base path: `/api/v1`

- `subscriptions`
- `nodes`
- `runtime`
- `settings`

The current router also includes:

- runtime group query and selection
- forwarding summary and policy endpoints
- proxy apply / runtime reload endpoints

## Data Model

Current schema is initialized by `server/internal/store/migrations/0001_init.sql`.

Core tables:

- `subscriptions`
- `nodes`
- `runtime_state`
- `proxy_settings`
- `routing_settings`
- `forwarding_policy`
- `subscription_rule_sets`
- `subscription_rules`
- `subscription_group_members`
- `runtime_group_selections`

## Subscription Refresh Flow

1. manual refresh or scheduler trigger
2. conditional fetch with `etag` / `last_modified`
3. parse supported formats
4. extract nodes, rule sets, routing rules, business groups
5. replace subscription-owned runtime metadata in a transaction
6. if forwarding is running, queue debounced reload

The scheduler checks refresh eligibility every 30 seconds. Actual refresh cadence comes from each subscription's `refresh_interval_sec`.

## Config Generation

Runtime config is built by `generator.BuildConfigWithRuntime`.

Generated parts include:

- HTTP / SOCKS5 inbounds
- fixed outbounds: `direct`, `block`
- imported node outbounds
- `manual` selector
- `biz-*` selectors when business targets exist
- `biz-*-auto` urltest outbounds when auto mode is available

Routing combines:

- private domain / CIDR bypass
- optional `geosite-cn` and `geoip-cn` direct routing
- imported rule sets
- imported business rules mapped to `biz-*` selectors
- final fallback to `manual`

## Runtime Apply and Rollback

Apply flow:

1. validate restart contract
2. write candidate config
3. run `SINGBOX_CHECK_CMD`
4. atomically replace runtime config
5. run `SINGBOX_RESTART_CMD`
6. save `.last-good` on success
7. roll back to previous or last-known-good config on failure

## Static Assets

When `WEB_ROOT` is set, Gin:

- serves real static files first
- falls back SPA routes to `index.html`
- keeps `/api/*` as API-only paths
