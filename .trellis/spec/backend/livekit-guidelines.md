# LiveKit 视频会议对接规范

## 适用范围

修改 `common/livekitx`、`app/meeting`、LiveKit JWT/Webhook/Twirp、房间/参与者、Egress、Ingress、SIP 或 Agent 调度时读取。API 版本基线见 [对接指南](../../../docs/livekit-integration-guide.md)，本规范不把本地开发工作树当作稳定版本。

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

## 会议业务服务（app/live）规范

适用：`app/live`（Live 会议 gRPC 业务服务）及其网关（livegtw）。机制层约定见上文。

### 1. Webhook 链路（验签 → 原始字节 → SDK 对象处理）

- 链路：LiveKit 推送 → livegtw 用 `webhook.ReceiveWebhookEvent` 验签（失败 401，不调用业务）→ 把 `*livekit.WebhookEvent` **proto 序列化后的原始字节**经 `WebhookNotify(WebhookNotifyReq{Data: bytes})` 透传给 live 服务 → live 服务 `proto.Unmarshal` 解析为 SDK 对象后处理业务。禁止在 proto 中做字段扁平化（eventId/eventType/roomName/...），会丢失 SDK 结构。
- **不做 TTL 幂等**：LiveKit 事件可能重复、迟到、补发（对账闭环需要重放），而处理操作本身幂等（参会记录 `Where+Assign+FirstOrCreate` upsert、会议状态仅 active→ended 流转、未知会议安全忽略），重复/补发重放结果一致。禁止再加 event ID 去重键阻碍补发。
- **事件全 case**：`room_finished`/`participant_joined`/`participant_left` 处理；`room_started` 无需处理（创建即 active）；`participant_connection_aborted`/`track_published`/`track_unpublished`/`egress_started`/`egress_updated`/`egress_ended`/`ingress_started`/`ingress_ended` 建 case 标注 TODO 并记日志；未知事件安全忽略。禁止 default 静默吞掉已知事件。

### 2. 会议业务约定

- 会议号：`tool.IdUtil.NextId("M", "live")`（Redis 序号 + 日期；`category` 参数按业务域隔离，防止与其他服务撞号）。
- Redis key：统一以业务域前缀开头（如 `live:lock:meeting:*`、`live:outId_M`），最前段系统 key 由 redis 配置自动附加。
- 入会 token：`CanPublish`/`CanSubscribe`/`CanPublishData` 全开（否则参与者无法发布媒体/发 Data）；业务直接调用 `livekitx.NewJoinToken(opts)` 签发。
- 时间输出：RPC 出参时间统一 `carbonx.FormatDateTimeOrEmpty`/`FormatNullDateTime`（`yyyy-MM-dd HH:mm:ss` 字符串），不使用时间戳。
- 创建人/更新人/机构：从 gRPC metadata 取（`grpcx.LoggerInterceptor` 注入 `x-user-id` 等 → `authctx.GetUserId`），proto 入参不传；模型保留 `create_user`/`update_user`/`dept_code`。
- 并发控制：`redis.NewRedisLock(r, key)` 直接构造（go-zero RedisLock，Lua 原子 + `SetExpire` TTL 自动释放），`AcquireCtx` 返回 `(bool, error)` 区分"未获得锁"与"Redis 错误"。

### 3. 模型风格

新业务表（会议等）按 `app/trigger/model/gormmodel` 的 plan 系列风格：`gormx.LegacyStringBaseModel`（string 主键 + create_time/update_time + is_deleted 软删）+ `gormx.VersionMixin` + `CreateUser`/`UpdateUser`/`DeptCode`（sql.NullString）+ 可空字段用 `sql.NullString`/`sql.NullTime` + `int` 状态 + 索引名 `idx_<表名>_<字段>`。

### 4. 业务错误码

业务错误统一 extproto + `tool.NewErrorByPbCode`/`NewErrorByPbCodeWrap`（reason=六位错误码，HTTP 自动映射），禁止裸 `status.Error`/`status.Errorf`。常用映射：参数 `101101`、记录不存在 `102102`、记录已存在 `102103`、缓存/Redis `103101`、未认证 `104101`、业务状态不允许 `105102`、重复操作 `105103`、DB `102101`、第三方（LiveKit）`106102`。

