# UI/交互设计规范（必须遵守）

请按现代轻量 SaaS 风格实现 UI：更柔和的边框、更卡片化、更友好的空状态与文案，但仍保持管理台的信息密度。
页面主体外层用 Card 包裹
空状态提供引导说明（例如 “Create your first subscription to start syncing.”）

## 1. 视觉风格（Design Tokens）
- 主题：浅色（light）
- 字体：系统默认字体栈即可
- 圆角：8px（卡片、按钮、输入框统一）
- 间距：采用 8px 网格（8/16/24/32）
- 阴影：轻阴影（卡片/弹窗使用，表格不加阴影）
- 边框：使用浅灰边框，尽量减少强对比线条

## 2. 页面布局
- 顶部 Header：标题 + 右侧操作区（New / Refresh）
- 主体：优先 Table 展示，支持容器自适应宽度
- 表格上方可预留 Filter 区（可先不实现搜索）
- 页面内容区域最大宽度不做限制，适配桌面为主

## 3. Table（信息密度与可读性）
- Table size: middle（不要太松也不要太挤）
- 列定义：
  - Name：左对齐，可点击复制（可选）
  - Status：用 Tag，视觉上突出
  - UpdatedAt：等宽字体展示更好（可选），格式 YYYY-MM-DD HH:mm:ss
  - Actions：右对齐，使用 link 按钮风格
- Actions 规范：
  - Edit：link 按钮
  - Refresh：link 按钮，刷新中显示 loading
  - Delete：link + danger，必须二次确认

## 4. 状态（Status Tag 规范）
- active：成功态（绿色系）
- paused：警告态（黄色/橙色系）
- error：失败态（红色系）
Tag 文案首字母大写：Active / Paused / Error

## 5. 交互反馈（必须一致）
- 所有异步操作：
  - 按钮进入 loading
  - 禁止重复点击
  - 成功 toast：动词 + 名词（例如 “Created subscription”）
  - 失败 toast：使用后端 message；无 message 用通用文案
- 列表刷新：
  - 不要全屏 loading，使用表格 loading
  - ErrorState 提供 Retry 按钮

## 6. Modal（新增/编辑）
- Modal 标题：
  - Create Subscription / Edit Subscription
- 布局：label 左对齐，表单字段宽度统一
- 行为：
  - Esc 关闭
  - 点击遮罩可关闭
  - 关闭时 reset 表单
  - Enter 在表单内触发提交（若不冲突）

## 7. Empty / Error / Loading
- EmptyState：
  - 图标 + 简短说明 + 主按钮（New Subscription）
- ErrorState：
  - 图标 + 错误说明 + Retry
- Loading：
  - 首次加载：Skeleton（可选）
  - 后续刷新：Table loading

## 8. 文案风格
- 简短、明确、偏英文控制台风格
- 禁止使用夸张语气或 marketing 文案

# BoxPilot 前端实现 Prompt（可直接喂给 AI）

你是一个资深前端工程师。请基于以下约束与规范，生成一个可运行的前端项目，并实现 Subscription 管理页面的完整功能（列表 / 新增 / 编辑 / 删除 / 刷新 / 错误处理 / 加载态 / 空态）。要求工程化、可维护、结构清晰，代码可直接运行。

## 0. 技术栈与约束

* 构建工具：Vite
* 框架：React 18 + TypeScript
* 路由：React Router v6
* 请求：fetch 封装（不要 axios，除非我明确要求）
* UI：默认使用 Ant Design（如果没有则用 Tailwind + Headless UI，但优先 Ant Design）
* 状态管理：React hooks（不要 Redux）
* 代码风格：ESLint + Prettier（给出配置）
* 目录结构：必须按本 Prompt 的规范生成
* 所有 API 仅允许 GET / POST
* POST 请求业务字段必须放 body，不允许放 query
* 前端不得在页面内直接写 fetch，必须通过 `src/api/*` 封装调用

---

## 1. API 约定（必须实现）

后端提供以下接口：

* GET `/api/v1/subscriptions`
* POST `/api/v1/subscriptions/create`
* POST `/api/v1/subscriptions/update`
* POST `/api/v1/subscriptions/delete`
* POST `/api/v1/subscriptions/refresh`

### 1.1 通用返回结构（前端必须兼容）

假设返回格式为：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

* `code === 0` 表示成功
* 非 0 表示失败，`message` 用于 toast 提示
* 若请求失败（网络错误/超时），toast 提示：“网络异常，请稍后再试”

> 你需要写一个通用 `request` 封装：自动解析 JSON、处理 code、抛错、统一 toast。

### 1.2 Subscription 数据结构（如无字段则按此定义）

```ts
export type SubscriptionStatus = "active" | "paused" | "error";

export interface Subscription {
  id: string;
  name: string;
  status: SubscriptionStatus;
  updatedAt: string; // ISO string
  createdAt: string; // ISO string
  remark?: string;
}
```

如果 GET 返回字段不同，允许做映射，但类型文件必须存在。

---

## 2. 页面与路由（必须实现）

路由：

* `/subscriptions` → SubscriptionList 页面（核心页面）
* `/` 自动重定向到 `/subscriptions`

页面必须实现：

