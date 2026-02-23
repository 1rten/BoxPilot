# BoxPilot 数据库 Migration 规范

版本：v0.1  
数据库：SQLite（modernc.org/sqlite）  
适用范围：BoxPilot 单实例部署（Docker / 本地运行）

---

## 1. 设计目标

### 1.1 为什么需要 Migration

即使是个人项目，也必须支持：

- 新版本新增字段
- 表结构调整
- 索引变更
- 数据迁移
- 未来扩展功能

如果没有 migration 机制：

- 用户升级版本可能直接启动失败
- 或 silent 破坏数据
- 或需要手动删库（不可接受）

---

## 2. 总体设计原则

### 2.1 启动时自动执行

- 应用启动时自动检查数据库版本
- 执行未运行的 migration
- 失败则拒绝启动

> 不允许“启动后懒执行”

### 2.2 顺序执行（严格单向）

- Migration 必须按版本号递增顺序执行
- 不允许跳过版本
- 不允许回滚（SQLite 单实例模式不做 down migration）

### 2.3 不支持自动 downgrade

BoxPilot v0.x 阶段：

- 仅支持向前升级
- 不支持降级数据库版本
- 如果用户降级版本 → 需手动处理

---

## 3. 目录结构规范

Migration 文件必须放在：

```
server/internal/store/migrations/
```

文件命名规范：

```
0001_init.sql
0002_runtime_state.sql
0003_add_scheduler_fields.sql
0004_xxx.sql
```

命名规则：

- 4 位数字前缀（递增）
- 下划线
- 描述性名称
- `.sql` 结尾

---

## 4. schema_migrations 表规范

必须存在以下表：

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL
);
```

说明：

| 字段 | 类型 | 说明 |
|------|------|------|
| version | INTEGER | migration 版本号 |
| applied_at | TEXT | 执行时间（RFC3339） |

---

## 5. Migration 执行流程（启动时）

启动流程：

1. 打开数据库
2. 创建 `schema_migrations`（如果不存在）
3. 读取 migrations 目录
4. 按 version 排序
5. 查询数据库已执行版本
6. 对比未执行的 migration
7. 按顺序执行
8. 每执行一个：
   - 开启事务
   - 执行 SQL
   - 插入 `schema_migrations`
   - 提交事务

如果任一步失败：

- 回滚事务
- 打印错误日志
- 退出进程

---

## 6. Migration 编写规则

### 6.1 每个 migration 必须满足

- 可重复执行（幂等）
- 使用 `IF NOT EXISTS`
- 不依赖外部状态
- 不包含应用逻辑

### 6.2 示例：0001_init.sql

```sql
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS subscriptions (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  url TEXT NOT NULL,
  type TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  refresh_interval_sec INTEGER NOT NULL DEFAULT 3600,
  etag TEXT NOT NULL DEFAULT '',
  last_modified TEXT NOT NULL DEFAULT '',
  last_fetch_at TEXT,
  last_success_at TEXT,
  last_error TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_enabled
ON subscriptions(enabled);

CREATE TABLE IF NOT EXISTS nodes (
  id TEXT PRIMARY KEY,
  sub_id TEXT NOT NULL,
  tag TEXT NOT NULL,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  outbound_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (sub_id) REFERENCES subscriptions(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_nodes_tag
ON nodes(tag);
```

### 6.3 示例：0002_runtime_state.sql

```sql
CREATE TABLE IF NOT EXISTS runtime_state (
  id TEXT PRIMARY KEY,
  config_version INTEGER NOT NULL DEFAULT 0,
  config_hash TEXT NOT NULL DEFAULT '',
  last_reload_at TEXT,
  last_reload_error TEXT
);

INSERT OR IGNORE INTO runtime_state (id, config_version, config_hash)
VALUES ('runtime', 0, '');
```

---

## 7. SQLite 特殊限制处理规范

SQLite 不支持：

- 删除列（DROP COLUMN）
- 修改列类型
- 修改约束

如需修改结构：必须使用“创建新表 + 数据迁移 + 替换表”方式。

### 7.1 结构变更标准流程

假设你要修改 `subscriptions` 表，步骤：

```sql
-- 1. 新建表
CREATE TABLE subscriptions_new (...);

-- 2. 复制数据
INSERT INTO subscriptions_new (fields...)
SELECT fields... FROM subscriptions;

-- 3. 删除旧表
DROP TABLE subscriptions;

-- 4. 重命名
ALTER TABLE subscriptions_new RENAME TO subscriptions;

-- 5. 重建索引
CREATE INDEX ...
```

必须在事务内完成。

---

## 8. 数据迁移规范（Data Migration）

如果新增字段需要默认值以外的数据迁移，必须写在 migration 文件中。

示例：

```sql
ALTER TABLE subscriptions ADD COLUMN user_agent TEXT DEFAULT '';

UPDATE subscriptions
SET user_agent = 'BoxPilot/0.2'
WHERE user_agent = '';
```

禁止：

- 在 Go 代码中偷偷执行结构性 SQL
- 在 handler/service 层做 schema 变更

---

## 9. 并发与锁规范

SQLite 在单实例 Docker 模式下：

- 不会出现多个实例同时 migration
- 仍建议 migration 在应用启动最早阶段执行

禁止：

- 在 HTTP 请求期间执行 migration
- 在 scheduler 中执行 migration

---

## 10. Migration 不可做的事情

❌ 不允许：

- 删除用户数据（除非重大版本且写清楚）
- 自动清空表
- 修改字段语义
- 执行非幂等操作
- 依赖网络
- 依赖运行时环境变量

---

## 11. 版本演进策略

### 11.1 v0.x 阶段

- 允许破坏性变更
- 但必须通过 migration 实现
- 不能让已有用户数据库无法启动

### 11.2 v1.0 之后

- 禁止破坏性 schema 变更
- 必须兼容旧数据
- 必须保证 upgrade 安全

---

## 12. 测试规范

建议：每次新增 migration，必须写单元测试：

- 空库升级
- 老版本升级
- 重复执行不报错

---

## 13. 回滚策略说明

当前阶段：

- 不支持 down migration
- 若 migration 执行失败：启动失败，用户可恢复数据库备份

未来可扩展：

- 支持 `--migrate-only`
- 支持 `--dry-run`

---

## 14. 生产环境注意事项

- 建议用户升级前备份 `/data/app.db`
- 重大版本升级在 Release Notes 中写清楚
- 若 migration 复杂（数据迁移耗时），可在 UI 中提示维护模式

---

## 15. 最佳实践总结

必须做到：

- 有 schema_migrations 表
- 启动时自动 migration
- 顺序执行
- 事务保护
- 幂等
- 不破坏数据

做到这些，BoxPilot 数据库演进就会非常稳。

---

## 16. 未来扩展方向

- 支持 migration 校验 hash
- 支持 migration 签名
- 支持 CLI 执行 migration
- 支持 read-only 检查模式
