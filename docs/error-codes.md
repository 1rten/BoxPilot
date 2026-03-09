# BoxPilot Error Codes

[中文](./zh-CN/error-codes.md)

Stable backend error codes are defined in `server/internal/util/errorx/codes.go`.

## Error Envelope

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

## Categories

### `REQ_*`

Request validation and unsupported input:

- `REQ_BAD_REQUEST`
- `REQ_VALIDATION_FAILED`
- `REQ_MISSING_FIELD`
- `REQ_INVALID_FIELD`
- `REQ_UNSUPPORTED_OPERATION`
- `REQ_TOO_LARGE`

### `DB_*`

Database and migration failures:

- `DB_ERROR`
- `DB_MIGRATION_FAILED`
- `DB_CONSTRAINT_VIOLATION`
- `DB_NOT_FOUND`
- `DB_TX_FAILED`

### `SUB_*`

Subscription fetch and parse failures:

- `SUB_NOT_FOUND`
- `SUB_DISABLED`
- `SUB_INVALID_URL`
- `SUB_FETCH_FAILED`
- `SUB_FETCH_TIMEOUT`
- `SUB_HTTP_STATUS_ERROR`
- `SUB_RESPONSE_TOO_LARGE`
- `SUB_PARSE_FAILED`
- `SUB_FORMAT_UNSUPPORTED`
- `SUB_EMPTY_OUTBOUNDS`
- `SUB_REPLACE_NODES_FAILED`

### `NODE_*`

- `NODE_NOT_FOUND`
- `NODE_TAG_CONFLICT`
- `NODE_INVALID_OUTBOUND`
- `NODE_UPDATE_FAILED`
- `NODE_LIST_FAILED`

### `CFG_*`

Runtime config build and apply failures:

- `CFG_BUILD_FAILED`
- `CFG_NO_ENABLED_NODES`
- `CFG_JSON_INVALID`
- `CFG_WRITE_FAILED`
- `CFG_BACKUP_FAILED`
- `CFG_ROLLBACK_FAILED`
- `CFG_CHECK_FAILED`

### `RT_*`

- `RT_RESTART_FAILED`
- `RT_START_FAILED`
- `RT_STOP_FAILED`
- `RT_STATUS_FAILED`

### `JOB_*`

- `JOB_RELOAD_IN_PROGRESS`
- `JOB_REFRESH_IN_PROGRESS`
- `JOB_SCHEDULER_FAILED`
- `JOB_RATE_LIMITED`

### Fallback

- `INTERNAL_ERROR`
- `NOT_IMPLEMENTED`

## HTTP Mapping

Typical mapping:

- `REQ_*` -> `400`
- `*_NOT_FOUND` -> `404`
- conflict / in-progress errors -> `409`
- upstream subscription failures -> `502`
- internal/runtime failures -> `500` or `503`

## Frontend Consumption

The frontend Axios interceptor stores backend errors on `error.appError`.

UI code usually prefers:

1. `error.appError.message`
2. `error.response.data.error.message`
3. `error.message`
