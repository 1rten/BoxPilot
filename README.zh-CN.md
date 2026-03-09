# BoxPilot

[English](./README.md)

BoxPilot 是一个面向 `sing-box` 的自托管控制面，用来管理订阅、节点、转发策略和运行时配置。

当前仓库已经不是早期 MVP 草稿，而是一个可运行的单仓库应用：

- 后端：Go + Gin + SQLite
- 前端：React 18 + Vite + Ant Design
- 运行方式：进程模式，BoxPilot 生成 `sing-box.json` 并通过 `SINGBOX_RESTART_CMD` 驱动重载

## 当前能力

- 订阅管理：新增、编辑、删除、手动刷新、自动刷新
- 订阅解析：传统 URI 列表、sing-box JSON、Clash YAML，以及它们的 base64 变体
- 节点管理：启用/停用、转发开关、批量操作、HTTP/PING 测试
- 运行时观测：状态、流量、连接、日志、代理链路检查
- 代理设置：HTTP / SOCKS5 监听地址、端口、认证
- 路由设置：私网绕过、自定义域名/CIDR 绕过
- 转发策略：健康筛选、延迟阈值、未测速节点策略、测速并发
- 业务分组：从订阅中的规则/规则集生成 `biz-*` 运行时分组，支持手动选择或自动探测
- 安全应用：预检查、原子写入、失败回滚、运行中自动防抖重载

## 页面结构

- `Dashboard`：运行状态、流量、连接、日志、路由摘要
- `Subscriptions`：订阅列表、自动刷新配置、同步状态
- `Nodes`：节点筛选、批量转发、连通性测试、详情抽屉
- `Settings`
  - `Access`：HTTP / SOCKS5 全局入口配置
  - `Routing`：绕过规则与转发策略
  - `Runtime`：`manual` 与 `biz-*` 分组切换

## 快速开始

### 方式一：预构建流程

```bash
git clone <repo-url>
cd BoxPilot
make up-prebuilt
```

打开 `http://localhost:8080`。

如需指定预构建架构：

```bash
make PREBUILT_GOARCH=amd64 up-prebuilt
```

### 方式二：标准 Docker 构建

```bash
docker compose up --build
```

只绑定本机：

```bash
export BIND_IP=127.0.0.1
docker compose up --build
```

默认会暴露：

- `8080`：Web UI + API
- `7890`：HTTP 代理
- `7891`：SOCKS5 代理

## 本地开发

### 后端

```bash
cd server
ADDR=:8080 \
DB_PATH=../data/app.db \
SINGBOX_CONFIG=../data/sing-box.json \
SINGBOX_RESTART_CMD='pkill -HUP sing-box' \
go run .
```

### 前端

```bash
cd web
npm ci
npm run dev
```

如果希望由后端直接提供前端静态资源，需要先构建前端，并设置：

```bash
WEB_ROOT=../web/dist
```

## 常用命令

```bash
make build
make build-prebuilt
make image-prebuilt
make up-prebuilt
make test
make migrate-gen
make diagnose
```

## 文档

- [English README](./README.md)
- [架构说明](/Users/1rten/Documents/workspace/BoxPilot/docs/zh-CN/architecture.md)
- [前端架构](/Users/1rten/Documents/workspace/BoxPilot/docs/zh-CN/frontend-architecture.md)
- [错误码](/Users/1rten/Documents/workspace/BoxPilot/docs/zh-CN/error-codes.md)
- [Migration 规范](/Users/1rten/Documents/workspace/BoxPilot/docs/zh-CN/migrations.md)
- [UI 说明](/Users/1rten/Documents/workspace/BoxPilot/docs/zh-CN/ui-design.md)

## License

MIT
