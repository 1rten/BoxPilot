# BoxPilot 前端架构规范文档

版本：v0.1  
技术栈：Vite + React 18 + TypeScript  
目标：工程级、可维护、类型安全、可扩展

---

## 1. 设计目标

### 1.1 核心目标

BoxPilot 前端必须满足：

- **类型安全**（与后端 OpenAPI 完全对齐）
- **不产生 API 字段漂移**
- **错误处理统一**
- **状态管理清晰**
- **易扩展**
- **可长期维护**
- **不过度工程化**

---

## 2. 技术选型

| 类型 | 技术 |
|------|------|
| 构建工具 | Vite |
| 框架 | React 18 |
| 语言 | TypeScript |
| API 客户端 | Axios |
| 类型生成 | openapi-typescript |
| 服务器状态管理 | TanStack Query（React Query） |
| UI 本地状态 | Zustand |
| 路由 | React Router |
| 表单 | 原生 + 自定义 Hook（暂不引入重型表单库） |

---

## 3. 前端分层架构

严格遵循分层模型：

```
Pages
  ↓
Domain Hooks
  ↓
API Client (typed)
  ↓
HTTP Layer
```

### 3.1 分层说明

**1️⃣ Page 层**

- 只负责布局与组合
- 不直接写 API 调用
- 不直接写复杂业务逻辑

**2️⃣ Hooks 层（Domain Layer）**

- `useSubscriptions`、`useNodes`、`useRuntime`
- 负责：调用 API、管理缓存、错误处理封装、向 Page 暴露简单状态

**3️⃣ API Client 层**

- 只负责 HTTP 请求
- 不做业务逻辑
- 所有类型来自 OpenAPI 自动生成

---

## 4. 项目目录结构规范

```
web/src/
├── api/
│   ├── client.ts
│   ├── types.ts              # openapi 自动生成
│   ├── subscriptions.ts
│   ├── nodes.ts
│   └── runtime.ts
├── hooks/
│   ├── useSubscriptions.ts
│   ├── useNodes.ts
│   └── useRuntime.ts
├── pages/
│   ├── Dashboard.tsx
│   ├── Subscriptions.tsx
│   └── Nodes.tsx
├── components/
│   ├── layout/
│   │   ├── Layout.tsx
│   │   └── Sidebar.tsx
│   └── common/
│       ├── Table.tsx
│       ├── Modal.tsx
│       ├── Button.tsx
│       └── Spinner.tsx
├── store/
│   └── uiStore.ts
├── utils/
│   ├── error.ts
│   └── date.ts
├── App.tsx
└── main.tsx
```

---

## 5. API 类型安全规范

### 5.1 类型必须由 OpenAPI 自动生成

**禁止手写后端 DTO 类型。**

生成命令：

```bash
openapi-typescript ../docs/api.openapi.yaml -o src/api/types.ts
```

使用方式：

```ts
import type { components } from "./types";

type Subscription = components["schemas"]["Subscription"];
```

### 5.2 API Client 规范

**client.ts** 示例：

```ts
const api = axios.create({
  baseURL: "/api/v1",
});
```

必须：

- 全局错误拦截
- 统一返回 `data`
- 抛出标准 `ErrorEnvelope`

---

## 6. 错误处理架构

### 6.1 全局错误结构

后端返回：

```json
{
  "error": {
    "code": "SUB_FETCH_FAILED",
    "message": "subscription fetch failed",
    "details": {}
  }
}
```

前端必须：

- 解析 `error.code`
- 显示 `error.message`
- 可根据 `code` 做特殊处理

### 6.2 错误处理流程

```
API 调用失败
  ↓
axios interceptor
  ↓
解析 error envelope
  ↓
throw AppError(code, message, details)
  ↓
React Query onError
  ↓
统一 Toast / Modal
```

### 6.3 错误行为规范

| 错误码 | 前端行为 |
|--------|----------|
| JOB_RELOAD_IN_PROGRESS | 显示 warning toast |
| SUB_NOT_FOUND | 自动刷新列表 |
| RT_RESTART_FAILED | 弹出日志窗口 |
| REQ_VALIDATION_FAILED | 高亮表单字段 |

---

## 7. 状态管理规范

### 7.1 状态分类

| 类型 | 管理方式 |
|------|----------|
| subscriptions 列表 | React Query |
| nodes 列表 | React Query |
| runtime 状态 | React Query |
| 当前 Modal 状态 | Zustand |
| 当前选中 Tab | Zustand |

### 7.2 严禁做的事

- ❌ 不要把服务器数据放入 Zustand
- ❌ 不要在组件内直接缓存 API 结果
- ❌ 不要用 `useEffect` 手动 fetch

---

## 8. React Query 使用规范

### 8.1 Query Key 规范

```ts
["subscriptions"];
["nodes", filters];
["runtime-status"];
```

### 8.2 Mutation 规范

```ts
const mutation = useMutation({
  mutationFn: api.updateSubscription,
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ["subscriptions"] });
  },
});
```

---

## 9. 页面结构规范

### 9.1 Dashboard

显示：

- 当前 config_version
- 节点数量
- 上次 reload 时间
- Reload 按钮

### 9.2 Subscriptions

功能：列表、创建、更新、删除、Refresh。

### 9.3 Nodes

功能：列表、启用/禁用、搜索、查看 tag。

---

## 10. UI 组件规范

### 10.1 原子组件

- Button、Input、Modal、Table、Badge、Toast

必须：

- 无业务逻辑
- 可复用

---

## 11. 构建与环境规范

### 11.1 开发模式

- Vite dev server
- proxy `/api` → `http://localhost:8080`

**vite.config.ts** 示例：

```ts
export default defineConfig({
  server: {
    proxy: {
      "/api": "http://localhost:8080",
    },
  },
});
```

### 11.2 生产模式

- `vite build` → dist
- 由 Go 静态托管
- 无跨域

---

## 12. 性能与缓存策略

- React Query 默认缓存 30 秒
- 不做本地持久缓存
- 不做 localStorage 缓存节点数据

---

## 13. 日志与调试

**开发环境：**

- `console.error` 输出 AppError
- 显示 `req_id`

**生产环境：**

- 不输出敏感 details
- 不打印 outbound_json

---

## 14. 可扩展设计

预留扩展点（v0.1 不实现）：

- 多 profile 支持
- 主题系统
- 实时日志 SSE
- urltest 结果展示

---

## 15. 前端非目标声明（v0.1）

- 不做节点测速图表
- 不做分流规则编辑
- 不做多用户系统
- 不做权限控制
- 不做复杂动画

---

## 16. 版本一致性要求

前端版本必须与后端版本一致发布：

- Docker 镜像同版本
- UI build embed 进 Go 二进制
- 不允许 UI 与 API 版本错配

---

## 17. 安全声明

前端：

- 不存储订阅 URL 在 localStorage
- 不暴露敏感信息
- 不持久化 runtime 状态

---

## 18. 总结

BoxPilot 前端遵循：

- **类型驱动开发**
- **API-first**
- **分层架构**
- **状态分离**
- **统一错误处理**
- **构建一致性**

这保证：

- 长期可维护
- 不出现字段漂移
- 不会写成“组件里到处写 API”
