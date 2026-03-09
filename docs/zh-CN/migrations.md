# BoxPilot Migration 规范

[English](../migrations.md)

当前项目只有一个 migration 文件：`server/internal/store/migrations/0001_init.sql`。

后续 schema 变化应新增新版本文件，而不是继续改 `0001`。

## 原则

- 只向前升级
- 一个版本一个文件
- 启动时自动执行未应用 migration
- 失败则拒绝启动

## 命名

```text
0001_init.sql
0002_add_xxx.sql
0003_add_yyy.sql
```

## 当前 schema 范围

- subscriptions
- nodes
- runtime_state
- proxy_settings
- routing_settings
- forwarding_policy
- subscription_rule_sets
- subscription_rules
- subscription_group_members
- runtime_group_selections
