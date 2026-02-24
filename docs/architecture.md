# BoxPilot 架构设计文档（单文件版）
> BoxPilot：自托管的 sing-box 控制面（Control Plane）  
> 版本：v0.1（初始设计）  
> 目标用户：个人自用（开源）  
> 前端：Vite + React（与后端同仓库）  
> 后端：Go + Gin + SQLite  
> 部署：Docker（生成镜像运行）

---

## 目录
- [1. 项目概述](#1-项目概述)
- [2. 系统架构](#2-系统架构)
- [3. 技术选型](#3-技术选型)
- [4. 仓库结构与构建策略](#4-仓库结构与构建策略)
- [5. 数据模型与数据库设计（SQLite）](#5-数据模型与数据库设计sqlite)
- [6. 核心业务流程设计](#6-核心业务流程设计)
- [7. sing-box 配置生成设计](#7-sing-box-配置生成设计)
- [8. Runtime 控制与重载策略](#8-runtime-控制与重载策略)
- [9. API 设计（供 React 调用）](#9-api-设计供-react-调用)
- [10. 前端设计（Vite + React）](#10-前端设计vite--react)
- [11. Docker 镜像与部署设计](#11-docker-镜像与部署设计)
- [12. 安全设计（个人自用）](#12-安全设计个人自用)
- [13. 可观测性与运维](#13-可观测性与运维)
- [14. 版本发布与开源约定](#14-版本发布与开源约定)
- [15. 迭代路线图](#15-迭代路线图)
- [附录 A：环境变量清单](#附录-a环境变量清单建议)
- [附录 B：非目标声明](#附录-b非目标声明避免误解)

---

## 1. 项目概述

### 1.1 背景与动机
BoxPilot 用于个人自托管场景：把订阅管理、节点持久化、配置生成、sing-box 重载这一整套流程做成一个简单、稳定的 Web 管理系统。

### 1.2 范围与边界
**BoxPilot 负责：**
- 管理 sing-box 订阅（HTTP 拉取、缓存）
- 解析订阅内容得到节点（outbounds）
- 持久化订阅与节点（SQLite）
- 生成运行时 sing-box 配置文件
- 控制 sing-box（容器方式）重载/重启
- 提供 Web UI（Vite + React）与 REST API

**BoxPilot 不负责：**
- 不实现代理协议栈（交给 sing-box）
- 不内置节点、不提供公共订阅
- 不做“服务器端按应用分流”（那应在客户端完成）

### 1.3 设计目标
- 部署简单：`docker compose up -d` 即可运行
- 可靠：原子写配置、失败可回滚、日志清晰
- 可扩展：未来可加入 urltest、分流规则、更多订阅格式
- 对个人自用友好：默认不暴露公网代理端口

---

## 2. 系统架构

### 2.1 控制面 / 数据面 分离
- **控制面（Control Plane）**：BoxPilot（Go API + React UI）
- **数据面（Data Plane）**：sing-box（官方镜像）


```
+---------------------------+
| Browser (Vite+React UI)   |
+-------------+-------------+
              |
              | HTTP (8080)
              v
+---------------------------+
| BoxPilot                  |
| Go(Gin) + SQLite + UI     |
| - Subscription mgmt       |
| - Parse sing-box sub      |
| - Generate config         |
| - Restart sing-box        |
+-------------+-------------+
              |
              | write /data/sing-box.json
              v
+---------------------------+
| sing-box                  |
| - http inbound :7890      |
| - socks inbound:7891      |
| - outbounds/routing       |
+---------------------------+
```


### 2.2 运行端口规划
| 组件 | 端口 | 说明 |
|---|---:|---|
| BoxPilot | 8080 | Web UI + REST API |
| sing-box | 7890 | HTTP Proxy（含 CONNECT，可代理 HTTPS 网站） |
| sing-box | 7891 | SOCKS5 Proxy |

> 推荐：7890/7891 默认只绑定 `127.0.0.1`（或仅内网 IP），避免变成开放代理。

**代理端口暴露策略（产品设计，避免误用）：**  
个人用最常见事故是用户把 7890/7891 映射到公网后被扫描滥用。建议在文档与 UI 中明确：
- 默认 compose 绑定 `127.0.0.1`。
- 若用户改成 `0.0.0.0`，UI 弹警告。
- 可选：后端通过 env 或配置判断“代理端口是否对公网开放”，在 Dashboard 显示警告 banner。  
（技术上无法完全判断真实网络暴露，但能在配置层提醒。）

---

## 3. 技术选型

### 3.1 后端（Go）
- Go 1.22+
- Gin（HTTP 框架）
- modernc.org/sqlite（纯 Go SQLite 驱动，无 CGO）
- 标准库 `net/http` 拉取订阅
- 通过 docker.sock 执行 `docker restart singbox`（MVP 最简单可靠）

### 3.2 前端（Vite + React）
- Vite + React 18 + TypeScript
- React Router（路由）
- Axios / fetch（API）
- 状态管理：Zustand（推荐）或 React Context（MVP）

### 3.3 存储与文件
- SQLite 文件：`/data/app.db`
- sing-box 运行配置：`/data/sing-box.json`
- 可选配置备份：`/data/backup/sing-box.<ts>.json`

---

## 4. 仓库结构与构建策略

### 4.1 Monorepo 目录结构（前后端同仓库）

```
boxpilot/
├── README.md
├── LICENSE
├── docker-compose.yml
├── Makefile
├── .gitignore
│
├── docs/
│   ├── architecture.md      # 本架构文档（可合并/拆分）
│   ├── api.openapi.yaml     # OpenAPI 契约
│   ├── error-codes.md       # 错误码说明
│   └── migrations.md        # migration 规范
│
├── server/
│   ├── main.go
│   ├── internal/
│   │   ├── api/
│   │   │   ├── router.go
│   │   │   ├── middleware/
│   │   │   │   ├── request_id.go
│   │   │   │   ├── recover.go
│   │   │   │   └── cors.go
│   │   │   ├── handlers/
│   │   │   │   ├── subscriptions.go
│   │   │   │   ├── nodes.go
│   │   │   │   ├── runtime.go
│   │   │   │   └── system.go
│   │   │   └── dto/
│   │   │       ├── subscription.go
│   │   │       ├── node.go
│   │   │       ├── runtime.go
│   │   │       └── common.go
│   │   ├── store/
│   │   │   ├── sqlite.go
│   │   │   ├── migrator.go
│   │   │   ├── migrations/
│   │   │   │   ├── 0001_init.sql
│   │   │   │   └── 0002_runtime_state.sql
│   │   │   └── repo/
│   │   │       ├── subscriptions.go
│   │   │       ├── nodes.go
│   │   │       └── runtime_state.go
│   │   ├── service/
│   │   │   ├── subscription_refresh.go
│   │   │   ├── config_build.go
│   │   │   ├── runtime_control.go
│   │   │   └── scheduler.go
│   │   ├── parser/
│   │   │   └── singbox.go
│   │   ├── generator/
│   │   │   └── singbox.go
│   │   ├── runtime/
│   │   │   ├── docker_restart.go
│   │   │   └── process_mode.go    # 预留：不挂 docker.sock 时使用
│   │   ├── util/
│   │   │   ├── atomic_write.go
│   │   │   ├── hash.go
│   │   │   ├── id.go
│   │   │   ├── time.go
│   │   │   └── errorx/
│   │   │       ├── codes.go
│   │   │       └── error.go
│   │   └── observability/
│   │       └── logger.go
│   ├── go.mod
│   └── Dockerfile
│
└── web/
    ├── src/
    │   ├── api/              # client、generated（OpenAPI 类型）
    │   ├── hooks/            # useSubscriptions、useNodes、useRuntime
    │   ├── pages/            # Dashboard、Subscriptions、Nodes
    │   ├── components/       # Table、Modal 等
    │   └── store/            # 仅 UI 状态（见 10.2/10.5）
    ├── public/
    ├── package.json
    └── vite.config.ts
```

**文档与契约（`docs/`）：**
- **architecture.md**：本文档，系统架构与设计说明。
- **api.openapi.yaml**：OpenAPI 3 契约，供前端/联调与代码生成使用。
- **error-codes.md**：API 错误码定义与说明。
- **migrations.md**：SQL 迁移规范（版本号、幂等、升级流程）。

**后端核心分层（`server/internal/`）：**
- **api**：HTTP 层；router、middleware（request_id、recover、cors）、handlers、dto。
- **store**：SQLite 连接、migrator、`migrations/*.sql`、repo（subscriptions、nodes、runtime_state）。
- **service**：订阅刷新、配置构建、runtime 控制、定时调度。
- **parser**：sing-box 订阅解析（object/array）。
- **generator**：sing-box 运行时 config 生成。
- **runtime**：执行重载（docker_restart / process_mode 可选）。
- **util**：原子写、hash、id、时间、统一错误码（errorx）。
- **observability**：结构化日志等。

### 4.2 构建策略（推荐：后端镜像内包含前端静态资源）
- `web/` 先 `vite build` 输出 `web/dist`
- 构建 Go 后端时把 `web/dist` 拷贝进镜像，并由 Go 提供静态资源服务
- 最终只需要两个容器：`boxpilot`（Go + UI）、`singbox`（官方镜像）

优点：部署最简单（少一个 web 容器）、UI/API 版本严格一致。

---

## 5. 数据模型与数据库设计（SQLite）

### 5.1 表：subscriptions
用途：订阅地址、缓存信息、刷新策略、最后状态。

| 字段 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | UUID |
| name | TEXT | 显示名 |
| url | TEXT | 订阅 URL |
| type | TEXT | 固定 `singbox`（预留扩展） |
| enabled | INTEGER | 0/1 |
| refresh_interval_sec | INTEGER | 刷新间隔（默认 3600） |
| etag | TEXT | HTTP 缓存 |
| last_modified | TEXT | HTTP 缓存 |
| last_fetch_at | TEXT | 上次拉取时间 |
| last_success_at | TEXT | 上次成功时间 |
| last_error | TEXT | 上次错误（摘要） |
| created_at | TEXT | RFC3339 |
| updated_at | TEXT | RFC3339 |
| user_agent | TEXT | 可选，拉取时 UA |
| headers_json | TEXT | 可选，key/value 如 Authorization、Cookie |
| timeout_sec | INTEGER | 可选，请求超时（秒） |
| max_size_kb | INTEGER | 可选，最大响应体（KB），防异常巨大订阅 |

索引建议：
- `idx_subscriptions_enabled(enabled)`

### 5.2 表：nodes
用途：解析订阅得到的节点 outbounds（原样保存 JSON）。

| 字段 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | UUID |
| sub_id | TEXT | 订阅 ID（FK） |
| tag | TEXT | sing-box outbound tag，**全局唯一**（见 6.2 规则） |
| name | TEXT | 展示名（默认 tag） |
| type | TEXT | outbound type（vless/vmess/ss/trojan/...） |
| enabled | INTEGER | 0/1 |
| outbound_json | TEXT | 原始 outbound JSON |
| created_at | TEXT | RFC3339 |

索引建议：
- `idx_nodes_sub_id(sub_id)`
- `idx_nodes_enabled(enabled)`
- **唯一索引**：`UNIQUE(nodes.tag)`（保证 tag 全局唯一，与 6.2 规则一致）

### 5.3 表：schema_migrations（DB 演进，强烈建议）

开源项目升级时最易踩坑的是 DB schema 变更。建议从一开始就做：

- 表名：`schema_migrations`（或通用名），记录已执行的 migration 版本。
- 每次启动执行 migrations：按版本号排序，只执行未执行过的 migration。
- 保证旧版本升级不丢数据、可重复执行（幂等）。

实现与规范：迁移文件放在 `server/internal/store/migrations/`（如 `0001_init.sql`、`0002_runtime_state.sql`）；**migration 编写与发布规范见 `docs/migrations.md`**。

### 5.4 表：runtime_state（单行即可）
用途：记录当前运行配置版本、hash、上次重载结果。

| 字段 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | 固定 `runtime` |
| config_version | INTEGER | 自增版本 |
| config_hash | TEXT | 配置 hash |
| last_reload_at | TEXT | 上次重载时间 |
| last_reload_error | TEXT | 上次重载错误 |

### 5.5 表：proxy_settings
用途：HTTP/SOCKS 代理设置（全局）。

| 字段 | 类型 | 说明 |
|---|---|---|
| proxy_type | TEXT PK | `http` / `socks` |
| enabled | INTEGER | 1/0 |
| listen_address | TEXT | `127.0.0.1` / `0.0.0.0` |
| port | INTEGER | 端口 |
| auth_mode | TEXT | `none` / `basic` |
| username | TEXT | 可选 |
| password | TEXT | 可选 |
| updated_at | TEXT | 更新时间 |

### 5.6 表：node_proxy_overrides
用途：节点级代理覆盖（用于 Forwarding）。

| 字段 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | UUID |
| node_id | TEXT | 节点 ID（FK） |
| proxy_type | TEXT | `http` / `socks` |
| enabled | INTEGER | 1/0 |
| port | INTEGER | 端口 |
| auth_mode | TEXT | `none` / `basic` |
| username | TEXT | 可选 |
| password | TEXT | 可选 |
| created_at | TEXT | RFC3339 |
| updated_at | TEXT | RFC3339 |

---

## 6. 核心业务流程设计

### 6.1 订阅刷新（自动/手动）
触发方式：
- 定时任务（Scheduler）
- UI 点击“刷新订阅”或“全量重载”

**订阅拉取“真实世界”兼容性（建议）：**  
订阅源常有：gzip 压缩、非 UTF-8（少见）、需要特定 UA、需要 headers（Authorization、Cookie）、301/302 重定向。建议 fetcher：
- 支持自动解压（Go 的 Transport 可处理 gzip，需确保 `Accept-Encoding` 允许）
- 支持每订阅配置：`user_agent`、`headers_json`、`timeout_sec`、`max_size_kb`（如 5MB 上限，避免内存被打爆）
- 跟随重定向（默认 `net/http` 会跟 301/302）

流程：
1. 读取 enabled subscriptions（含可选 user_agent、headers_json、timeout_sec、max_size_kb）
2. 对每个订阅发起 HTTP GET（受 6.3 并发与 SubLock 限制）
3. 带缓存头：
   - `If-None-Match: <etag>`
   - `If-Modified-Since: <last_modified>`
4. 处理响应：
   - 304：更新缓存元信息（可选），结束
   - 200：解析 body（JSON），写 nodes 表（Replace）
5. 全部订阅处理完后：
   - 读取所有 enabled nodes
   - 生成新的 sing-box 运行配置
   - 原子写入 `sing-box.json`
   - 重启/重载 sing-box
6. 写 runtime_state（version/hash/时间/错误）

**失败时可用性优先**：若本次刷新失败（拉取失败/解析失败），不要清空 nodes，保留上次成功结果（与 7.5 回滚策略一致）。

### 6.2 sing-box 订阅解析（必须支持）
输入支持两种常见形式：

- **形式 A：完整 config（object with outbounds）**
  ```json
  { "outbounds": [ ... ] }
  ```

- **形式 B：outbounds 数组**
  ```json
  [ { ... }, { ... } ]
  ```

**过滤策略（不当作“节点”入库）：**  
direct、block、dns、selector、urltest 等。

**节点 tag 全局唯一规则（关键）：**  
sing-box 的 selector outbounds 依赖 tag，tag 冲突会导致 selector 指向错误节点或配置校验失败。规则建议写死：

- **默认 tag 格式**：`<subShort>-<index>-<safeName>`
  - `subShort`：订阅 ID 短码
  - `index`：该订阅内节点序号
  - `safeName`：对节点名做一次规范化（空格/特殊字符替换为 `_` 或去掉）
- **冲突**：若与已有 tag 冲突，则加后缀 `-2`、`-3`……
- **持久化**：DB 中 `nodes.tag` 建**唯一索引**（或在生成/入库时保证唯一），便于生成时校验。

### 6.3 任务调度：互斥与并发控制（强烈建议）

个人用也会出现：UI 点击 reload 的同时 scheduler 在跑，两次“生成/写文件/重启”互相打架。建议：

- **互斥锁**：
  - **ReloadLock**：全局锁，同一时间只能有一个“生成配置 + 写文件 + 重启”流程。
  - **SubLock(sub_id)**：按订阅维度，同一订阅同一时间只能刷新一次。
- **并发策略**：
  - **拉订阅**：可并发，**最多 3～5** 个并发，避免把订阅源打爆。
  - **生成配置与重启**：必须串行，且受 ReloadLock 保护。

实现方式：`sync.Mutex` 或更细粒度锁均可。

---

## 7. sing-box 配置生成设计

### 7.1 目标：生成运行时 sing-box.json

核心内容：
- **inbounds**：HTTP + SOCKS
- **outbounds**：direct/block + 节点 outbounds + selector(proxy)
- **route**：final=proxy（MVP）

### 7.2 Inbounds（可配置）

- HTTP inbound：listen/address/port 可配置（默认 0.0.0.0:7890）
- SOCKS inbound：listen/address/port 可配置（默认 0.0.0.0:7891）
- 可选 Basic 认证（HTTP/SOCKS）
- 建议开启 sniff（提升体验）

示例（片段）：

```json
{
  "inbounds": [
    {
      "type": "http",
      "tag": "http-in",
      "listen": "0.0.0.0",
      "listen_port": 7890,
      "sniff": true
    },
    {
      "type": "socks",
      "tag": "socks-in",
      "listen": "0.0.0.0",
      "listen_port": 7891,
      "sniff": true
    }
  ]
}
```

### 7.3 Outbounds（生成）

- **固定**：direct、block
- **节点**：DB 中 enabled nodes 的 `outbound_json`（反序列化后写回）
- **selector**：tag: `proxy`，outbounds: 所有节点 tag 列表

示例（片段）：

```json
{
  "outbounds": [
    { "type": "direct", "tag": "direct" },
    { "type": "block",  "tag": "block"  },

    { "type": "vless", "tag": "hk-1", "...": "..." },
    { "type": "ss",    "tag": "jp-1", "...": "..." },

    { "type": "selector", "tag": "proxy", "outbounds": ["hk-1", "jp-1"] }
  ]
}
```

### 7.4 Route（MVP）

默认全走 proxy：

```json
{
  "route": { "final": "proxy" }
}
```

后续扩展：geosite/geoip 分流、urltest 自动选优、自定义规则编辑器。

### 7.5 原子写入与备份（强烈建议）

配置写入必须**原子 + 校验**，步骤如下：

1. **生成**：在内存中生成完整 config。
2. **校验**：`json.Unmarshal` 校验结构可解析（已有）。
3. **可选预检**：在重启前执行 `sing-box check -c /data/sing-box.json.tmp`，在“订阅更新导致配置不合法”时拦住问题，避免 sing-box 起不来。  
   - 实现方式：在 boxpilot 容器内同装 sing-box 二进制，或临时启动 sing-box 容器执行 check。
4. **写临时文件**：`/data/sing-box.json.tmp`。
5. **备份旧配置**：`/data/backup/sing-box.<timestamp>.json`，**保留最近 N 份（建议默认 10）**。
6. **rename** 覆盖到 `/data/sing-box.json`。

**回滚策略（可用性优先）：**
- 若**本次刷新失败**（拉取失败 / 解析失败）：**不要清空 nodes**，保留上次成功结果。
- 若**写入失败**：不触发重启。
- 若**重启失败**：回滚到上一个备份并再次重启。

---

## 8. Runtime 控制与重载策略

### 8.1 MVP：docker restart（推荐）

- boxpilot 容器挂载 `/var/run/docker.sock`
- 执行：`docker restart singbox`

**优点：** 实现简单、行为稳定  
**缺点：** 短暂断连（个人自用通常可接受）

### 8.2 健康检查（可选）

- 检查 singbox 端口连通（7890/7891）
- 检查 singbox 容器状态（`docker inspect`）

### 8.3 sing-box 控制模式（可配置，建议两种）

开源用户环境不同，建议做成可配置：

| 模式 | 说明 | 适用场景 |
|------|------|----------|
| **Mode A：Docker**（默认） | 依赖 `docker.sock`，执行 `docker restart <container>` | 标准 Docker 部署 |
| **Mode B：Process** | sing-box 与 boxpilot 同机/同容器以进程运行；boxpilot 负责启动/stop/reload | 不想给 docker.sock 权限、NAS/群晖/软路由等 |

- **配置项**：`RUNTIME_MODE=docker|process`
- Docker 模式需要：`SINGBOX_CONTAINER`
- Process 模式需要：`SINGBOX_BIN=/usr/bin/sing-box`（boxpilot 调用该二进制完成 check/run）

docker.sock 权限较大，部分用户不愿挂载；提供 process mode 可提高项目接受度。

---

## 9. API 设计（供 React 调用）

### 9.1 约定

- **契约与错误码**：接口定义以 `docs/api.openapi.yaml`（OpenAPI 3.0.3）为准；错误码说明见 `docs/error-codes.md`。
- **风格**：RPC 风格，仅使用 **GET / POST**（无 PATCH/DELETE 等），所有写操作均为 POST，资源 id 放在 body 中。
- **Base path**：`/api/v1`（所有 API 路径以此为前缀；健康检查为 `/healthz`）。
- **成功响应**：统一包一层 `{ data: ... }`（单资源或列表的 `data` 数组）；删除成功为 `{ success: true }`。
- **错误响应**：`{ error: { code: string, message: string, details?: object } }`，HTTP 4xx/5xx；错误码定义见 error-codes.md。
- **常用状态码**：业务/校验错误 400、未找到 404、服务错误 500；创建/更新/删除成功均为 200。

### 9.2 System

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/healthz` | 存活探针，返回 `text/plain` `ok` |

### 9.3 Subscriptions

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/subscriptions` | 列表，200 返回 `{ data: Subscription[] }` |
| POST | `/api/v1/subscriptions/create` | 创建，body 见 `CreateSubscriptionRequest`（必填 `url`），200 返回 `{ data: Subscription }` |
| POST | `/api/v1/subscriptions/update` | 更新，body 见 `UpdateSubscriptionRequest`（必填 `id`，可选 name、enabled、refresh_interval_sec），200 返回 `{ data: Subscription }` |
| POST | `/api/v1/subscriptions/delete` | 删除，body `{ id: string }`，200 返回 `{ success: true }` |
| POST | `/api/v1/subscriptions/refresh` | 刷新该订阅（拉取+解析+替换 nodes），body `{ id: string }`，200 返回 `RefreshSubscriptionResponse`（sub_id、not_modified、nodes_total、nodes_enabled、fetched_at） |

### 9.4 Nodes

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/nodes` | 列表，query：`enabled`(0/1)、`sub_id`、`q`，200 返回 `{ data: Node[] }` |
| POST | `/api/v1/nodes/update` | 更新，body 见 `UpdateNodeRequest`（必填 `id`，可选 name、enabled），200 返回 `{ data: Node }` |

### 9.5 Runtime

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/runtime/status` | 运行时状态，200 返回 `RuntimeStatusResponse`（config_version、config_hash、last_reload_at/error、ports.http/socks、runtime_mode、singbox_container） |
| POST | `/api/v1/runtime/plan` | **Dry Run**：不落盘、不重启；body 可选 `RuntimePlanRequest`（include_disabled_nodes），200 返回 nodes_included、tags、config_hash |
| POST | `/api/v1/runtime/reload` | 写盘并重启 sing-box；body 可选 `RuntimeReloadRequest`（force_restart，默认 true），200 返回 config_version、config_hash、nodes_included、restart_output、reloaded_at |
| GET | `/api/v1/settings/proxy` | 代理设置（HTTP/SOCKS）与状态，返回 `ProxySettingsResponse` |
| POST | `/api/v1/settings/proxy/update` | 保存代理设置，返回 `ProxySettingsResponse` |
| POST | `/api/v1/settings/proxy/apply` | 写盘并重启 sing-box，使代理配置生效，返回 `ProxyApplyResponse` |
| GET | `/api/v1/nodes/forwarding` | 节点代理信息（global/override），query: `node_id` |
| POST | `/api/v1/nodes/forwarding/update` | 保存节点代理 override（或 `use_global=true` 清除 override） |
| POST | `/api/v1/nodes/forwarding/restart` | 重启该节点代理（MVP 触发整体重启） |

### 9.6 Health

- `GET /healthz`：进程存活（Liveness），返回 `text/plain` `ok`。

### 9.7 节点健康检查/测速（可选，建议后期）

测速实用但易复杂化，建议后期做：

- **TCP connect 测试**：轻量。
- **URL test**：通过 sing-box 路由到节点做 HTTP GET，更真实。
- 接口建议：`POST /api/v1/nodes/test`（并发限制 + 超时 + 结果缓存）。
- UI：展示延迟/失败率。

---

个人自用建议通过 docker 端口绑定把 UI/API 也限制在本机或内网，无需复杂鉴权。

---

## 10. 前端设计（Vite + React）

> **详细规范**：前端分层、类型生成、错误处理、状态管理、目录与构建等见 **`docs/frontend-architecture.md`**（前端架构规范文档）。本节与主架构保持一致，仅保留与整体系统相关的要点。

### 10.1 页面信息架构（MVP）

| 页面 | 内容 |
|------|------|
| **Dashboard** | singbox 状态、当前节点数量、上次重载、按钮：Reload（可选 Plan 预览）；若检测到代理端口对公网开放则显示警告 banner |
| **Subscriptions** | 列表、添加、启用/禁用、手动 refresh |
| **Nodes** | 列表、搜索、启用/禁用、查看 outbound 详情（折叠） |
| **Settings**（可选） | 端口显示、刷新间隔默认值、备份保留数量 |

### 10.2 前端分层结构（必须明确）

逻辑必须按层划分，避免业务散落在组件里。**自上而下**：

```
UI Components（展示与交互）
        ↓
Page Layer（页面组合、路由）
        ↓
Hooks（useSubscriptions / useNodes / useRuntime）
        ↓
API Client（基于 OpenAPI 生成或封装）
        ↓
HTTP layer（axios/fetch 实例）
```

**推荐目录结构**（`web/src/`）：

```
web/src/
├── api/
│   ├── client.ts           # axios 实例、baseURL、拦截器
│   └── generated/          # OpenAPI 自动生成的类型与 client（见 10.3）
├── hooks/
│   ├── useSubscriptions.ts
│   ├── useNodes.ts
│   └── useRuntime.ts
├── pages/
│   ├── Dashboard.tsx
│   ├── Subscriptions.tsx
│   └── Nodes.tsx
├── components/
│   ├── Table.tsx
│   ├── Modal.tsx
│   └── ...
└── store/
    └── appStore.ts         # 仅 UI 状态，见 10.5
```

- **Page** 只负责布局与组合，从 hooks 取数据、调方法。
- **Hooks** 封装服务端状态（React Query）与 API 调用，不直接在各组件里写 fetch。
- **API Client** 统一出口，类型来自 OpenAPI 生成，不手写请求/响应类型。

### 10.3 API 类型必须自动生成（强烈建议）

已有 OpenAPI 契约（`docs/api.openapi.yaml`），**不允许手写 TS 类型**。

- **生成命令**（示例）：
  ```bash
  openapi-typescript docs/api.openapi.yaml -o src/api/generated/types.ts
  ```
- **使用方式**：从生成文件中引用，例如
  ```ts
  type Subscription = components["schemas"]["Subscription"];
  ```
- **要求**：
  - 所有请求/响应类型、枚举均来自生成文件。
  - API 变更后重新生成并修复编译错误，避免前后端字段不一致或升级时静默破坏。

可选：使用 openapi-fetch 或类似工具生成 client，或手写基于 axios 的封装，但类型必须 100% 来自 OpenAPI 生成。

### 10.4 前端错误处理策略（必须明确）

后端统一返回 Error Envelope（见 `docs/error-codes.md`）：

```json
{
  "error": {
    "code": "SUB_FETCH_FAILED",
    "message": "subscription fetch failed",
    "details": { ... }
  }
}
```

**前端必须统一处理**：

1. **所有 API 失败统一拦截**（axios 响应拦截器或 fetch 封装）：解析 `error.code`、`error.message`、`error.details`。
2. **默认展示**：用 `error.message` 做 Toast 或页面 inline 提示。
3. **按 code 做特殊行为**（建议下表，可扩展）：

| 错误码 | 前端行为 |
|--------|----------|
| JOB_RELOAD_IN_PROGRESS | Toast：Reload in progress，可选禁用 Reload 按钮 |
| JOB_REFRESH_IN_PROGRESS | Toast：Refresh in progress，可选禁用该订阅的刷新按钮 |
| SUB_NOT_FOUND | Toast 提示，并刷新订阅列表 |
| NODE_NOT_FOUND | Toast 提示，并刷新节点列表 |
| RT_RESTART_FAILED | Toast + 可选弹出日志/详情（展示 details.output 截断） |
| REQ_VALIDATION_FAILED | 表单字段高亮（用 details.field / reason） |
| RT_DOCKER_SOCK_UNAVAILABLE / RT_SINGBOX_CONTAINER_NOT_FOUND | 明确提示“运行环境不可用”，引导检查部署 |
| 其他 | 仅展示 message，必要时展示 details |

禁止在业务组件里到处 `try/catch` 后各写各的提示；统一在 API 层或全局错误处理里根据 code 分支。

### 10.5 状态管理策略

**不做**：不用 Redux，不引入复杂全局 store。

**建议**：严格区分**服务器状态**与**UI 状态**。

| 类型 | 归属 | 工具 |
|------|------|------|
| subscriptions 列表 | 服务器状态 | React Query（TanStack Query） |
| nodes 列表 | 服务器状态 | React Query |
| runtime status | 服务器状态 | React Query |
| 当前弹窗开关、侧栏折叠、表单 draft | UI 状态 | Zustand（或 useState 局部） |

- **服务器状态**：通过 hooks（useSubscriptions、useNodes、useRuntime）封装，内部用 React Query 的 useQuery/useMutation，缓存与失效策略由 React Query 管理。
- **UI 状态**：仅存与后端无关的界面状态；不把“从 API 刚拿到的列表”塞进 Zustand，避免双源真相。

这样可避免“不知道数据该放哪、重复请求、升级后状态乱”等问题。

### 10.6 前端环境与构建策略（非常重要）

**开发模式**：

- 使用 Vite dev server。
- **必须配置 proxy**：将 `/api` 代理到后端，避免跨域。例如在 `vite.config.ts` 中：
  ```ts
  proxy: { '/api': { target: 'http://localhost:8080', changeOrigin: true } }
  ```
- 前端访问同源（如 `http://localhost:5173`），请求 `/api/v1/...` 被代理到 `http://localhost:8080/api/v1/...`。
- 可选：通过 `VITE_API_BASE` 在开发时覆盖 API 根路径（生产一般不设，见下）。

**生产模式**：

- `vite build` 输出 `web/dist`。
- 由 Go 静态托管：同一域名、同一端口，**不跨域**；base path 为 `/`。
- 生产环境**不需要**配置 API base（请求相对路径 `/api/v1/...` 即可）。

**环境变量**：

- `VITE_API_BASE`：仅开发时可选使用；生产留空或同域，保证请求发往同一 host。

**静态资源与缓存**（由 Go 提供静态文件时）：

- `/assets/*`：`Cache-Control: public, max-age=31536000, immutable`
- `index.html`：`Cache-Control: no-cache`（或 no-store），避免发版后用户仍拿到旧 HTML。

事先在架构里约定上述 proxy 与同域部署，可避免后续出现跨域、baseURL 不一致等问题。

### 10.7 UI 与运行时解耦原则（非常重要）

**架构上必须明确**：

- 前端**不直接操作** Docker（不调 docker 命令、不访问 docker.sock）。
- 前端**不直接读写**服务器文件系统（不访问 `/data`、不写 sing-box 配置）。
- **所有**与订阅、节点、配置、重载相关的操作**仅通过 API** 完成。

因此：

- UI 不假设“某字段一定存在”或“某接口一定返回某结构”——以 OpenAPI 与生成类型为准；缺失时用可选链或默认展示。
- 不在前端写死“容器名、路径、端口”等运行时细节；这些由后端与 env 决定，前端只消费 API 返回的 status/ports 等。

这样可避免“前端与部署/运行时隐性耦合”，便于单测与后续多环境部署。

---

## 11. Docker 镜像与部署设计

### 11.1 运行形态

- **容器 A**：boxpilot（Go + UI，含 SQLite 文件）
- **容器 B**：singbox（官方镜像）
- **共享 volume**：`./data:/data`

### 11.2 docker-compose（建议默认本机绑定）

```yaml
version: "3.9"
services:
  boxpilot:
    image: ghcr.io/<you>/boxpilot:latest
    ports:
      - "127.0.0.1:8080:8080"
    environment:
      - DATA_DIR=/data
      - DB_PATH=/data/app.db
      - SINGBOX_CONFIG=/data/sing-box.json
      - SINGBOX_CONTAINER=singbox
      - HTTP_PROXY_PORT=7890
      - SOCKS_PROXY_PORT=7891
    volumes:
      - ./data:/data
      - /var/run/docker.sock:/var/run/docker.sock
    depends_on:
      - singbox

  singbox:
    image: ghcr.io/sagernet/sing-box:latest
    container_name: singbox
    volumes:
      - ./data:/data
    command: ["run", "-c", "/data/sing-box.json"]
    ports:
      - "127.0.0.1:7890:7890"
      - "127.0.0.1:7891:7891"
```

### 11.3 BoxPilot 镜像构建（多阶段）

- **Stage 1**：Node 构建前端（`vite build`）
- **Stage 2**：Go 编译后端
- **Stage 3**：最终镜像仅包含 Go 二进制（与静态资源）

---

## 12. 安全设计（个人自用）

### 12.1 默认安全策略（推荐）

- API/UI 端口：只监听 127.0.0.1:8080（或只暴露内网）
- Proxy 端口：只监听 127.0.0.1:7890/7891（或只暴露内网）
- 不在仓库中提供任何节点/订阅示例（避免敏感）
- **代理端口暴露**：默认 compose 绑定 127.0.0.1；若用户改为 0.0.0.0，文档与 UI 需明确警告（见 2.2）。

### 12.2 可选增强

- sing-box inbound 用户名/密码（SOCKS/HTTP 支持）
- 反向代理（Caddy/Nginx）做 BasicAuth + TLS（若需要远程访问 UI）

---

## 13. 可观测性与运维

### 13.1 日志

- BoxPilot：结构化日志（JSON 或 key-value）
- 关键日志事件：subscription fetch start/end（耗时、状态码）、parse 结果（节点数、过滤数）、config build（hash/version）、singbox restart（stdout/stderr）
- UI 可做简易日志页（后续可 SSE）

### 13.2 健康检查

- `/healthz`：进程存活
- 可选 `/readyz`：DB 可写、config 目录可写、docker.sock 可用

### 13.3 故障恢复

- 配置写坏：依靠备份与回滚
- 订阅异常：保留上次成功 nodes 与 config，不因一次失败清空

---

## 14. 版本发布与开源约定

### 14.1 版本规范

SemVer：MAJOR.MINOR.PATCH。

| 类型 | 说明 |
|------|------|
| MINOR | 新增功能（兼容） |
| PATCH | 修复与小改动 |
| MAJOR | 破坏性变更 |

### 14.2 Release 产物（建议）

- Docker 镜像：`ghcr.io/<you>/boxpilot:<version>`
- Release Notes：新增、修复、升级注意事项
- 提供示例 docker-compose.yml

### 14.3 License

推荐：MIT 或 Apache-2.0（个人项目常用）。  
文档强调：BoxPilot 为通用网络代理控制工具，不提供任何节点/订阅。

---

## 15. 迭代路线图

### Phase 1（MVP，可运行）
- SQLite migrations（含 schema_migrations 机制）
- Subscriptions CRUD
- Fetch + ETag/Last-Modified
- Parse sing-box subscription（object/array）
- Nodes 入库（Replace）
- 生成 sing-box config（http+socks+selector）
- 原子写 config + 备份
- docker restart singbox
- React UI：Dashboard / Subscriptions / Nodes / Reload

### Phase 2（体验增强）
- 节点搜索/过滤/批量启用禁用
- selector 选择“当前默认节点”（记住选择）
- 手动刷新单个订阅与全量刷新区分
- 失败重试与退避
- Dry Run / Preview（POST /api/v1/runtime/plan）
- 任务调度互斥与并发控制（ReloadLock、SubLock）
- 订阅拉取兼容（user_agent、headers、timeout、max_size）
- 代理端口暴露策略与 UI 警告
- 静态资源缓存头

### Phase 3（高级能力，可选）
- urltest 自动选优
- 分流规则编辑（域名/geo）
- SSE/WS 推送任务与日志
- 多 profile/多策略组
- 节点健康检查/测速（POST /api/v1/nodes/test，可选）

---

## 附录 A：环境变量清单（建议）

| 变量 | 默认值 | 说明 |
|------|--------|------|
| DATA_DIR | /data | 数据目录 |
| DB_PATH | /data/app.db | SQLite 路径 |
| SINGBOX_CONFIG | /data/sing-box.json | sing-box 配置文件 |
| SINGBOX_CONTAINER | singbox | 容器名（Docker 模式） |
| RUNTIME_MODE | docker | `docker` 或 `process`（见 8.3） |
| SINGBOX_BIN | （无） | Process 模式下 sing-box 二进制路径，如 `/usr/bin/sing-box` |
| HTTP_PROXY_PORT | 7890 | http 入站端口 |
| SOCKS_PROXY_PORT | 7891 | socks 入站端口 |
| BACKUP_KEEP | 10 | 配置备份保留数量（建议默认 10） |

---

## 附录 B：非目标声明（避免误解）

BoxPilot 不内置节点、不提供订阅源

BoxPilot 不鼓励、也不提供任何绕过网络管理的专用策略模板

BoxPilot 仅提供配置管理与运行控制能力，具体使用场景由用户自行负责。
