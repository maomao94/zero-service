# 错误处理与上下文传播

## 适用范围

修改 gRPC/HTTP/MCP/网关边界、trace 与业务元数据、错误类型、日志拦截器或响应序列化时读取。

## 上下文传播

- 请求链路继续传递原始 `context.Context`；服务内调用不要替换为 `context.Background()`。
- 身份与认证 context key、claims 映射由 `common/authctx` 管理；gRPC client interceptor 通过 `common/grpcx` 注入允许传播的 metadata，server interceptor 从 metadata 提取并写回 context；MCP `_meta` 与 trace 适配由 `common/mcpx` 管理。
- metadata key 规范为小写；只传播非空字符串。非 ASCII 值按 `common/grpcx/metadata.go` 的编码规则处理，不自定义第二套格式。
- `trace_id`、用户/租户信息和领域任务标识分别由其现有 key 管理，不能互相借用。

依据：`common/authctx/context.go`、`common/grpcx/metadata.go`、`common/grpcx/client_interceptor.go`、`common/grpcx/server_interceptor.go`、`common/mcpx/context_meta.go`。

## 错误所有权

- 领域或公共包定义传输中立的 sentinel/typed error；支持 `errors.Is` / `errors.As` 的包装必须保留原 cause。
- Proto 错误码到 Go error 的统一入口是 `common/tool/errorutil.go`，契约源是 `third_party/extproto.proto`。
- gRPC `status` / `codes`、标准 HTTP 网关 body、OpenAI-compatible body 和 MCP 错误只在对应边界映射。
- ISP 等协议内“对端业务拒绝”与本地超时、断连、关闭等运行错误要分开，不能全部变成 `codes.Internal`。
- DJI `*djisdk.DJIError` 是设备业务拒绝的 typed error，通过 `djisdk.NewDJIError(code)` 构造并由 `djisdk.IsDJIError(err)` 解包；支持 `errors.As` 链。Logic 层通过 `commandError(err)` 区分：DJIError → `CommonRes{Code:-1}` + nil gRPC error；非 DJIError → 原样返回为 gRPC error。
- DJI `*djisdk.PlatformError` 是 handler 侧传输中立错误，携带 `PlatformResult` 码供 `status_reply` / `events_reply` / `requests_reply` 的 `data.result` 使用。普通 error 默认 `PlatformResultHandlerError`；`nil` 默认 `PlatformResultOK`。
- `common/djisdk.ErrSkipRequestReply` 是 sentinel error，request handler 返回它时 `HandleRequests` 跳过 `requests_reply`。
- 不要在底层公共包直接依赖具体网关响应类型。

## 日志边界

- server interceptor 负责集中记录返回错误和请求上下文；Logic 只记录增加业务定位价值的信息，避免同一错误逐层重复打印。
- 日志保留 trace、业务任务/设备等非敏感标识，但不记录认证头、Token、完整连接串或未经筛选的大 payload。
- 记录错误时保留可解包链，避免只留下格式化字符串。
- 返回给外部的 message 不暴露堆栈、内部地址、SQL 或凭据。

依据：`common/grpcx/server_interceptor.go`、`common/gtwx/errorhandler.go`、`common/gtwx/openai_error.go`、`common/tool/errorutil.go`。

## 反模式

- 在领域包返回 `status.Error`，把 gRPC 绑定到可复用逻辑。
- 同一错误在 SDK、Logic、Server 和网关重复记录。
- 手工复制 metadata key 或编码逻辑。
- 通过字符串比较错误，或包装时丢失 cause/GRPCStatus。

## 验证

- 单测 `errors.Is`、`errors.As`、gRPC status 和网关 body 的映射。
- 测试 metadata 的空值、大小写、非 ASCII 和流式 RPC 路径。
- 审查日志样例，确认包含必要关联标识且不含敏感内容。
