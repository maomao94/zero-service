# 错误码规范

本项目基于 gRPC Code 统一 HTTP 和 gRPC 错误码互转，并通过 `detail` + `reason` 扩展业务错误码。自定义错误码定义在 [`third_party/extproto.proto`](../third_party/extproto.proto)。

## 架构

```
gRPC Code (标准) ──→ HTTP 状态码 (标准映射)
        │
        └── detail.reason ──→ 自定义六位错误码 (extproto)
```

- **gRPC Code**：标准 google.rpc.Code，控制 HTTP 状态码转换
- **detail.reason**：自定义六位错误码（`1BCDEF`），携带业务错误分类

## HTTP / RPC 状态码映射

| HTTP | gRPC Code | 说明 |
|------|-----------|------|
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

## detail.reason 自定义错误码

### 编码规则

错误码为 **ABCDEF** 六位数字：

| 位 | 含义 | 说明 |
|----|------|------|
| A | 错误来源 | 固定 `1`（平台/服务端） |
| BC | 功能模块 | 见下方模块划分 |
| DEF | 模块内错误 | 模块内自定义 |

### 通用系统错误（BC=00）

| reason | 对应 gRPC Code | 说明 |
|--------|---------------|------|
| 100999 | `UNKNOWN` | 未知错误 |
| 100998 | `INTERNAL` | 系统内部错误 |
| 100997 | `DEADLINE_EXCEEDED` | 系统超时 |

### 参数 / 校验错误（BC=01）

| reason | 对应 gRPC Code | 说明 |
|--------|---------------|------|
| 101101 | `INVALID_ARGUMENT` | 参数错误 |
| 101102 | `INVALID_ARGUMENT` | 缺少必要参数 |
| 101103 | `INVALID_ARGUMENT` | 参数不合法 |

### 数据相关错误（BC=02）

| reason | 对应 gRPC Code | 说明 |
|--------|---------------|------|
| 102101 | `INTERNAL` | 数据库错误 |
| 102102 | `NOT_FOUND` | 记录不存在 |
| 102103 | `ALREADY_EXISTS` | 记录已存在 |
| 102104 | `ABORTED` | 数据冲突 |

### 缓存 / 中间件错误（BC=03）

| reason | 对应 gRPC Code | 说明 |
|--------|---------------|------|
| 103101 | `INTERNAL` | 缓存错误 |
| 103102 | `INTERNAL` | 缓存未命中 |
| 103103 | `UNAVAILABLE` | 消息队列错误 |

### 权限 / 认证错误（BC=04）

| reason | 对应 gRPC Code | 说明 |
|--------|---------------|------|
| 104101 | `UNAUTHENTICATED` | 未认证 |
| 104102 | `PERMISSION_DENIED` | 无权限访问 |

### 业务通用错误（BC=05）

| reason | 对应 gRPC Code | 说明 |
|--------|---------------|------|
| 105101 | `INVALID_ARGUMENT` | 业务处理失败 |
| 105102 | `ABORTED` | 业务状态不允许 |
| 105103 | `ALREADY_EXISTS` | 重复操作 |

### 外部依赖错误（BC=06）

| reason | 对应 gRPC Code | 说明 |
|--------|---------------|------|
| 106101 | `UNAVAILABLE` | 远程调用失败 |
| 106102 | `UNAVAILABLE` | 第三方服务异常 |

## 错误消息示例

| HTTP | gRPC Code | 错误消息示例 |
|------|-----------|-------------|
| 400 | `INVALID_ARGUMENT` | 请求字段 x.y.z 是 xxx，预期为 [yyy, zzz] 内的一个 |
| 400 | `FAILED_PRECONDITION` | 资源 xxx 是非空目录，因此无法删除 |
| 400 | `OUT_OF_RANGE` | 参数 "age" 超出范围 [0, 125] |
| 401 | `UNAUTHENTICATED` | 身份验证凭据无效 |
| 403 | `PERMISSION_DENIED` | 使用权限 "xxx" 处理资源 "yyy" 被拒绝 |
| 404 | `NOT_FOUND` | 找不到资源 "xxx" |
| 409 | `ABORTED` | 无法锁定资源 "xxx" |
| 409 | `ALREADY_EXISTS` | 资源 "xxx" 已经存在 |
| 429 | `RESOURCE_EXHAUSTED` | 超出配额限制 "xxx" |
| 499 | `CANCELLED` | 请求被客户端取消 |
| 500 | `INTERNAL` | 内部服务器错误 |
| 501 | `NOT_IMPLEMENTED` | 方法 "xxx" 未实现 |
| 503 | `UNAVAILABLE` | 服务暂不可用 |
| 504 | `DEADLINE_EXCEEDED` | 请求超时 |

## 错误详情扩展

gRPC 错误可通过 `google.rpc.Status.details` 附加结构化上下文：

| gRPC Code | 建议详情类型 | 用途 |
|-----------|-------------|------|
| `INVALID_ARGUMENT` | `google.rpc.BadRequest` | 携带字段级违规列表 |
| `FAILED_PRECONDITION` | `google.rpc.PreconditionFailure` | 携带不满足的前置条件 |
| `OUT_OF_RANGE` | `google.rpc.BadRequest` | 携带越界参数信息 |
| `NOT_FOUND` | `google.rpc.ResourceInfo` | 携带资源类型和名称 |
| `ALREADY_EXISTS` | `google.rpc.ResourceInfo` | 携带已存在的资源标识 |
| `RESOURCE_EXHAUSTED` | `google.rpc.QuotaFailure` | 携带配额违规明细 |

各模块可按需在 `BC=07~99` 区段扩展自定义 reason 码。
