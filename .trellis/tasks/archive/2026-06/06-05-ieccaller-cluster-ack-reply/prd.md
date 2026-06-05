# ieccaller 集群指令 ACK 回传

## 目标

将 `app/ieccaller` 已有的指令 ACK 等待能力升级为集群部署可用。当 gRPC 请求落到一个不持有目标 IEC104 客户端连接的 `ieccaller` 实例时，该实例应通过 Kafka 广播下发指令，等待持有目标客户端连接的实例通过 Kafka ACK 队列回传响应，将 ACK 响应值解析成现有 gRPC 返回结构体，并返回给原始 gRPC 调用方。

## 已确认事实

- `ServiceContext.PushPbBroadcast(ctx, method, in)` 当前会把请求序列化到 `types.BroadcastBody`，再推送到 `KafkaConfig.BroadcastTopic`；它没有请求/响应选项，只返回 Kafka 推送成功或失败。
- `PushBroadcast` 发布前会把本地配置的 `BroadcastGroupId` 写入广播消息。
- `kafka.Broadcast.Consume` 会忽略 `BroadcastGroupId` 等于本地配置的消息，然后按 gRPC full method name 分发处理。
- 本地 ACK 型指令逻辑已经调用 IEC104 client 方法并传入 `client.WithAck()`，再把 `client.CommandAck.Value` 转成现有 gRPC response 字段。
- 本地 IEC104 `CommandReplyPool` 是每个 client 独立持有的 reply pool，key 为 `coa:typeID:ioa`，默认 TTL 为 10 秒。
- ACK 超时和重复 pending command 错误已经由 `command_ack_helper.go` 映射为现有 protobuf 错误码。
- `internal/iec/clienthandler.go` 中的 ACK 解析逻辑只会在本地 client reply pool 里存在 pending key 时 resolve 或 reject。
- `app/ieccaller/ieccaller.go` 在配置 Kafka brokers 后会启动一个 `BroadcastTopic` 消费者。
- 现有配置包含 `KafkaConfig.Topic`、`BroadcastTopic`、`BroadcastGroupId`、`IsPush`；目前没有专门的 broadcast ACK topic。
- 用户期望集群 broadcast ACK 同步 resolve 原始 gRPC 主线程，而不是异步稍后通知。

## 需求

- 为需要 ACK 返回值的广播指令增加集群安全的 Kafka 请求/响应链路。
- 保留单机/本地行为：如果当前进程持有目标 IEC104 client，指令逻辑继续走现有 `client.WithAck()` 路径和现有 gRPC response 解析。
- 保留非 ACK 方法的 fire-and-forget 广播行为，例如清缓存、通用非 ACK 指令等。
- 扩展 `PushPbBroadcast` 或新增 option 风格的配套 API，让调用方可以请求一个 reply string，并把 reply 解析到现有 typed gRPC response。
- 增加 broadcast ACK Kafka topic 配置、ACK producer 和 ACK consumer，使集群任意实例都可以把指令执行结果发回原请求方。
- 每个需要等待 ACK 的 broadcast request 和对应 ACK reply 都必须携带唯一 traceId 作为关联标识。
- 只有原始发起实例可以按 traceId resolve 自己 replypool 中的 pending wait；其他实例必须忽略不属于自己的 ACK reply。
- 持有目标 IEC104 client 的 broadcast consumer 必须用和本地 gRPC 直连逻辑相同的 `client.WithAck()` 方法执行 ACK 型指令，然后向 broadcast ACK 队列发送 typed success payload 或 error payload。
- ACK 型 broadcast 只覆盖已经有强业务 ACK 返回值的 RPC：`SendSingleCommand`、`SendDoubleCommand`、`SendStepCommand`、`SendSetpointNormalized`、`SendSetpointScaled`、`SendSetpointFloat`、`SendBitstringCommand`。
- 通用 `SendCommand` 必须保持 fire-and-forget，不纳入集群 ACK reply 范围。
- 超时、拒绝、异常 COT、解析失败、Kafka 发布/消费错误都必须以和现有 command ACK 错误处理一致的方式返回给原始 gRPC 调用方。
- 现有非 ACK broadcast 路由必须保持兼容。

## 验收标准

- [ ] 集群模式下，当接收 gRPC 请求的实例不持有目标 IEC104 client 时，所有纳入范围的 ACK 型指令都可以通过 broadcast 执行，并返回与本地直连路径相同结构的响应值。
- [ ] 持有 IEC104 client 的实例执行 broadcast 指令时使用 `client.WithAck()`，并为它处理的请求发送且只发送一次 success 或 error ACK reply。
- [ ] 原始发起实例等待带 traceId 的 ACK reply，并用解析后的 response 或包装后的错误 resolve replypool 中的 wait。
- [ ] 非原始发起实例和指令执行实例不会误 resolve 其他实例的 pending reply。
- [ ] 现有非 ACK 方法的 fire-and-forget broadcast 行为继续可用。
- [ ] 配置中包含 broadcast ACK topic，并有安全默认值；配置 Kafka 后运行时会启动 ACK consumer。
- [ ] 测试覆盖 request/reply correlation、timeout/error payload 行为，以及至少一个 typed command response 转换。
- [ ] 现有相关测试继续通过。

## 不在范围内

- 修改公开 gRPC/protobuf API。
- 给通用 `SendCommand` 增加 ACK 语义。
- 用直接 gRPC 路由或服务发现转发替代 Kafka broadcast。
- 把 ACK reply 持久化到内存 pending pool 之外的存储。
- 修改 IEC104 `CommandReplyPool` 的 key 语义，除非实现正确性确实需要。

## 已确认决策

- `SendCommand` 保持 fire-and-forget，因为它的 response 为空，且当前本地路径也没有调用 `client.WithAck()`。
