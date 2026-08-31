# 实现核对记录

## SDK v2.18.1 API 差异

- `lksdk.NewLiveKitAPI` 只接受 `WithURL`、`WithAPIKey`、`WithToken`，没有 HTTP
  client/transport option；源码中的 `newAPIHTTPClient` 在统一管理入口内部创建并由
  所有子 client 共享。因此 `livekitx` 保留 `Config.HTTPClient` 供调用方和后续扩展
  使用，但不能声称它改变 `LiveKitAPI` 的 transport。
- `lksdk.NewRoom` 只返回 `*Room`，实时连接应先创建 Room，再调用
  `JoinWithContextAndToken`；不能按旧示例把它当作返回 error 的构造函数。
- `auth.VideoGrant.CanPublish` 和 `CanSubscribe` 在锁定 Protocol 中是 `*bool`，便捷
  Token 构造使用局部 bool 指针，以保留显式 false 与字段缺省的协议表达。
- Webhook 校验入口是 `webhook.ReceiveWebhookEvent(*http.Request, auth.KeyProvider)`，
  其返回类型是 `*livekit.WebhookEvent`；包通过 `auth.NewSimpleKeyProvider` 传入 signing
  key，原始 body 和 Authorization 由 SDK 读取校验。

## 未覆盖能力

本阶段未实现完整的通用 typed Realtime Hook 总线（Room、Participant、Track、Connection、
Data、RPC 的每个回调类型），只保留 SDK 原生 `RoomCallback` 接入口和 Chat Hook 类型。
也未添加需要本地 LiveKit dev server 才能运行的集成测试；普通测试不依赖服务端。Egress、
Ingress、SIP、AgentDispatch 的请求入口由 SDK 原生 accessor 暴露，但真实外部服务验证仍需
对应环境。
