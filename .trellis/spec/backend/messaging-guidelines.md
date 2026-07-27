# 客户端与消息规范

## 适用范围

修改 `common/netx`、`common/wsx`、`common/mqttx`、MQTT 请求应答、上传下载或其他长连接 client 时读取。业务 topic/事件契约还要读取本层对应的领域规范。

## 通用 client 规则

- client option 只构造配置；连接、状态、锁和统计由构造函数初始化。
- 所有 I/O 接收 context 或由连接级 context 管理，遵守已有 deadline；不能用 client 默认超时覆盖更短的调用 deadline。
- 请求 body、响应 body、上传下载和帧大小使用现有上限；流式 reader 的读取错误必须返回，不能变成部分成功。
- 调用方负责检查 transport error、状态码/协议结果和解码错误，这三者不能合并成“有 Response 即成功”。
- 长连接组件明确状态转换、认证、心跳、重连和 Stop/Close 所有权；回调在锁外执行并能观察取消。

依据：`common/netx/client.go`、`common/netx/response.go`、`common/netx/upload.go`、`common/netx/download.go`、`common/wsx/client.go`、`common/wsx/config.go`。

## HTTP `netx`

- 复用 `Request` builder 表达 header、query、JSON、form、raw 或 reader body；同一请求只选择一种 body 语义。
- request header 覆盖 client default header；构造时克隆外部 map/slice，避免调用方后续修改造成数据竞争。
- 解码 helper 在非成功状态、nil body、大小超限或格式错误时返回 error；不要绕过 `Response` 约束直接无限读取。
- 上传使用 streaming pipe 时传播生产端错误；下载文件先遵守大小限制和 context，再写目标路径。

## MQTT 请求应答

- `mqttx.ReplyRouter` 只管理关联与解码，业务协议包拥有 topic、method、payload 和结果语义。
- 请求必须按“生成 TID -> 在响应 topic 注册 -> publish -> await -> 清理”的顺序，发送失败立即移除/拒绝 pending。
- router 的 decoder 只提取协议中稳定的关联 ID；解析业务结果由 typed handler/协议层完成。
- 同一 topic 的 response handler 只注册一次，并与 router 生命周期一致；不要为每个请求重复注册 broker handler。

依据：`common/mqttx/reply_router.go`、`common/mqttx/request_replyer.go` 及测试。

## WebSocket `wsx`

- 状态只能通过包内状态机变更；外部使用 `WithOnStateChange` 等回调观察，不能直接修改运行态字段。
- 认证、token refresh、heartbeat 和 reconnect 属于连接生命周期；Stop 后不得再次启动后台循环或写入旧连接。
- 重连期间使用连接 session/context 区分新旧循环，避免旧 reader/heartbeat 影响新连接。

## 反模式

- 记录认证 header、完整请求/响应或设备敏感 payload。
- 无限制 `io.ReadAll` 外部响应或下载。
- 在公共 MQTT 包硬编码 DJI、IEC 104 或其他业务 topic。
- 把 publish 成功描述为业务处理成功或 Exactly Once。

## 验证

- HTTP 覆盖 deadline、取消、header 优先级、状态码、大小上限和 streaming error。
- MQTT 覆盖快速响应、发送失败、未知 TID、超时、重复响应和关闭。
- WebSocket 覆盖连接、认证失败、断线重连、Stop 幂等和旧 session 不再回调；并发变更运行 race test。
