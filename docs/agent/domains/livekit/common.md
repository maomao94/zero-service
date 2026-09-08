# LiveKit 视频会议对接规范

## 适用范围

修改 `common/livekitx`、LiveKit JWT/Webhook/Twirp 或房间管理 API 时读取。版本、部署和集成验证见[平台与运维](./operations.md)；完整 API 基线见[对接指南](../../../live/integration-guide.md)。

## Scenario: common/livekitx 公共 API 契约

### 1. Scope / Trigger

`common/livekitx` 是零业务依赖的机制层：统一配置与生命周期、Twirp 管理 API 访问、Token 签发、Webhook 验签 KeyProvider。**不包装任何 SDK 方法**——业务通过 `client.Room()` 拿到 `livekit.RoomService` 后直接调用 SDK 原生方法，通过 `lksdk.NewRoom()` + `JoinWithContext` 加入房间，通过 `livekitx.NewJoinToken()` 签发 token。**实时事件（断开、重连、轨道、Data、聊天、RPC）全部由业务通过 SDK 原生 `RoomCallback` 处理**，包内不存在任何事件分发机制。修改该包配置/生命周期、Token 构造或 Webhook 验签时适用。

### 2. Signatures

```go
// 业务系统在服务启动时构造一次并复用；Option 直接作用于 Client。
client, err := livekitx.New(
    livekitx.WithURL(cfg.LiveKit.ServerURL),
    livekitx.WithAPIKey(cfg.LiveKit.APIKey, cfg.LiveKit.APISecret),
    livekitx.WithHTTPService(httpc.NewService("livekit")), // 可选
)

// 管理 API：业务直接使用 SDK 原生 RoomService 方法
roomSvc := client.Room()
roomSvc.CreateRoom(ctx, &livekit.CreateRoomRequest{Name: roomName, ...})
roomSvc.DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: roomName})
roomSvc.RemoveParticipant(ctx, &livekit.RoomParticipantIdentity{Room: roomName, Identity: id})
roomSvc.SendData(ctx, &livekit.SendDataRequest{Room: roomName, Topic: &topic, Data: payload, Kind: livekit.DataPacket_RELIABLE, DestinationIdentities: dests})
roomSvc.MutePublishedTrack(ctx, &livekit.MuteRoomTrackRequest{Room: roomName, Identity: id, TrackSid: sid, Muted: muted})
roomSvc.ListParticipants(ctx, &livekit.ListParticipantsRequest{Room: roomName})

// 加入房间：SDK 原生方式
room := lksdk.NewRoom(&lksdk.RoomCallback{...})
room.JoinWithContext(ctx, cfg.LiveKit.ServerURL, lksdk.ConnectInfo{
    APIKey: key, APISecret: secret, RoomName: roomName, ParticipantIdentity: identity,
})
defer room.Disconnect()

// 入会 Token 签发：直接调用函数
token, err := livekitx.NewJoinToken(livekitx.JoinTokenOptions{
    APIKey: apiKey, APISecret: apiSecret, Room: roomName, Identity: identity,
    ValidFor: validFor, CanPublish: true, CanSubscribe: true, CanPublishData: true,
})

// Webhook：业务自行验签
event, err := webhook.ReceiveWebhookEvent(req, livekitx.NewWebhookKeyProvider(cfg.WebhookKey))
```

- `New(opts ...Option) (*Client, error)`：Option 为 `func(*Client)`，配置类（URL/凭据/HTTP）写入 `c.config`；校验 URL、成对凭据和 transport 冲突；不派生任何默认超时。
- `(*Client).Room() livekit.RoomService`：返回底层 SDK RoomService，供业务直接调用。
- `(*Client).API() *API`：返回协议生成的管理 client（Room/Egress/Ingress/SIP/AgentDispatch）。
- `(*Client).Config() Config`：返回构造时的配置副本。
- `(*Client).Close() error`：仅幂等标记关闭；不管理实时连接、不关闭业务注入的 HTTP client。

### 3. Contracts

