# Unify context propagation helpers

## Goal

将身份 context、claims、gRPC metadata、MCP `_meta` 和 Raw Codec 按数据所有权与 transport 重新归类，删除职责重叠的 `ctxdata`、`ctxprop` 历史包，并保持所有请求、通信 key 和运行行为不变。

## Background

- `app/trigger/cron/cronservice.go:157` 原有一份未使用、名称为 `proto_raw` 的 codec 死代码。
- `app/trigger/internal/task/deferTriggerProtoTask.go:103` 和 `app/trigger/internal/invoke/grpc_invoker.go:78` 各自定义并使用相同 codec，名称分别为 `proto_raw` 和 `invoke_raw`。
- 三份实现都通过 `common/tool.ToProtoBytes` 编码请求，并将响应追加到调用方提供的 `*[]byte`。
- 该能力不依赖 Trigger proto、model 或业务错误，适合作为独立 gRPC 公共机制，而不应继续堆入历史混合包 `common/tool`。
- `common/Interceptor/rpcclient` 提供 unary/stream 客户端 metadata 传播拦截器，共有 14 个导入点。
- `common/Interceptor/rpcserver` 提供 unary/stream 服务端 context 提取和错误日志拦截器，共有 21 个导入点。
- 旧目录使用大写 `Interceptor`，且 client/server 包均声明为不自解释的 `interceptor`；这些能力均只依赖公共 gRPC、`ctxprop` 和 `logx`，与 `common/grpcx` 的职责一致。
- `ctxdata` 混合身份 context key、gRPC/HTTP header 和 MCP `_meta`，`ctxprop` 又混合 claims、gRPC metadata 与 MCP trace，职责和依赖方向不清晰。
- 全仓依赖面为 `ctxdata` 37 个文件、`ctxprop` 6 个文件、`grpcx` 35 个文件；迁移必须原子完成并编译所有直接调用方。

## Requirements

- 在 `common/grpcx` 提供可配置名称的 gRPC Raw Codec。
- 保持现有请求编码行为，继续复用 `common/tool.ToProtoBytes`。
- 保持响应解码行为：只接受 `*[]byte`，并将收到的数据追加到目标切片。
- 保持现有 codec 名称：原 `proto_raw` 调用点仍使用 `proto_raw`，原 `invoke_raw` 调用点仍使用 `invoke_raw`。
- 替换并删除 Trigger 中三份本地 codec 实现。
- 公共 API 和非直观的原始字节行为必须有简短说明与单元测试。
- 将 `common/Interceptor/rpcclient` 和 `common/Interceptor/rpcserver` 的公开拦截器移动到 `common/grpcx`。
- 全仓调用方直接导入并使用 `common/grpcx`；不保留旧路径兼容转发包。
- 保持 `UnaryMetadataInterceptor`、`StreamTracingInterceptor`、`LoggerInterceptor`、`StreamLoggerInterceptor` 的签名和行为不变。
- 删除迁移完成后的 `common/Interceptor` 旧目录，避免大小写敏感文件系统和重复所有权问题。
- 新建 `common/authctx`，拥有身份/认证 context key、getter、claims string conversion 和 claim mapping。
- 将 gRPC metadata key、ASCII/base64 wire 编解码和 metadata 注入/提取移动到 `common/grpcx`。
- 将 MCP context 收集、`_meta` 提取、原始 `_meta` context 存取和 trace 提取移动到 `common/mcpx`。
- 全仓迁移 `ctxdata`、`ctxprop` 调用后删除旧包，不保留兼容 wrapper。
- 允许 Go API 按职责重命名，但所有通信字符串、context 字符串 key、字段集合和转换顺序必须保持不变。

## Acceptance Criteria