1. Subscription 列表展示（Table）
2. 新建 Subscription（Modal 表单）
3. 编辑 Subscription（Modal 表单预填）
4. 删除 Subscription（Confirm 二次确认）
5. 刷新 Subscription（单行刷新按钮）
6. Loading Skeleton（首次加载）
7. Empty State（无数据）
8. Error State（接口失败时可重试）
9. Toast 提示（成功/失败）

---

## 3. UI 结构（必须按此布局）

SubscriptionList 页面结构：

* Header 区：

  * 标题：Subscriptions
  * 主按钮：New Subscription
  * 次按钮：Refresh List
* Body 区：

  * Table（列：Name / Status / UpdatedAt / Actions）
  * Actions（每行：Edit / Delete / Refresh）
* Footer 区：

  * Pagination（如果后端没有分页，也要预留结构与状态）

---

## 4. 交互行为（必须逐条实现）

### 4.1 页面加载

* 页面 mount 时调用 `GET /api/v1/subscriptions`
* 显示 skeleton
* 成功：渲染 table
* 失败：显示 ErrorState（含“重试”按钮 → 重新拉取）

### 4.2 Refresh List（刷新列表）

* 点击 Refresh List → 重新调用 GET
* Table 顶部显示 loading（不要全页闪屏）

### 4.3 New Subscription

* 点击 New Subscription → 打开 Modal
* 表单字段：

  * name（必填）
  * status（下拉，默认 active）
  * remark（可选）
* 提交：

  * 提交时按钮 loading 且禁用
  * POST `/create` body：{ name, status, remark }
  * 成功：toast 成功、关闭 modal、刷新列表
  * 失败：toast 显示错误 message

### 4.4 Edit

* 点击 Edit → 打开 Modal，预填该行数据
* 提交：

  * POST `/update` body：{ id, name, status, remark }
  * 成功：toast 成功、关闭 modal、刷新列表

### 4.5 Delete

* 点击 Delete → Confirm 弹窗二次确认
* Confirm 后：

  * POST `/delete` body：{ id }
  * 成功：toast 成功、刷新列表
  * 失败：toast

### 4.6 Refresh 单行刷新

* 点击单行 Refresh：

  * POST `/refresh` body：{ id }
  * 行内按钮 loading
  * 成功后更新该行（优先直接用返回 data 更新该行；若无返回则刷新列表）

---

## 5. 状态管理（必须实现且命名清晰）

SubscriptionList 页面 state（示例）：

```ts
const [list, setList] = useState<Subscription[]>([]);
const [loading, setLoading] = useState(false);
const [error, setError] = useState<string | null>(null);

const [modalOpen, setModalOpen] = useState(false);
const [modalMode, setModalMode] = useState<"create" | "edit">("create");
const [current, setCurrent] = useState<Subscription | null>(null);
const [submitting, setSubmitting] = useState(false);

const [rowRefreshingId, setRowRefreshingId] = useState<string | null>(null);
```

---

## 6. 工程目录结构（必须生成）

```
src/
  api/
    request.ts
    subscriptions.ts
  pages/
    Subscriptions/
      index.tsx
      components/
        SubscriptionTable.tsx
        SubscriptionModal.tsx
        ErrorState.tsx
        EmptyState.tsx
  router/
    index.tsx
  types/
    subscription.ts
  utils/
    datetime.ts
  App.tsx
  main.tsx
```

---

## 7. 代码规范要求（必须生成配置）

* ESLint + Prettier 配置文件齐全
* 提供 `npm run dev` 可运行
* 提供 `npm run lint` 可运行
* 每个组件必须是独立文件，禁止一个文件写完所有东西
* 必须有合理的注释（不要过多）

---

## 8. 组件实现要求（必须）

### SubscriptionTable.tsx

* props：

  * list
  * loading
  * onEdit(row)
  * onDelete(row)
  * onRefreshRow(row)
  * rowRefreshingId
* 使用 antd Table（或等价）实现

### SubscriptionModal.tsx

* props：

  * open
  * mode
  * initialValues（edit 模式传入）
  * submitting
  * onCancel
  * onSubmit(values)
* 使用 antd Modal + Form

### ErrorState.tsx

* 显示错误信息 + Retry 按钮

### EmptyState.tsx

* 无数据时显示空态

---

## 9. 细节与体验要求（必须满足）

* 所有 async 操作必须 try/catch
* 不允许 silent fail
* toast 提示：

  * create/update/delete/refresh 成功要 toast
  * 失败 toast 使用后端 message
* updatedAt 显示为可读格式（例如 `YYYY-MM-DD HH:mm:ss`）
* Status 用 Tag 渲染（active/paused/error）
* 删除按钮危险色
* Modal 关闭后表单要 reset
* 必须保证不会重复提交（按钮禁用）

---

## 10. 输出要求（必须）

你需要输出：

1. 完整项目文件树
2. 所有关键文件的代码内容（按文件分隔）
3. 如何启动项目（命令）
4. 如果你做了字段映射，说明映射逻辑

---

## 11. 扩展（可选但加分）

* 支持搜索框（按 name 过滤，前端过滤即可）
* 支持排序（按 updatedAt）
* 支持轮询刷新（可配置开关）

---

**请严格按以上规范生成代码，不要省略关键文件，不要仅给片段。要保证我复制到本地即可运行。**