## 版本与依赖

- 审计基线为 LiveKit Server `v1.13.6`、`github.com/livekit/server-sdk-go/v2 v2.18.1`。
- Protocol 版本由选定 SDK 的 `go.mod` 解析；不得手工猜测或截断 pseudo-version。
- 升级后重新执行核心示例的 `go mod tidy`、`go test ./...`，并检查生成字段、枚举、oneof、配置和官方链接。

## Token 与授权

- 业务服务使用 `auth.NewAccessToken` 签发短期、指定房间、最小权限的 join token；管理 token 不返回客户端。`livekitx` 提供 `NewJoinToken(opts JoinTokenOptions)`（自定义 grant 场景）。
- `VideoGrant` 的房间、创建、管理、录制、Ingress、Agent、发布/订阅和数据字段必须以锁定 Protocol 生成类型核对；`CanPublish`/`CanSubscribe` 是 `*bool`，用 `grant.SetCanPublish(bool)` 设置，不能直接结构体字面量赋值。
- 聊天/Data 链路 Token 必须设置 `CanPublishData`（`grant.SetCanPublishData(true)`），否则 Data 发布被服务端拒绝。
- SIP 使用 `SIPGrant`，不得写成 `VideoGrant` 字段。
- API key/secret、Webhook signing key、SIP 密码只能来自安全配置。
- `auth.NewAccessToken(key, secret).ToJWT()` 在 key 为空时报错；Token 测试断言用 `auth.ParseAPIToken(token)` + `verifier.Verify(secret)`，不要在测试中输出 token/secret。

## 状态与 Webhook

LiveKit 拥有房间、参与者、轨道和录制运行态；zero-service 拥有用户、会议单据、角色授权、审计和业务状态。Webhook 只触发同步或对账，不授予权限，也不替代管理 API。

## 实时连接与回调

- 业务直接使用 SDK 原生 `*lksdk.RoomCallback`，通过 `lksdk.NewRoom(callback)` 构造，`room.JoinWithContext` 加入。
- `JoinWithContext` 成功返回即"已连接"（SDK 原生无 connected 回调）；断开由业务调 `room.Disconnect()`。
- 聊天识别是业务职责：`*livekit.ChatMessage`（文本）与 `UserDataPacket`（业务自定义 topic）都在 `OnDataPacket` 到达。
- 服务端 RPC（`client.Room().PerformRpc()`）是服务端→单个客户端的定向请求-响应。
- SendData 与 PerformRpc 选型：只通知不关心结果→ `SendData`；要客户端执行并返回结果→ `PerformRpc`。
- 管理 API 方法名以锁定 protocol 源码核对：静音是 `MutePublishedTrack`；结束会议统一用 `DeleteRoom`。

## 错误、生命周期与反模式

- 每个外部调用设置 context 超时，保留 Twirp code/message。
- ServiceContext 复用 client，关闭服务时释放。
- 不在请求中重复创建 API client，不硬编码 secret，不在公共包固化业务 grant。

## 部署与验证

生产使用可信 TLS 的 HTTPS/WSS；WebRTC UDP/TCP、ICE、NAT、公网地址、TURN、WebSocket、Redis、多节点负载均衡和独立 Egress/Ingress/SIP/Agent 进程按官方当前配置逐项验收。开放 API 端口不等于媒体链路可用。

验证至少包括：临时 module 锁定稳定 SDK 后 `go mod tidy` 与 `go test ./...`；Token/Webhook/API 的成功、权限失败、超时、重复和边界测试；`git diff --check` 以及文档链接、版本、secret 和个人路径扫描。没有 Server、凭据、浏览器媒体、TURN、Redis 集群或外部运营商时，必须报告未完成，不得声称端到端通过。

`common/livekitx` 本地集成测试以 `LIVEKITX_INTEGRATION=1` 显式开启（默认跳过，不影响普通单测），连接 `http://127.0.0.1:7880`、`devkey`/`secret`；覆盖房间生命周期、SDK 原生入会、原生回调（入会/聊天双路径/RPC 往返/断开原因）、`SendData` 富媒体投递。Egress/Ingress/SIP/Agent 与 Webhook 服务端推送不做真实端到端断言，只能标注环境前置条件。
