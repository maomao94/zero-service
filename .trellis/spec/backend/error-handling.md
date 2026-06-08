# 错误处理规范

> HTTP/gRPC 错误、错误码、包装和传播的 canonical source。错误码映射说明见 [docs/error-codes.md](../../../docs/error-codes.md)，枚举源码见 [third_party/extproto.proto](../../../third_party/extproto.proto)。

## When to read

- 改 Logic、Handler、Server 或网关错误返回。
- 新增或调整 `extproto.Code`、HTTP/gRPC 映射、错误日志传播。
- 包装 DB、RPC、MQTT、Kafka、OSS、Docker、第三方 SDK 等外部错误。

## 总原则

- 每个 `err` 都要处理，不吞错，不为通过检查而忽略错误。
- 错误返回保留业务上下文，但不得泄露密钥、连接串、认证头、个人信息或完整内部路径。
- 先参考相邻 Logic、Handler、Server 的错误返回和日志方式，再新增错误封装。
- 用户可见错误保持稳定，内部日志保留排查线索。

## 网关错误模型

| 网关 | Handler | Logic 应返回 |
| --- | --- | --- |
| 标准 BFF `gtw/gtw.go` | `gtwx.SetGrpcErrorHandler()` | `tool.NewErrorByPbCode(...)` 或下游结构化 gRPC 错误 |
| OpenAI 兼容 `aiapp/aigtw/aigtw.go` | `gtwx.SetOpenAIErrorHandler()` | `gtwx.NewInvalidRequestError(...)`、`status.Error(codes.Unauthenticated, ...)` 或透传上游 gRPC 错误 |

OpenAI 网关规则：

- 本地参数和业务校验用 `gtwx.NewInvalidRequestError` 或本地 `invalidRequestError` helper。
- JWT 用户缺失用 `status.Error(codes.Unauthenticated, msg)`。
- 透传上游 gRPC 错误时直接 `return nil, err`。
- flow 式 SSE 接口在 handler 先写 `text/event-stream` header，再调用 Logic。严重身份校验应放在 handler 层，否则 Logic 返回错误时无法改 HTTP 状态码。

Wrong:

```go
return nil, errors.New("missing user id")
return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "bad request")
```

Correct:

```go
return nil, gtwx.NewInvalidRequestError("missing user id")
return nil, status.Error(codes.Unauthenticated, "missing user id")
```

## 项目错误码体系

- `extproto.Code` 是错误码枚举源码，位置是 [`third_party/extproto.proto`](../../../third_party/extproto.proto)。
- 用户可见映射说明在 [`docs/error-codes.md`](../../../docs/error-codes.md)。
- 结构为六位数字 `ABCDEF`：`A` 错误来源，`BC` 功能模块，`DEF` 模块内具体错误。
- proto extension 提供错误名和 HTTP 映射：

```protobuf
_1_02_RECORD_NOT_EXIST = 102102 [(name) = "记录不存在", (httpCode) = 404];
_1_05_BIZ              = 105101 [(name) = "业务处理失败", (httpCode) = 400];
```

## 错误工厂签名

```go
tool.NewErrorByPbCode(code extproto.Code, args ...interface{}) error
tool.NewErrorByPbCodeWrap(code extproto.Code, cause error, args ...interface{}) error
tool.IsErrorByPbCode(err error, code extproto.Code) bool
```

Contracts:

- `NewErrorByPbCode` 根据 proto `(name)` 和 `(httpCode)` 创建结构化 gRPC 错误。
- `args` 为空时使用 proto 默认错误名；传一个 string 时作为自定义消息；传 string 加参数时按 `fmt.Sprintf` 格式化。
- `NewErrorByPbCodeWrap` 保留原始 cause，并让 `GRPCStatus()` 指向结构化错误。
- `status.FromError(err)` 解析结构化错误码；`errors.Is/As` 可遍历原始 cause。
- `IsErrorByPbCode` 通过 `gkiterrors.Reason` 判断项目错误码。

HTTP to gRPC 映射由 proto `(httpCode)` 和 `common/tool` 决定：400 BadRequest，401 Unauthorized，403 Forbidden，404 NotFound，409 Conflict，499 ClientClosed，500 InternalServer，503 ServiceUnavailable，504 GatewayTimeout。

