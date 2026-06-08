# 技术设计：ieccaller 错误日志与 gRPC 错误转换优化

## 架构概览

```
┌─────────────────────────────────────────────────────────────────┐
│                        gRPC Client                              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   LoggerInterceptor                             │
│  - 提取 ctxprop (trace, span, host, port)                       │
│  - 打印结构化 ERROR (method, duration, code, reason)             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Logic Layer                                  │
│  ┌──────────────────┐  ┌──────────────────┐                    │
│  │  单机模式         │  │  集群模式         │                    │
│  │  cli.SendXxx()   │  │  PushBroadcast()  │                    │
│  └────────┬─────────┘  └────────┬─────────┘                    │
│           │                      │                              │
│           ▼                      ▼                              │
│  ┌──────────────────┐  ┌──────────────────┐                    │
│  │ CommandReplyPool │  │ BroadcastReply   │                    │
│  │ (本地 ACK 匹配)   │  │ Pool (MQTT ACK)  │                    │
│  └──────────────────┘  └──────────────────┘                    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                 ClientHandler (ASDU Callback)                   │
│  - resolveCommandAck()                                         │
│  - 结构化拒绝错误 (CommandAck metadata)                         │
└─────────────────────────────────────────────────────────────────┘
```

## 核心改动

### 1. ClientHandler: 结构化拒绝错误

**当前问题**：[clienthandler.go:518](app/ieccaller/internal/iec/clienthandler.go:518) 丢弃 CommandAck 结构化数据

**设计**：创建 `CommandRejectedError` 类型，携带完整 ACK 元数据

```go
// common/iec104/client/errors.go
type CommandRejectedError struct {
    TypeID     int
    Coa        uint
    Ioa        uint
    Cot        string
    CotCause   int
    IsNegative bool
    Status     CommandAckStatus
}

func (e *CommandRejectedError) Error() string {
    return fmt.Sprintf("command rejected: cot=%s isNegative=%v typeId=%d coa=%d ioa=%d",
        e.Cot, e.IsNegative, e.TypeID, e.Coa, e.Ioa)
}
```

**改动位置**：
- `clienthandler.go:516-519` - 使用 CommandRejectedError 替代 fmt.Errorf
- `clienthandler.go:527-528` - 同上

### 2. CommandAckHelper: 精细化错误分类

**当前问题**：[command_ack_helper.go:22](app/ieccaller/internal/logic/command_ack_helper.go:22) 默认归类为 THIRD_PARTY

**设计**：增加 IEC 命令拒绝的专用分类

```go
func wrapCommandAckError(err error, fallbackMsg string) error {
    switch {
    case errors.Is(err, antsx.ErrReplyExpired), errors.Is(err, context.DeadlineExceeded):
        return tool.NewErrorByPbCodeWrap(extproto.Code__1_00_TIMEOUT, err, "IEC控制命令ACK超时: %v", err)
    case errors.Is(err, antsx.ErrDuplicateID):
        return tool.NewErrorByPbCodeWrap(extproto.Code__1_05_BIZ_REPEAT, err, "IEC控制命令重复下发: %v", err)
    case isCommandRejectedError(err):
        // 设备明确拒绝命令，使用业务状态错误码
        return tool.NewErrorByPbCodeWrap(extproto.Code__1_05_BIZ_STATE, err, "IEC命令被设备拒绝: %v", err)
    default:
        return tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "%s: %v", fallbackMsg, err)
    }
}

func isCommandRejectedError(err error) bool {
    var rejected *client.CommandRejectedError
    return errors.As(err, &rejected)
}
```

**错误码映射**：
| 错误类型 | 错误码 | HTTP | 原因 |
|---------|--------|------|------|
| 设备拒绝 (isNegative=true) | 105102 | 409 | 业务状态不允许 |
| ACK 超时 | 100997 | 504 | 超时 |
| 重复下发 | 105103 | 409 | 重复操作 |
| 第三方异常 | 106102 | 503 | 服务不可用 |

### 3. LoggerInterceptor: 增强结构化日志

**当前问题**：[loggerInterceptor.go:18](common/Interceptor/rpcserver/loggerInterceptor.go:18) 只打印泛化错误

**设计**：提取 gRPC Status 详情，打印结构化字段

```go
func LoggerInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
    start := time.Now()
    ctx = ctxprop.ExtractFromGrpcMD(ctx)
    
    resp, err = handler(ctx, req)
    
    if err != nil {
        duration := time.Since(start)
        st, _ := status.FromError(err)
        
        logx.WithContext(ctx).Errorw("【RPC-SRV-ERR】",
            logx.Field("method", info.FullMethod),
            logx.Field("duration", duration.String()),
            logx.Field("grpc_code", st.Code().String()),
            logx.Field("reason", st.Proto().GetReason()),
            logx.Field("message", st.Message()),
        )
    }
    return resp, err
}
```

### 4. 业务 Logic 层: 统一日志级别