- [x] `common/grpcx` 提供满足 gRPC codec 调用要求的 Raw Codec，并允许调用方指定名称。
- [x] protobuf 消息和原始字节请求的编码行为与现有 `tool.ToProtoBytes` 一致。
- [x] 响应可以写入 `*[]byte`，多次反序列化保持追加语义，错误目标类型返回错误。
- [x] Trigger 两个真实调用点统一使用公共实现且名称不变，第三份未使用实现已删除。
- [x] 不再存在 Trigger 私有的重复 Raw Codec 类型。
- [x] 全仓 gRPC client/server 拦截器调用统一使用 `common/grpcx`，旧导入路径和旧目录不再存在。
- [x] 四个拦截器的 metadata/context 传播和错误透传行为保持不变，并有公共包单元测试。
- [x] `common/grpcx`、所有直接调用方编译/测试通过；除既有 `iecagent` vet 告警外相关 `go vet` 通过，`git diff --check` 通过。
- [x] `common/authctx` 成为认证身份 context 与 claim mapping 的唯一所有者，旧 `ctxdata` 不再存在。
- [x] gRPC metadata 转换只位于 `common/grpcx`，旧 `ctxprop` 不再存在。
- [x] MCP `_meta`、raw meta context 和 trace 处理只位于 `common/mcpx`。
- [x] 全仓调用方迁移到新 API，且不存在旧包导入或旧符号残留。
- [x] 所有通信 key 与重构前完全一致，并由契约测试锁定。
- [x] gRPC、MCP、JWT/HTTP 入口和身份读取的直接调用方测试/编译通过。
- [x] `ClaimMapping` 的配置与代码注释统一明确为 `internalKey -> externalKey`，不改变配置值或运行方向。
- [x] `grpcx` metadata schema 只保留实际使用的 context key、gRPC metadata key 和稳定顺序；不再携带无消费者的 HTTP/Sensitive 字段。
- [x] metadata key 小写、ClaimMapping 方向和现有传播行为有回归测试。

## Out Of Scope

- 不抽取或修改 `RawProtoToJSON`。
- 不抽取 `optionalJSON`、CronJob Extra、`sql.NullString` JSON 转换等 Trigger 领域逻辑。
- 不修改 protobuf 契约、生成文件、gRPC 方法、超时、连接缓存或错误映射。
- 不改变 `common/tool.ToProtoBytes` 的现有转换语义。
- 不重命名现有拦截器公开函数，不改变拦截器顺序或日志内容。
- 不迁移 HTTP、中间件或非 gRPC 拦截器。
- 不改变以下 context/claim key：`user-id`、`user-name`、`dept-code`、`authorization`、`auth-type`、`_meta`。
- 不改变以下 gRPC metadata key：`x-user-id`、`x-user-name`、`x-dept-code`、`authorization`、`x-auth-type`。
- 不改变 `b64:` 前缀、非 ASCII 编解码、metadata 覆盖/首值读取、空值过滤或字段处理顺序。
- 不改变 ClaimMapping 的 `internalKey -> externalKey` 方向或现有宽松字符串转换。
- 不改变 Authorization 在 gRPC/MCP 中的传播行为，不在本任务处理安全策略收缩。
- 不改变 MCP `_meta` 和 W3C traceparent 的请求结构。
- 不改变 `ClaimMapping` 配置值，只修正文档语义。
- 不改变 gRPC metadata key、HTTP header 外部协议或 Authorization/Sensitive 的传播策略；本轮仅删除未被运行时代码消费的 schema 字段。

## Deferred Items

- Typed context key：长期应将进程内 typed key 与 JWT/MCP wire key 分离；迁移会影响直接 `context.WithValue` 调用，需独立设计双读/双写方案。
- `b64:` 编码协议：当前前缀用于标识中文等非 ASCII metadata 的 Base64 编码，本任务保持现有发送和解码规则；任何协议升级需考虑混合版本服务。
- Authorization 传播策略：当前继续在 gRPC metadata 和 MCP `_meta` 中传播，不在本任务收缩认证链路。
- Claim 值规范化：保留 float64 和其他类型的宽松字符串转换，不在本任务改变 JWT 兼容行为。
- 重复 metadata：保留客户端 `Set` 覆盖和服务端读取首值的现有规则。