## Logic 返回模式

| 模式 | 何时用 | 示例 |
| --- | --- | --- |
| `tool.NewErrorByPbCode` | 业务错误，无需保留原始错误 | `tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)` |
| `tool.NewErrorByPbCodeWrap` | DB、SDK、外部系统失败，需要保留 cause | `tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "无人机紧急停桨失败")` |
| `errors.BadRequest` | 仅限不需要区分项目错误码的简单校验 | 优先改用精确 `extproto.Code` |
| 直接透传 `err` | 上游已带结构化 gRPC 状态码 | `return nil, err` |

Good:

```go
if err == sqlx.ErrNotFound {
    return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)
}

if err := db.Count(&total).Error; err != nil {
    return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询设备总数失败")
}

if totalTokens > maxTokens {
    return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "context too large: %d tokens > %d limit", totalTokens, maxTokens)
}
```

Base:

```go
resp, err := l.svcCtx.AiSoloCli.CreateSession(l.ctx, req)
if err != nil {
    return nil, err
}
```

Bad:

```go
return nil, errors.New("结束时间必须晚于开始时间")
return nil, status.Errorf(codes.NotFound, "model %s not found", in.Model)
return nil, errors.BadRequest("9999", "登录失败")
return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, fmt.Errorf("第 %d 个超出范围", i))
```

## Validation & Error Matrix

| 条件 | 正确行为 |
| --- | --- |
| 参数缺失或格式错误 | 返回 `Code__1_01_*` 或 OpenAI 网关的 `invalid_request_error` |
| 记录不存在 | 返回 `Code__1_02_RECORD_NOT_EXIST` |
| 数据库或原生 SDK 错误 | 用 `NewErrorByPbCodeWrap` 添加项目错误码和业务动作 |
| 下游已返回项目结构化 gRPC 错误 | 直接透传，不重复包装 |
| 需要用户可见错误信息 | 文案稳定，不暴露内部路径、连接串、密钥或完整请求体 |
| 需要排障上下文 | 日志记录关键 ID、外部系统名和失败阶段，避免多层重复打印 |

## RPC 拦截器日志约定

- 所有 error 日志统一在 `LoggerInterceptor` 打印，使用 `%+v` 直接输出完整错误链。
- Logic 层只返回 error，**不在业务层单独打 error/warn 日志**，避免重复。
- 格式：`logx.WithContext(ctx).Errorf("【RPC-SRV-ERR】%+v method=%s duration=%s", err, info.FullMethod, ...)`
- `%+v` 会自动展开 `withCause` 链：`第三方服务异常(106102): IEC命令被设备拒绝: command rejected: cot=UnknownTypeID ...`

Wrong:
```go
// Logic 层重复打日志
if err != nil {
    logx.WithContext(l.ctx).Errorw(...)
    return nil, wrapCommandAckError(err, "IEC发送命令失败")
}
```

Correct:
```go
// Logic 层只返回 error，拦截器统一打印
if err != nil {
    return nil, wrapCommandAckError(err, "IEC发送命令失败")
}
```

## Tests Required

- 错误工厂单测：断言 `status.FromError`、`gkiterrors.Reason`、HTTP 映射和默认错误名。
- wrap 单测：断言 `errors.Is/As` 能找到原始 cause，且 `status.FromError` 仍是结构化错误。
- 网关集成或 handler 单测：标准 BFF 映射 HTTP 状态码，OpenAI 网关输出 OpenAI 风格错误 JSON。
- Logic 单测或集成测试：记录不存在、参数错误、外部系统失败、下游结构化错误透传。

## Common mistakes

- 在 Handler 中写业务逻辑并直接拼接错误响应。
- 为了快速返回而丢失底层错误原因。
- 把完整请求体、认证头、连接串、路径或账号写入错误日志。
- 新增一套与项目 `extproto.Code` 不一致的错误码体系。
- 用 Java 风格异常、Result 包装或 Builder 模式替代 Go 的显式错误返回。
- **在 Logic 层打 error/warn 日志然后继续返回 error**，导致同一错误被打印多次。应只在 LoggerInterceptor 统一打印。
