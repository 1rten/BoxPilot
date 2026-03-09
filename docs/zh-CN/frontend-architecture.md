# BoxPilot 前端架构

[English](../frontend-architecture.md)

本文档描述当前前端实现。

## 技术栈

- React 18
- TypeScript
- Vite 5
- React Router 6
- Ant Design 6
- TanStack Query 5
- Axios
- 自定义 i18n context

当前依赖里虽然有 `zustand`，但并未实际使用。

## 结构

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

## 数据访问模式

分成两层：

1. `api/*`：请求封装和响应映射
2. `hooks/*`：Query key、轮询、失效、toast 副作用

## 类型策略

- `types.gen.ts`
- `types.compat.ts`
- `types.ts`

应用层通过 `types.ts` 导入。
