# 错误处理规范

> 错误处理遵循 Go/go-zero/gRPC 习惯和项目错误码规范，避免 Java 异常式分层和无上下文的吞错。

## 总原则

- 每个 `err` 都要处理；不要空 `catch` 式吞错，不要仅为通过检查而忽略错误。
- 错误返回要保留业务上下文，但不得泄露密钥、连接串、认证头、个人信息或完整内部路径。
- 先参考相邻 Logic、Handler、Server 的错误返回和日志方式，再新增错误封装。
- HTTP 和 gRPC 错误码以 [错误码规范](../../../code.md) 为准。

## HTTP 网关错误模型

BFF 网关和 AI 网关使用不同的错误 handler，Logic 层的错误返回需与 handler 匹配。

### 标准 BFF 网关（`gtw/gtw.go`）

使用 `gtwx.SetGrpcErrorHandler()`，将 gRPC status code 映射为 HTTP status code：

```go
// gRPC status → HTTP status 映射由 common/gtwx/errorhandler.go 完成
// Logic 返回 tool.NewErrorByPbCode(...) 即可自动映射
```

### OpenAI 兼容网关（`aiapp/aigtw/aigtw.go`）

使用 `gtwx.SetOpenAIErrorHandler()`，将错误映射为 OpenAI 风格 JSON 响应：

```go
// 本地校验错误 → invalidRequestError (HTTP 400, OpenAI invalid_request_error)
func invalidRequestError(msg string) error {
    return gtwx.NewInvalidRequestError(msg)
}

// 认证/鉴权错误 → unauthenticatedError (HTTP 401, OpenAI authentication_error)
func unauthenticatedError(msg string) error {
    return status.Error(codes.Unauthenticated, msg)
}
```

**关键规则**：
- 不使用 `tool.NewErrorByPbCode`（会映射为 gRPC 风格而非 OpenAI 风格）
- 本地参数/业务校验用 `gtwx.NewInvalidRequestError` 或本地 `invalidRequestError` helper
- JWT 用户缺失用 `status.Error(codes.Unauthenticated, msg)`（handler 的 `SetOpenAIErrorHandler` 将其转为 OpenAI `authentication_error`）
- 透传上游 gRPC 错误（`aisolo`、`aichat` 等）直接 `return nil, err`，已有结构化状态码
- **附加说明**：`aigtw` 的 flow 式接口（如 `/chat`, `/resume`）在 handler 中先设置了 SSE Header（`text/event-stream`）再调用 Logic。如果 Logic 在校验中返回错误，handler 层面无法改 HTTP 状态码；因此严重的用户身份校验应在 handler 层完成，不要在 SSE 响应中返回非流式错误

### 网关层禁止模式

```go
// ❌ 在 OpenAI 网关使用 errors.New（会兜底到 HTTP 500 internal_error）
return nil, errors.New("missing user id")  // 应改为 invalidRequestError/unauthenticatedError

// ❌ 在 OpenAI 网关使用 tool.NewErrorByPbCode（非 OpenAI 风格）
return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "bad request")  // 应改为 gtwx.NewInvalidRequestError
```

## RPC/gRPC 错误

- RPC Logic 返回符合 gRPC/status/google.rpc.Code 项目约定的错误。
- `.proto` 注释应说明关键失败场景、业务限制和字段含义。
- 下游 RPC、MQTT、Kafka、OSS、Docker、数据库等调用失败时，错误要能定位外部系统和业务动作，但不要泄露敏感参数。

## 项目错误码体系

项目使用 `extproto.Code` 枚举定义统一错误码，结构为六位数字 `ABCDEF`：

| 位 | 含义 | 示例 |
|---|------|------|
| A | 错误来源（固定 `1` = 服务端） | `1` |
| BC | 功能模块 | `02` = 数据, `05` = 业务, `06` = 外部依赖 |
| DEF | 模块内具体错误 | `102` = 记录不存在 |

错误码通过 proto extension 附加名称和 HTTP 映射：

```protobuf
_1_02_RECORD_NOT_EXIST = 102102 [(name) = "记录不存在", (httpCode) = 404];
_1_05_BIZ              = 105101 [(name) = "业务处理失败", (httpCode) = 400];
```

完整错误码表见 [`third_party/extproto.proto`](../../../third_party/extproto.proto) 和 [`code.md`](../../../code.md)。

## 错误工厂

使用 `common/tool` 的三个函数创建和判断错误：

```go
import (
    "zero-service/common/tool"
    "zero-service/third_party/extproto"
)

// 根据 extproto.Code 创建 gRPC 错误，自动读取 name 和 httpCode 映射。
// args 支持三种用法：
//   1. 不传 -> 使用 proto 定义的默认错误名
//   2. 传单个 string -> 作为自定义消息
//   3. 传 string + 额外参数 -> 等价 fmt.Sprintf(msg, args...)
tool.NewErrorByPbCode(code extproto.Code, args ...interface{}) error

// 包装底层错误，保留原始 cause。
// 实现 Go 1.20 多层 Unwrap() []error，errors.Is/As 可同时遍历 structured 和 cause。
// GRPCStatus 指向 structured 错误，status.FromError 正常解析。
tool.NewErrorByPbCodeWrap(code extproto.Code, cause error, args ...interface{}) error

// 判断错误是否匹配指定错误码（通过 gkiterrors.Reason 检测）
tool.IsErrorByPbCode(err error, code extproto.Code) bool
```

`NewErrorByPbCode` 根据 proto `(httpCode)` 自动映射 gRPC status：

