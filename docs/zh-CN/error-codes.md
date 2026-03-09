# BoxPilot 错误码说明

[English](../error-codes.md)

稳定错误码定义位于 `server/internal/util/errorx/codes.go`。

## 统一结构

```json
{
  "error": {
    "code": "SUB_FETCH_FAILED",
    "message": "subscription fetch failed",
    "details": {
      "status": 502
    }
  }
}
```

## 分类

- `REQ_*`：请求与字段校验
- `DB_*`：数据库与 migration
- `SUB_*`：订阅拉取与解析
- `NODE_*`：节点查询与更新
- `CFG_*`：配置生成、检查、回滚
- `RT_*`：运行时启停与状态
- `JOB_*`：并发刷新与调度
- `INTERNAL_ERROR` / `NOT_IMPLEMENTED`：兜底
