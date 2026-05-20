# 错误处理规范

> 错误处理遵循 Go/go-zero/gRPC 习惯和项目错误码规范，避免 Java 异常式分层和无上下文的吞错。

## 总原则

- 每个 `err` 都要处理；不要空 `catch` 式吞错，不要仅为通过检查而忽略错误。
- 错误返回要保留业务上下文，但不得泄露密钥、连接串、认证头、个人信息或完整内部路径。
- 先参考相邻 Logic、Handler、Server 的错误返回和日志方式，再新增错误封装。
- HTTP 和 gRPC 错误码以 [错误码规范](../../../code.md) 为准。

## HTTP/API 错误

- Handler 只做请求解析、基础校验、调用 Logic 和响应输出。
- 返回错误时使用 go-zero 项目既有 `httpx`/错误响应模式，不直接 `w.Write` 或 `fmt.Fprintf` 拼响应。
- `.api` 中的注释、请求响应结构和实际错误行为要保持一致。

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

使用 `common/tool` 的两个函数创建和判断错误：

```go
import (
    "zero-service/common/tool"
    "zero-service/third_party/extproto"
)

// 根据 extproto.Code 创建 gRPC 错误，自动读取 name 和 httpCode 映射
tool.NewErrorByPbCode(code extproto.Code, args ...interface{}) error

// 判断错误是否匹配指定错误码
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

RPC Logic 中返回错误有三种模式，按优先级排列：

### 模式 A — `tool.NewErrorByPbCode` ✅ 推荐

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

// 预定义服务级错误变量
var (
    ErrMcpClientNotConfigured = tool.NewErrorByPbCode(extproto.Code__1_05_BIZ, "mcp client 未配置")
    ErrMcpToolNotFound        = tool.NewErrorByPbCode(extproto.Code__1_05_BIZ, "mcp 工具未找到")
)
```

### 模式 B — 直接 `errors.BadRequest` ⚠️ 仅限简单参数校验

仅在不需要区分错误码类型的简单校验中使用：

```go
return nil, errors.BadRequest("", "参数错误")
```

更好的方式是使用精确错误码替代：

```go
return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "id")
```

### 模式 C — `errors.New` ❌ 避免使用

```go
// 不推荐：无法携带 gRPC status code，调用方无法识别错误类型
return nil, errors.New("结束时间必须晚于开始时间")
```

### 禁止模式

```go
// ❌ 硬编码 reason 字符串，无法映射到 extproto.Code
return nil, errors.BadRequest("9999", "登录失败")

// ❌ Java 风格异常包装、Result 包装或 Builder 模式
return nil, errors.InternalServer("", "系统内部错误")
```

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
