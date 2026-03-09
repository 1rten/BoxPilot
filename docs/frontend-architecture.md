# BoxPilot Frontend Architecture

[中文](./zh-CN/frontend-architecture.md)

This document reflects the current frontend implementation.

## Stack

- React 18
- TypeScript
- Vite 5
- React Router 6
- Ant Design 6
- TanStack Query 5
- Axios
- custom i18n context

`zustand` exists in dependencies but is not used in the current code.

## App Bootstrap

`web/src/main.tsx` initializes:

1. `QueryClientProvider`
2. `I18nProvider`
3. `ToastProvider`
4. `App`

## Structure

```text
web/src/
  api/
  components/
  hooks/
  i18n/
  pages/
  utils/
  App.tsx
  main.tsx
```

## Data Access Pattern

Two layers are used:

1. `api/*`: request wrappers and response mapping
2. `hooks/*`: query keys, polling, invalidation, toast side effects

Examples:

- `useSubscriptions`
- `useNodes`
- `useRuntimeStatus`
- `useRuntimeTraffic`
- `useProxySettings`

## API Client

`web/src/api/client.ts` currently:

- uses `/api/v1` as `baseURL`
- sets JSON content type
- attaches backend error payload to `error.appError`

UI code usually reads messages in this order:

1. `error.appError.message`
2. `error.response.data.error.message`
3. `error.message`

## Type Strategy

The project does not import generated OpenAPI output directly.

- generated: `types.gen.ts`
- compatibility layer: `types.compat.ts`
- app import entry: `types.ts`

## Polling

Current polling-heavy queries:

- runtime status: 8s
- runtime traffic: 4s
- runtime connections: 3s
- runtime logs: 5s by default
- runtime groups: 10s
- forwarding summary: 5s

## UI Model

- `Dashboard`: runtime and diagnostics
- `Subscriptions`: list, search, auto polling, modal editing
- `Nodes`: batch forwarding, tests, drawer details
- `Settings`: access, routing, runtime sections with explicit apply step

## i18n

Files:

- `web/src/i18n/context.tsx`
- `web/src/i18n/en.ts`
- `web/src/i18n/zh.ts`

New user-facing text should update both locales.