| httpCode | gRPC Reason | gkit 构造函数 |
|----------|-------------|--------------|
| 400 | BadRequest | `gkiterrors.BadRequest(reason, message)` |
| 401 | Unauthorized | `gkiterrors.Unauthorized(reason, message)` |
| 403 | Forbidden | `gkiterrors.Forbidden(reason, message)` |
| 404 | NotFound | `gkiterrors.NotFound(reason, message)` |
| 409 | Conflict | `gkiterrors.Conflict(reason, message)` |
| 499 | ClientClosed | `gkiterrors.ClientClosed(reason, message)` |
| 500 | InternalServer | `gkiterrors.InternalServer(reason, message)` |
| 503 | ServiceUnavailable | `gkiterrors.ServiceUnavailable(reason, message)` |
| 504 | GatewayTimeout | `gkiterrors.GatewayTimeout(reason, message)` |

## 编码模式

RPC Logic 中返回错误有四种模式，按优先级排列：

### 模式 A — `tool.NewErrorByPbCode` ✅ 推荐（无原始错误需保留）

业务错误使用精确错误码，调用方能通过 gRPC `reason` 识别错误类型：

```go
// 精确错误码
if err == sqlx.ErrNotFound {
    return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)
}

// 带业务上下文
if in.Value > 65535 {
    return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ, "值超过 16 位寄存器的最大值")
}

// 带格式化参数（等价 fmt.Sprintf）
if totalTokens > maxTokens {
    return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "context too large: %d tokens > %d limit", totalTokens, maxTokens)
}

// 预定义服务级错误变量
var (
    ErrMcpClientNotConfigured = tool.NewErrorByPbCode(extproto.Code__1_05_BIZ, "mcp client 未配置")
    ErrMcpToolNotFound        = tool.NewErrorByPbCode(extproto.Code__1_05_BIZ, "mcp 工具未找到")
)
```

### 模式 B — `tool.NewErrorByPbCodeWrap` ✅ 推荐（需保留原始错误）

DB、RPC、第三方 SDK 等调用失败时，用 `NewErrorByPbCodeWrap` 包装原始错误：

```go
// 数据库错误
if err := db.Count(&total).Error; err != nil {
    return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询设备总数失败")
}

// 第三方调用错误
if _, err := l.svcCtx.DjiClient.DroneEmergencyStop(l.ctx, deviceSn, seq); err != nil {
    return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "无人机紧急停桨失败")
}

// 参数解析错误（原始解析错误保留）
triggerTime := carbon.Parse(in.TriggerTime)
if triggerTime.Error != nil {
    return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, triggerTime.Error, "triggerTime 格式错误")
}
```

`NewErrorByPbCodeWrap` 的 `withCause` 类型实现 Go 1.20 多层链：

```go
type withCause struct {
    structured error  // NewErrorByPbCode 创建的结构化错误
    cause      error  // 原始 cause
}

func (w *withCause) Unwrap() []error {
    return []error{w.structured, w.cause}
}

func (w *withCause) GRPCStatus() *status.Status {
    return status.Convert(w.structured)
}
```

这意味着：
- `status.FromError(err)` → 解析到 structured 错误码，HTTP 网关正确映射
- `gkiterrors.Reason(err)` → 返回结构化错误码 reason
- `errors.Is(err, cause)` / `errors.As(err, &target)` → 可遍历到原始 cause

### 模式 C — 直接 `errors.BadRequest` ⚠️ 仅限简单参数校验

仅在不需要区分错误码类型的简单校验中使用：

```go
return nil, errors.BadRequest("", "参数错误")
```

更好的方式是使用精确错误码替代：

```go
return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "id")
```

### 模式 D — `errors.New` ❌ 避免使用

```go
// 不推荐：无法携带 gRPC status code，调用方无法识别错误类型
return nil, errors.New("结束时间必须晚于开始时间")
```

### 禁止模式

```go
// ❌ 硬编码 reason 字符串，无法映射到 extproto.Code
return nil, errors.BadRequest("9999", "登录失败")

// ❌ fmt.Errorf 作为 NewErrorByPbCode 的参数（冗余，直接传格式化字符串即可）
return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, fmt.Errorf("第 %d 个超出范围", i)) // 应改为：tool.NewErrorByPbCode(code, "第 %d 个超出范围", i)

// ❌ Java 风格异常包装、Result 包装或 Builder 模式
return nil, errors.InternalServer("", "系统内部错误")

// ❌ status.Errorf 直接使用（丢失项目错误码映射）
return nil, status.Errorf(codes.NotFound, "model %s not found", in.Model) // 应改为：tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST, "...")
```

### 透传规则

上游 gRPC、HTTP 调用返回的错误已经包含结构化状态码时，直接透传 `return nil, err` 无需再包装：

```go
// gRPC 客户端调用（下游已使用项目错误码）
resp, err := l.svcCtx.AiSoloCli.CreateSession(l.ctx, req)
if err != nil {
    return nil, err  // 透传，含已有结构化错误信息
}
```

仅当上游错误不包含项目结构（如原生 DB error、SDK error、第三方库 error）时，才需要通过 `NewErrorByPbCodeWrap` 添加上下文和错误码。

## 日志与传播

- 在边界层记录必要上下文，避免每一层重复打印同一个错误。
- Logic 可以补充业务动作、关键 ID、外部系统名和失败阶段。
- 不把用户可见错误直接等同于内部日志错误；用户返回保持稳定，内部日志保留排查线索。

## 常见错误

- 在 Handler 中写业务逻辑并直接拼接错误响应。
- 为了快速返回而丢失底层错误原因。
- 把完整请求体、认证头、连接串、路径或账号写入错误日志。
- 新增一套与项目 `code.md` 不一致的错误码体系。
- 用 Java 风格异常、Result 包装或 Builder 模式替代 Go 的显式错误返回。
