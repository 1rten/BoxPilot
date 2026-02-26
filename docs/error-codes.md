# BoxPilot 错误码规范（Error Code Spec）

版本：v0.1  
适用范围：BoxPilot API `/api/v1/*`（GET/POST only）

---

## 1. 目标与原则

### 1.1 目标

- **前后端一致**：React UI 可根据 `code` 做稳定的提示、重试、跳转等逻辑。
- **可观测**：每个错误都能定位到一个明确的失败点（fetch/parse/db/runtime…）。
- **可扩展**：新增功能时只需新增错误码，不破坏既有含义。

### 1.2 设计原则

- `code` 是稳定标识，**不会因为文案改动而变化**。
- `message` 面向人类阅读，可用于 UI 直接展示（默认英文或中英皆可）。
- `details` 面向调试，**允许包含更具体信息**（状态码、URL、字段名、内部异常摘要等）。
- 不在 `message/details` 输出敏感信息（订阅 token、节点密钥等）。

---

## 2. 统一错误响应格式（Error Envelope）

所有错误必须使用统一结构：

```json
{
  "error": {
    "code": "SUB_FETCH_FAILED",
    "message": "subscription fetch failed",
    "details": {
      "sub_id": "xxxx",
      "status": 502,
      "timeout_sec": 20
    }
  }
}
```

**字段说明：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| error.code | string | ✅ | 稳定错误码（本规范定义） |
| error.message | string | ✅ | 面向用户/开发者的简短描述 |
| error.details | object | ❌ | 调试信息（可选，避免敏感） |

---

## 3. HTTP 状态码建议

虽然 API 只用 GET/POST，但仍建议返回合理 HTTP 状态码，便于调试与通用工具兼容：

| 场景 | 建议状态码 |
|------|------------|
| 参数错误/校验失败 | 400 |
| 未找到资源 | 404 |
| 资源冲突（并发 reload、tag 冲突等） | 409 |
| 上游订阅源不可用/超时 | 502 |
| 内部错误/未知异常 | 500 |
| 服务不可用（依赖缺失，如 sing-box 运行环境不可用） | 503 |

也可以选择所有错误都返回 200，但不推荐：会让调试和代理/网关日志更难看懂。

---

## 4. 错误码命名规则

- 全大写 + 下划线：`SUB_FETCH_FAILED`
- 前缀代表领域：
  - **REQ_** 请求/参数
  - **DB_** 数据库
  - **SUB_** 订阅
  - **NODE_** 节点
  - **CFG_** 配置生成/写入/校验
  - **RT_** runtime（sing-box 重启/模式）
  - **JOB_** 调度/任务并发

---

## 5. 错误码清单（按领域分类）

### 5.1 请求与参数（REQ_*）

| code | 建议状态码 | message（建议） | details 建议字段 |
|------|------------|-----------------|------------------|
| REQ_BAD_REQUEST | 400 | bad request | hint, raw |
| REQ_VALIDATION_FAILED | 400 | validation failed | field, reason（可数组） |
| REQ_MISSING_FIELD | 400 | required field missing | field |
| REQ_INVALID_FIELD | 400 | invalid field | field, reason, value |
| REQ_UNSUPPORTED_OPERATION | 400 | unsupported operation | op |
| REQ_TOO_LARGE | 413 | payload too large | max_bytes |

BoxPilot 个人版通常无需鉴权错误码；若未来加 token，可再增加 AUTH_*。

### 5.2 数据库与存储（DB_*）

| code | 建议状态码 | message（建议） | details 建议字段 |
|------|------------|-----------------|------------------|
| DB_ERROR | 500 | database error | op（query/exec/tx）, table |
| DB_MIGRATION_FAILED | 500 | database migration failed | version, sql(可省略), err(简短) |
| DB_CONSTRAINT_VIOLATION | 409 | database constraint violation | constraint, table, field |
| DB_NOT_FOUND | 404 | record not found | table, id |
| DB_TX_FAILED | 500 | database transaction failed | op, err |

### 5.3 订阅（SUB_*）

| code | 建议状态码 | message（建议） | details 建议字段 |
|------|------------|-----------------|------------------|
| SUB_NOT_FOUND | 404 | subscription not found | id |
| SUB_DISABLED | 409 | subscription is disabled | id |
| SUB_INVALID_URL | 400 | invalid subscription url | url（可脱敏） |
| SUB_FETCH_FAILED | 502 | subscription fetch failed | id, status, timeout_sec |
| SUB_FETCH_TIMEOUT | 502 | subscription fetch timeout | id, timeout_sec |
| SUB_HTTP_STATUS_ERROR | 502 | subscription returned error status | id, status |
| SUB_RESPONSE_TOO_LARGE | 413 | subscription response too large | id, max_bytes |
| SUB_NOT_MODIFIED | 200/304* | subscription not modified | id |
| SUB_PARSE_FAILED | 400 | subscription parse failed | id, format |
| SUB_FORMAT_UNSUPPORTED | 400 | subscription format unsupported | id, hint |
| SUB_EMPTY_OUTBOUNDS | 400 | no usable outbounds found | id |
| SUB_REPLACE_NODES_FAILED | 500 | failed to replace nodes for subscription | id |

