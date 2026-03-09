# BoxPilot 架构说明

[English](../architecture.md)

本文档描述的是当前仓库已经实现的架构，而不是早期设计草案。

## 1. 系统边界

BoxPilot 是 `sing-box` 的控制面，不实现代理协议栈本身。

它负责：

- 管理订阅与节点
- 解析订阅中的 outbounds、规则集、业务路由规则
- 持久化状态到 SQLite
- 生成运行时 `sing-box` 配置
- 通过外部命令执行预检查与重启
- 提供 Web UI 和 REST API

它不负责：

- 提供公共节点或托管订阅
- 代替 `sing-box` 的数据面能力
- 实现复杂的多实例编排

## 2. 总体架构

```text
Browser
  -> React SPA
  -> /api/v1/*

BoxPilot
  -> Gin router
  -> handlers
  -> service
  -> repo/store
  -> SQLite
  -> generator
  -> runtime.Check / runtime.Restart

sing-box
  -> 读取 BoxPilot 生成的配置
  -> 对外提供 HTTP / SOCKS5
  -> 可选暴露 Clash API 给 BoxPilot 做观测与自动探测
```

## 3. 后端分层

代码位于 `server/internal`。

- `api/`
- `service/`
- `store/`
- `parser/`
- `generator/`
- `runtime/`
- `util/`

## 4. 前端结构

代码位于 `web/src`。

- `App.tsx`
- `pages/`
- `hooks/`
- `api/`
- `components/common/`

## 5. 数据模型

当前 schema 由 `server/internal/store/migrations/0001_init.sql` 初始化。

核心表：

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

## 6. 运行模型

1. 刷新订阅并解析节点/规则
2. 生成 `sing-box` 运行时配置
3. 执行预检查
4. 写入正式配置
5. 重启运行时
6. 失败时回滚