**设计原则**：
- 设备拒绝命令 → `Warnw` (业务层面的预期行为)
- 服务故障（找不到客户端、MQTT 失败）→ `Errorw` (服务异常)

**示例** (sendsetpointfloatlogic.go):
```go
if err != nil {
    if isCommandRejectedError(err) {
        logx.WithContext(l.ctx).Warnw("IEC命令被设备拒绝",
            logx.Field("method", "SendSetpointFloat"),
            logx.Field("coa", in.GetCoa()),
            logx.Field("ioa", in.GetIoa()),
            logx.Field("error", err.Error()),
        )
    } else {
        logx.WithContext(l.ctx).Errorw("IEC发送浮点设点命令失败",
            logx.Field("method", "SendSetpointFloat"),
            logx.Field("coa", in.GetCoa()),
            logx.Field("ioa", in.GetIoa()),
            logx.Field("error", err.Error()),
        )
    }
    return nil, wrapCommandAckError(err, "IEC发送浮点设点命令失败")
}
```

### 5. BroadcastAck: 细化错误分类

**当前问题**：[broadcast.go:325-332](app/ieccaller/mqtt/broadcast.go:325) 只有 timeout/duplicate/unknown

**设计**：增加 IEC 命令拒绝分类

```go
func (l *Broadcast) publishAckReply(ctx context.Context, tId, ackTopic, method string, success bool, responseBody string, ackErr error) {
    // ... existing code ...
    if ackErr != nil {
        errMsg = ackErr.Error()
        switch {
        case errors.Is(ackErr, antsx.ErrReplyExpired):
            errorKind = "timeout"
        case errors.Is(ackErr, antsx.ErrDuplicateID):
            errorKind = "duplicate"
        case isCommandRejectedError(ackErr):
            errorKind = "iec_rejected"
        default:
            errorKind = "unknown"
        }
    }
    // ... rest of code ...
}
```

### 6. BroadcastReplyPool: 保留原始错误

**当前问题**：[servicecontext.go:320](app/ieccaller/internal/svc/servicecontext.go:320) 丢失原始错误详情

**设计**：在 BroadcastAckBody 中携带原始错误类型

```go
if !ack.Success {
    switch ack.ErrorKind {
    case "timeout":
        return antsx.ErrReplyExpired
    case "duplicate":
        return antsx.ErrDuplicateID
    case "iec_rejected":
        // 重建 CommandRejectedError 以保留语义
        return reconstructRejectedError(ack.Error)
    default:
        return fmt.Errorf("broadcast command error: %s", ack.Error)
    }
}
```

## 日志字段规范

### 必需字段

| 字段 | 说明 | 示例 |
|------|------|------|
| `method` | RPC 方法名 | `/ieccaller.IecCaller/SendSetpointFloat` |
| `duration` | 请求耗时 | `1.234s` |
| `grpc_code` | gRPC 状态码 | `FAILED_PRECONDITION` |
| `reason` | 业务错误码 | `105102` |
| `message` | 错误摘要 | `IEC命令被设备拒绝: cot=UnknownTypeID` |
| `trace` | 链路追踪 ID | `2f6839dfe8b0b8a4ee28ce062a8c7ef5` |
| `span` | Span ID | `4234c9b2884a946f` |

### IEC 命令专用字段

| 字段 | 说明 | 示例 |
|------|------|------|
| `coa` | Common Address | `1` |
| `ioa` | Information Object Address | `1001` |
| `typeId` | ASDU Type ID | `46` |
| `cot` | Cause of Transmission | `UnknownTypeID` |
| `cotCause` | COT 枚举值 | `45` |
| `isNegative` | 是否否定确认 | `true` |
| `ackStatus` | ACK 状态 | `rejected` |

### 集群模式专用字段

| 字段 | 说明 | 示例 |
|------|------|------|
| `broadcastTid` | 广播事务 ID | `uuid-xxx` |
| `broadcastMethod` | 广播方法名 | `SendSetpointFloat` |
| `nodeId` | 节点实例 ID | `ieccaller-node-1` |
| `deployMode` | 部署模式 | `cluster` |

## 向后兼容性

1. **错误码兼容**：复用现有 `105102 业务状态不允许`，不新增枚举
2. **日志格式兼容**：保持 `【RPC-SRV-ERR】` 前缀，增加结构化字段
3. **gRPC Status 兼容**：`GRPCStatus()` 方法保持不变，HTTP 网关无需改动

## 验证方案

### 单元测试

1. `CommandRejectedError` 序列化/反序列化
2. `wrapCommandAckError` 分类逻辑
3. `publishAckReply` ErrorKind 分类

### 集成测试

1. 单机模式发送命令被拒 → 验证 409 / 105102
2. 集群模式发送命令被拒 → 验证错误链路完整
3. ACK 超时 → 验证 504 / 100997

### 日志验证

1. 检查 LoggerInterceptor 输出包含所有必需字段
2. 检查业务层 WARN/ERROR 级别正确
3. 检查集群日志包含 broadcastTid