\* 注：接口可以对 not modified 返回 200（不是 error），或用 `not_modified=true` 字段表达；一般不建议把它当错误。

### 5.4 节点（NODE_*）

| code | 建议状态码 | message（建议） | details 建议字段 |
|------|------------|-----------------|------------------|
| NODE_NOT_FOUND | 404 | node not found | id |
| NODE_TAG_CONFLICT | 409 | node tag conflict | tag, sub_id |
| NODE_INVALID_OUTBOUND | 400 | invalid outbound json | node_id(可选), reason |
| NODE_UPDATE_FAILED | 500 | failed to update node | id |
| NODE_LIST_FAILED | 500 | failed to list nodes | filters |

### 5.5 配置生成/写入（CFG_*）

| code | 建议状态码 | message（建议） | details 建议字段 |
|------|------------|-----------------|------------------|
| CFG_BUILD_FAILED | 500 | failed to build sing-box config | reason, nodes_included |
| CFG_NO_ENABLED_NODES | 409 | no enabled nodes to build config | enabled_nodes |
| CFG_JSON_INVALID | 500 | generated config is invalid json | reason |
| CFG_WRITE_FAILED | 500 | failed to write config file | path |
| CFG_BACKUP_FAILED | 500 | failed to backup config file | path |
| CFG_ROLLBACK_FAILED | 500 | failed to rollback config file | path |
| CFG_CHECK_FAILED | 500 | sing-box config check failed | output(截断) |

### 5.6 Runtime（RT_*）

| code | 建议状态码 | message（建议） | details 建议字段 |
|------|------------|-----------------|------------------|
| RT_RESTART_FAILED | 500 | failed to restart sing-box | cmd, output(截断) |
| RT_START_FAILED | 500 | failed to start sing-box | output(截断) |
| RT_STOP_FAILED | 500 | failed to stop sing-box | output(截断) |
| RT_STATUS_FAILED | 500 | failed to read runtime status | reason |

### 5.7 任务调度与并发（JOB_*）

| code | 建议状态码 | message（建议） | details 建议字段 |
|------|------------|-----------------|------------------|
| JOB_RELOAD_IN_PROGRESS | 409 | reload in progress | started_at |
| JOB_REFRESH_IN_PROGRESS | 409 | refresh in progress | sub_id |
| JOB_SCHEDULER_FAILED | 500 | scheduler failed | reason |
| JOB_RATE_LIMITED | 429 | too many requests | retry_after_sec |

### 5.8 未分类/兜底

| code | 建议状态码 | message（建议） | details 建议字段 |
|------|------------|-----------------|------------------|
| INTERNAL_ERROR | 500 | internal error | req_id |
| NOT_IMPLEMENTED | 501 | not implemented | feature |

---

## 6. details 字段规范（建议）

### 6.1 通用字段（建议每次尽量带上）

- **req_id**：string（由中间件生成并回写到响应头 X-Request-ID）
- **op**：string（操作名：sub_refresh、runtime_reload、cfg_build）
- **duration_ms**：number（耗时）

### 6.2 截断规则

- **output**（命令输出）建议截断：最多 2KB
- 错误堆栈不要直接返回给客户端（写日志即可）

### 6.3 脱敏规则

- subscription URL 可能包含 token：在 details 中只保留 host + path 前缀，或用 `url_masked`
- 节点 outbound_json 不要回传

---

## 7. API 端建议实现方式

### 7.1 统一错误构造器（建议）

在 Go 中实现一个 errorx 包：

- `errorx.New(code, message)`
- `WithDetails(map[string]any)`
- `HTTPStatus()`（可选，根据 code 映射）
- 并提供统一输出函数：`api.WriteError(c, errx)`

### 7.2 code → HTTP status 映射（建议）

| 前缀/模式 | HTTP 状态 |
|-----------|-----------|
| REQ_* | 400 |
| *_NOT_FOUND | 404 |
| *_CONFLICT / *_IN_PROGRESS | 409 |
| SUB_FETCH_* / SUB_HTTP_STATUS_ERROR | 502 |
| 其他 | 500 |

---

## 8. 示例

### 8.1 订阅 URL 不合法

```json
{
  "error": {
    "code": "SUB_INVALID_URL",
    "message": "invalid subscription url",
    "details": { "field": "url" }
  }
}
```

### 8.2 reload 正在进行

```json
{
  "error": {
    "code": "JOB_RELOAD_IN_PROGRESS",
    "message": "reload in progress",
    "details": { "started_at": "2026-02-23T10:00:00Z" }
  }
}
```

### 8.3 重启 sing-box 失败（输出截断）

```json
{
  "error": {
    "code": "RT_RESTART_FAILED",
    "message": "failed to restart sing-box",
    "details": {
      "container": "singbox",
      "output": "Error response from daemon: No such container: singbox"
    }
  }
}
```

---

## 9. 兼容性声明

错误码是稳定 API 合约的一部分：同一 major 版本内**不移除/不改变含义**。

- 可以**新增**错误码
- 可以**改** message 文案
- **不得**复用旧 code 表达新含义
