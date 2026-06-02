# 错误码规范

本项目遵循 `google.rpc.Code` 错误码标准，统一 HTTP 和 gRPC 错误码映射。

## HTTP / RPC 错误码映射

| HTTP | RPC | 说明 |
|------|-----|------|
| 200 | `OK` | 无错误 |
| 400 | `INVALID_ARGUMENT` | 客户端指定了无效参数 |
| 400 | `FAILED_PRECONDITION` | 请求无法在当前系统状态下执行（如删除非空目录） |
| 400 | `OUT_OF_RANGE` | 客户端指定了无效范围 |
| 401 | `UNAUTHENTICATED` | OAuth 令牌丢失、无效或过期 |
| 403 | `PERMISSION_DENIED` | 客户端权限不足 |
| 404 | `NOT_FOUND` | 找不到指定的资源 |
| 409 | `ABORTED` | 并发冲突（如读取/修改/写入冲突） |
| 409 | `ALREADY_EXISTS` | 资源已存在 |
| 429 | `RESOURCE_EXHAUSTED` | 资源配额不足或达到速率限制 |
| 499 | `CANCELLED` | 请求被客户端取消 |
| 500 | `DATA_LOSS` | 不可恢复的数据丢失或损坏 |
| 500 | `UNKNOWN` | 未知的服务器错误 |
| 500 | `INTERNAL` | 内部服务器错误 |
| 501 | `NOT_IMPLEMENTED` | API 方法未实现 |
| 503 | `UNAVAILABLE` | 服务不可用 |
| 504 | `DEADLINE_EXCEEDED` | 超出请求时限 |

## 错误消息示例

| HTTP | RPC | 错误消息示例 |
|------|-----|-------------|
| 400 | `INVALID_ARGUMENT` | 请求字段 x.y.z 是 xxx，预期为 [yyy, zzz] 内的一个 |
| 400 | `FAILED_PRECONDITION` | 资源 xxx 是非空目录，因此无法删除 |
| 400 | `OUT_OF_RANGE` | 参数"age"超出范围 [0,125] |
| 401 | `UNAUTHENTICATED` | 身份验证凭据无效 |
| 403 | `PERMISSION_DENIED` | 使用权限"xxx"处理资源"yyy"被拒绝 |
| 404 | `NOT_FOUND` | 找不到资源"xxx" |
| 409 | `ABORTED` | 无法锁定资源"xxx" |
| 409 | `ALREADY_EXISTS` | 资源"xxx"已经存在 |
| 429 | `RESOURCE_EXHAUSTED` | 超出配额限制"xxx" |
| 499 | `CANCELLED` | 请求被客户端取消 |
| 500 | `DATA_LOSS` | 请参阅注释 |
| 500 | `UNKNOWN` | 请参阅注释 |
| 500 | `INTERNAL` | 请参阅注释 |
| 501 | `NOT_IMPLEMENTED` | 方法"xxx"未实现 |
| 503 | `UNAVAILABLE` | 请参阅注释 |
| 504 | `DEADLINE_EXCEEDED` | 请参阅备注 |

## 建议的错误详细信息

| HTTP | RPC | 建议的错误详细信息 |
|------|-----|-------------------|
| 400 | `INVALID_ARGUMENT` | `google.rpc.BadRequest` |
| 400 | `FAILED_PRECONDITION` | `google.rpc.PreconditionFailure` |
| 400 | `OUT_OF_RANGE` | `google.rpc.BadRequest` |
| 401 | `UNAUTHENTICATED` | - |
| 403 | `PERMISSION_DENIED` | - |
| 404 | `NOT_FOUND` | `google.rpc.ResourceInfo` |
| 409 | `ABORTED` | - |
| 409 | `ALREADY_EXISTS` | `google.rpc.ResourceInfo` |
| 429 | `RESOURCE_EXHAUSTED` | `google.rpc.QuotaFailure` |
| 499 | `CANCELLED` | - |
| 500 | `DATA_LOSS` | - |
| 500 | `UNKNOWN` | - |
| 500 | `INTERNAL` | - |
| 501 | `NOT_IMPLEMENTED` | - |
| 503 | `UNAVAILABLE` | - |
| 504 | `DEADLINE_EXCEEDED` | - |