- 请求超时只能由业务 context 控制；`livekitx` 不设置、不覆盖、不派生 deadline。禁止重新引入 Timeout option。
- 管理 API 的请求级 Token 必须带房间范围 `VideoGrant.Room`，否则房间级参与者管理/Data/静音/元数据请求返回 `unauthenticated: permissions denied`。
- 实时回调是 SDK 原生 `*lksdk.RoomCallback`，原样透传（无桥接、无 Merge、无合成事件）；`JoinWithContext` 成功返回即"已连接"（SDK 原生无 connected 回调）；断开由业务调 `room.Disconnect()`。
- 聊天/富媒体约定交业务：文本可用 `PublishChatMessage`/`*livekit.ChatMessage`；语音、图片等二进制走 `SendData`（`UserDataPacket`，接收方在 `OnDataPacket` 收到），topic 与 payload 格式由业务定义。
- Webhook 事件可能重复、迟到、乱序或丢失；业务在 handler 中按 event ID 持久化去重，公共包不建默认永久内存去重，不承诺 Exactly Once。
- `NewJoinToken` 签发的 token 只用于客户端直连；加入房间走 `ConnectInfo` 固定最小 grant（RoomJoin+Room），业务需要自定义权限时用 `NewJoinToken(opts)` 或 SDK 原生 `JoinWithContextAndToken`。

### 4. Validation & Error Matrix

- 空 URL / 缺 key 或 secret / `HTTPClient` 与 `HTTPService` 同时注入 → `ErrInvalidConfig`（构造期返回）。
- Webhook 验签失败 → `webhook.ReceiveWebhookEvent` 返回签名错误，不调用任何业务代码。
- 管理 API 错误保留 Twirp code/message，不包装成"服务不可用"。

### 5. Good/Base/Bad Cases

- Good：`client.Room().CreateRoom(ctx, req)` 直接使用 SDK；`lksdk.NewRoom(cb)` + `JoinWithContext` 加入房间；`livekitx.NewJoinToken(opts)` 签发 token。
- Base：同一 `Client` 多次调用 `Room()` 获取 RoomService（每次返回同一实例）。
- Bad：试图调用 `client.CreateRoom()`、`client.JoinRoom()` 等已删除的 wrapper 方法。

### 6. Tests Required

- 单元（`api_test.go`）：Twirp mock 断言管理 API 请求路由与认证。
- `token_test.go`：`NewJoinToken` 用 `auth.ParseAPIToken` + `verifier.Verify(secret)` 断言 grant。
- `webhook_test.go`：`NewWebhookKeyProvider` 验真/验假。
- 集成（`integration_test.go`，`LIVEKITX_INTEGRATION=1` 开启）：用 `joinRoomHelper` 直接调用 SDK 加入房间，断言原生回调、聊天、RPC、SendData、踢人。
- 断言点：回调调用次数与字段、错误类型可 `errors.Is`、管理请求体字段。

### 7. Wrong vs Correct

#### Wrong

```go
// 1. 调用已删除的 wrapper 方法
client.CreateRoom(ctx, &livekit.CreateRoomRequest{Name: "room"})
client.JoinRoom(ctx, "room", "user")
client.DeleteRoom(ctx, "room")
client.SendData(ctx, "room", "topic", payload)

// 2. 调用已删除的 JoinToken 方法
token, _ := client.JoinToken("room", "user", time.Hour)

// 3. 使用已删除的 WithCallback/WithConnectOption
room, _ := client.JoinRoom(ctx, "room", "user", livekitx.WithCallback(cb))
```

#### Correct

```go
// 1. 直接使用 SDK
roomSvc := client.Room()
roomSvc.CreateRoom(ctx, &livekit.CreateRoomRequest{Name: "room"})
roomSvc.DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: "room"})
roomSvc.SendData(ctx, &livekit.SendDataRequest{Room: "room", Topic: &topic, Data: payload})

// 2. 用函数签发 token
token, _ := livekitx.NewJoinToken(livekitx.JoinTokenOptions{...})

// 3. 用 SDK 原生方式加入房间
room := lksdk.NewRoom(&lksdk.RoomCallback{...})
room.JoinWithContext(ctx, url, lksdk.ConnectInfo{...})
```
