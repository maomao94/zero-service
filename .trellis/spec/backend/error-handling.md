# 错误处理与上下文传播

> 修改 gRPC/HTTP/MCP/网关边界、trace 与业务元数据、错误类型、日志拦截器或响应序列化时读取。
> 通用错误处理规则见 [core-rules.md](./core-rules.md)。

## 上下文传播

| 组件 | 职责 | 依据 |
|------|------|------|
| `common/authctx` | 身份与认证 context key、claims 映射 | `context.go` |
| `common/grpcx` | gRPC client/server interceptor，metadata 注入/提取 | `metadata.go`、`*_interceptor.go` |
| `common/mcpx` | MCP `_meta` 与 trace 适配 | `context_meta.go` |

**规则：**
- metadata key 规范为小写；只传播非空字符串
- 非 ASCII 值按 `common/grpcx/metadata.go` 的编码规则处理，不自定义第二套格式
- `trace_id`、用户/租户信息和领域任务标识分别由其现有 key 管理，不能互相借用

## 错误所有权

| 错误类型 | 归属 | 说明 |
|---------|------|------|
| Proto 错误码 → Go error | `common/tool/errorutil.go` | 契约源：`third_party/extproto.proto` |
| ISP 对端业务拒绝 | ISP 领域错误 | 不能全部变成 `codes.Internal` |
| DJI 设备业务拒绝 | `*djisdk.DJIError` | `NewDJIError(code)` 构造，`IsDJIError(err)` 解包 |
| DJI handler 侧错误 | `*djisdk.PlatformError` | 携带 `PlatformResult` 码 |
| 跳过 request reply | `ErrSkipRequestReply` | sentinel error，`HandleRequests` 跳过回复 |

**DJI 错误处理模式：**
```go
// Logic 层通过 commandError(err) 区分
if djisdk.IsDJIError(err) {
    return &CommonRes{Code: -1}, nil // DJIError → 业务错误码
}
return nil, err // 非 DJIError → 原样返回 gRPC error
```

**关键规则：**
- 不要在底层公共包直接依赖具体网关响应类型
- Proto 错误码到 Go error 的统一入口是 `common/tool/errorutil.go`

依据：`common/grpcx/server_interceptor.go`、`common/gtwx/errorhandler.go`、`common/tool/errorutil.go`

## 日志边界

| 规则 | 说明 |
|------|------|
| 可解包链 | 记录错误时保留可解包链，避免只留下格式化字符串 |
| 外部 message | 不暴露堆栈、内部地址、SQL 或凭据 |

## 反模式

| 错误做法 | 正确做法 | 原因 |
|---------|---------|------|
| 在领域包返回 `status.Error` | 领域包用传输中立错误 | 把 gRPC 绑定到可复用逻辑 |
| 同一错误在 SDK/Logic/Server/网关重复记录 | 只在边界记录一次 | 日志冗余 |
| 手工复制 metadata key | 使用 `common/grpcx` | 编码逻辑不一致 |
| 字符串比较错误 | 用 `errors.Is`/`errors.As` | 包装后会失败 |

## 验证

- 单测 `errors.Is`、`errors.As`、gRPC status 和网关 body 的映射
- 测试 metadata 的空值、大小写、非 ASCII 和流式 RPC 路径
- 审查日志样例，确认包含必要关联标识且不含敏感内容
