# common/livekitx

`livekitx` 是 LiveKit Server SDK v2.18.1 的机制层封装。它复用 SDK 的 Protocol 请求/响应类型，提供统一配置、Twirp 管理 API、Token、实时连接、Data/RPC、Webhook 验签和可注销的 typed Hook。包不包含业务会议单据、数据库、Redis、消息历史或授权策略。

## 初始化与依赖

需要 `github.com/livekit/server-sdk-go/v2 v2.18.1`、LiveKit API URL、管理 API key/secret。请求超时必须由业务为每次调用传入的 `context.Context` 控制，本包不会派生或覆盖 context。

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "zero-service/common/livekitx"
)

func main() {
    httpClient := &http.Client{}
    client, err := livekitx.New(
        livekitx.WithURL("http://127.0.0.1:7880"),
        livekitx.WithAPIKey("devkey", "secret"),
        livekitx.WithHTTPClient(httpClient),
    )
    if err != nil { panic(err) }
    defer client.Close()
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    rooms, err := client.API().Room().ListRooms(ctx, nil)
    fmt.Println(rooms, err)
}
```

也可用 `WithHTTPService(go-zero httpc.Service)` 注入业务已有的 go-zero 客户端。SDK v2.18.1 的 `LiveKitAPI` 没有 HTTP client 注入选项，因此本包直接使用 SDK 生成的 Protocol Twirp client；注入的 `*http.Client` 或 `httpc.Service` 会实际执行请求。两种注入方式互斥，未注入时使用标准客户端。注入对象由调用方拥有，`Close` 不会关闭它。

## 管理 API

`client.API()` 返回复用的管理 API 聚合入口，方法返回 SDK 原生类型：

- `Room()`：创建、查询、删除房间；列出、查询、移除参与者；更新参与者 metadata、订阅和轨道静音；发送服务端 Data；更新房间 metadata；服务端 RPC。
- `Egress()`：录制、导出、停止和查询 Egress。需要 Egress 服务、对象存储或编码基础设施。
- `Ingress()`：创建、更新、查询、删除 Ingress。需要 Ingress 服务及输入源。
- `SIP()`：SIP trunk、dispatch rule、电话参与者和 SIP 管理。需要 SIP 服务及运营商配置。
- `AgentDispatch()`：向房间或参与者派发 Agent。需要 Agent worker。

示例：`client.API().Room().CreateRoom(ctx, &livekit.CreateRoomRequest{Name: "demo"})`。错误保留底层 Twirp/SDK error chain，可用 `errors.As` 判断 SDK ServerError。

**能力边界**：Connector、AgentSimulation 和 Cloud Agents 不属于本包 MVP。它们依赖额外的云服务或外部基础设施，未来基于相同认证/HTTP 基础设施以独立扩展包提供，不进入核心构造路径。本包不承担 LiveKit Cloud Agent 的源代码上传、构建、部署、发布或日志管理。

## Token

`NewJoinToken` 支持 Video grant 的 `RoomJoin`、指定 `Room`、`CanPublish`、`CanSubscribe`、`CanPublishData`，并设置 identity、name 和有效期。它适用于最小权限的普通参与者入会；聊天/Data 链路需要 `CanPublishData: true`。视频授权还可由业务直接使用 SDK `auth.VideoGrant` 增加 `RoomCreate`、`RoomList`、`RoomRecord`、`RoomAdmin`、`RoomConfiguration`、`CanPublishSources`、`CanUpdateMetadata`、`Hidden`、`Recorder`、`Agent` 等权限，但管理权限不应交给终端参与者。

`NewSIPToken` 只支持 `SIPGrant.Admin` 和 `SIPGrant.Call`，用于 SIP 管理或呼叫。API 管理凭据、参与者 Token、Webhook signing key、SIP 凭据和 Agent worker 凭据必须分开保管。

## 实时连接与回调桥

业务生成指定房间和 identity 的 Join Token 后调用 `Connect(ctx, token, callback, opts...)`。连接建立期间 context 控制连接超时和取消；成功后得到 `RealtimeRoom`，通过 `Room()` 获取 SDK 原生 Room，通过 `Close()` 幂等断开。`Client.Close()` 会关闭所有本进程连接、停止全部 Hook，并关闭 Store。SDK Room 指针只在本进程保存。

`Connect` 内部构造**默认回调桥**：SDK 的 Room/Participant/Track/Data 回调被桥接到下表的 typed Hook；连接建立成功后还会合成分发一次 `RoomConnectionEvent{Status: connected}`，便于业务在入会时同步会议状态。

调用方传入的 `callback *lksdk.RoomCallback` 按 SDK Merge 语义叠加：**非 nil 字段覆盖内置桥接的相同字段，nil 字段保留内置桥接**。传 `nil` 或空 `&lksdk.RoomCallback{}` 时完整保留内置桥接。

```go
package meeting

import (
    "context"
    lksdk "github.com/livekit/server-sdk-go/v2"
    "zero-service/common/livekitx"
)

// join 建立连接：传 nil 时完整保留内置回调桥，业务通过 client.OnXXX 注册 Hook。
func join(ctx context.Context, client *livekitx.Client, token string) (*livekitx.RealtimeRoom, error) {
    return client.Connect(ctx, token, nil)
}

// joinWithCustomCallback 叠加自定义回调：只覆盖 OnParticipantConnected，
// 其余字段仍走内置桥接。
func joinWithCustomCallback(ctx context.Context, client *livekitx.Client, token string) (*livekitx.RealtimeRoom, error) {
    return client.Connect(ctx, token, &lksdk.RoomCallback{
        OnParticipantConnected: func(rp *lksdk.RemoteParticipant) { /* 自定义处理 */ },
    })
}
```

注意：自定义回调覆盖桥接字段后，该字段不再触发对应 Hook（例如覆盖 `OnDataPacket` 后 `OnChatMessage` 不再自动触发）。覆盖 `OnDisconnected`/`OnDisconnectedWithReason` 时，内置的 Store 索引清理和 Hook context 取消也会被跳过，业务需要自行调用 `RealtimeRoom.Close()` 完成清理。

## Hook 总览

所有 Hook 均为实例级注册，返回 `Subscription`（`Unsubscribe()` 幂等、并发安全）。同一事件按注册顺序快照执行；handler 返回的错误被聚合返回；handler panic 被 recover 并转换为 `*HookPanicError`，不会穿透 SDK 读循环或 HTTP 边界。`Client.Close()` 后不再分发任何事件，注册返回空订阅。

| 注册 API | 事件类型 | 关键字段 | 触发来源 | 业务典型用途 |
|---|---|---|---|---|
| `OnRoomConnection` | `RoomConnectionEvent` | RoomName、Status（connected/reconnecting/reconnected/disconnected）、Reason、ProtocolReason | Connect 合成 + SDK `OnDisconnected`/`OnReconnecting`/`OnReconnected` 回调 | 会议状态同步、活跃会议计数、断线告警、对账 |
| `OnRoomMetadata` | `RoomMetadataEvent` | RoomName、Metadata | SDK `OnRoomMetadataChanged` 回调 | 会议主题/配置同步、审核留痕 |
| `OnRoomRecording` | `RoomRecordingEvent` | RoomName、IsRecording | SDK `OnRecordingStatusChanged` 回调 | 录制状态展示、计费、审计 |
| `OnRoomMoved` | `RoomMovedEvent` | RoomName、RoomSID、Token | SDK `OnRoomMovedWithSID` 回调 | 房间迁移感知、会议状态迁移 |
| `OnActiveSpeakers` | `ActiveSpeakersEvent` | RoomName、Participants | SDK `OnActiveSpeakersChanged` 回调 | 发言状态展示、会控提示 |
| `OnParticipantConnected` | `ParticipantConnectedEvent` | RoomName、Participant | SDK `OnParticipantConnected` 回调 | 参会名单更新、入会通知、落库 |
| `OnParticipantDisconnected` | `ParticipantDisconnectedEvent` | RoomName、Participant | SDK `OnParticipantDisconnected` 回调 | 离会名单更新、时长统计、踢人后处理 |
| `OnParticipantMetadata` | `ParticipantMetadataEvent` | RoomName、Participant、OldMetadata | SDK `OnMetadataChanged` 回调 | 用户资料同步、审核留痕 |
| `OnParticipantAttributes` | `ParticipantAttributesEvent` | RoomName、Participant、Changed | SDK `OnAttributesChanged` 回调 | 状态属性同步、会控指令回执 |
| `OnParticipantSpeaking` | `ParticipantSpeakingEvent` | RoomName、Participant、IsSpeaking | SDK `OnIsSpeakingChanged` 回调 | 麦位状态、发言统计 |
| `OnConnectionQuality` | `ConnectionQualityEvent` | RoomName、Participant、Update | SDK `OnConnectionQualityChanged` 回调 | 弱网告警、质量监控 |
| `OnTrack` | `TrackEvent` | RoomName、Kind（published/unpublished/subscribed/unsubscribed/subscription_failed/muted/unmuted/local_*）、Publication、Track、TrackSID | SDK `OnTrackPublished`/`OnTrackUnsubscribed`/`OnTrackMuted`/`OnTrackSubscriptionFailed`/`OnLocalTrack*` 回调 | 轨道状态同步、录屏/共享检测、订阅失败重试 |
| `OnData` | `DataEvent` | RoomName、SenderIdentity、SenderSID、Topic、Payload、Packet | SDK `OnDataPacket` 回调（任意 DataPacket） | 数据链路审计、业务自定义数据分发 |
| `OnChatMessage` | `ChatMessageEvent` | RoomName、SenderID、MessageID、Topic、Text、Payload、Timestamp、Deleted、Generated | SDK `OnDataPacket` 回调中的 `*livekit.ChatMessage` 或 topic 为 `ChatTopic` 的 `UserDataPacket` | 聊天落库、未读数、敏感词审核、消息通知、历史查询 |
| `OnRPCRequest` | `RPCRequestEvent` | RoomName、CallerIdentity、RequestID、Method、Payload | 经 `RealtimeRoom.RegisterRPC` 注册的方法收到请求时（SDK `OnRpcRequest` 路径） | RPC 审计、业务鉴权前置检查、调用计数 |
| `OnWebhook` | `WebhookEvent`（SDK 原生类型） | Event、Id、CreatedAt、Room、Participant、Track、EgressInfo、IngressInfo | `ReceiveWebhook` 验签成功后分发（服务端 HTTP 推送） | 会议状态落库、对账、未知事件告警、Egress/Ingress/SIP 状态同步 |

所有事件都保留可审计字段（房间名、参与者 identity/SID、track SID、topic、payload、事件 ID/时间）；公共事件类型不包含业务 ID 或业务角色，业务在 handler 中自行映射。

## Webhook

`OnWebhook(handler)` 注册可注销的 Webhook Hook；`ReceiveWebhook(ctx, request, signingKey, handler)` 校验 `Authorization` 与原始 body 的签名，**验签失败不会调用任何 handler**，成功后按注册顺序分发到全部 Hook。`handler` 参数兼容旧版一次性调用方式：非 nil 时在 Hook 之后执行。

```go
client.OnWebhook(func(ctx context.Context, event *livekitx.WebhookEvent) error {
    // 持久化 event.GetId() 做幂等；处理重试、迟到、乱序、未知事件和对账。
    // 例如 room_started/room_finished 更新会议状态，participant_joined 落库。
    return nil
})
// 收到 HTTP 请求后：
err := client.ReceiveWebhook(ctx, httpRequest, webhookSigningKey, nil)
```

签名方式与 livekit-server 一致：body SHA256 放入 API token 的 Sha256 claim。本包只接收 signing key（secret），API key claim 仅作标识。业务必须自行持久化 event ID 做幂等，本包不承诺 Exactly Once，也不内置永久去重；未知事件保留 event ID/type 安全交付给 handler，由业务决定处理策略。

## Data、聊天与 RPC

`PublishData(payload, lksdk.DataPublishOption...)` 原样转发 SDK DataChannel；选项决定可靠/不可靠、topic 和目标 identity。`PublishDataPacket(pck, ...)` 可发送任意 SDK DataPacket（含 `*livekit.ChatMessage`）；`PublishChatMessage(text, ...)` 是发送 SDK 原生聊天消息的便捷入口。可靠消息由 SDK/底层 DataChannel 提供传输语义，不等于业务持久化或业务确认。

聊天接收有两种路径，都触发 `OnChatMessage`：

- SDK 原生：`PublishChatMessage` / 任意 `*livekit.ChatMessage` 数据包；
- 兼容约定：topic 为 `ChatTopic`（`"chat"`）的 `UserDataPacket`。

`OnChatMessage` 只负责业务 Hook 边界，消息历史、未读数、审核和会话归属由业务实现。发送聊天需要 Token 带 `CanPublishData: true`。

`RegisterRPC(method, handler)` 注册 SDK RPC handler，请求到达时先触发 `OnRPCRequest` Hook 再调用业务 handler，错误返回远端；`UnregisterRPC(method)` 注销已注册的方法。`PerformRPC(params)` 返回 SDK response 或远端错误，SDK v2.18.1 通过 `params.ResponseTimeout` 控制 RPC 时限且没有 context 参数。RPC response 是请求级回执，不代表业务落库成功。未注册方法由 SDK 自动返回 UnsupportedMethod，不触发 Hook。

## Store 与分布式部署

`Store` 保存 `ConnectionState` 索引：`NodeID`、room name、identity、session ID、`ConnectionStatus`、更新时间和过期时间等可序列化字段，不保存 `*RealtimeRoom`、WebRTC peer connection 或其他 SDK 指针。通过 `client.Store()` 读取当前 Store，`client.Config()` 读取构造时配置副本。

默认 `NewMemoryStore()` 适合单测和单节点。`WithStore` 可注入业务实现，例如用 Redis/DB 保存索引。Redis/DB 的序列化、TTL、节点归属、抢占、租约、选主和分布式锁都由业务决定；本包只在连接建立/关闭（含 SDK 断开回调）时调用 `Put`/`Delete`，读取过期记录时删除。多节点部署必须让业务 Store 设计节点归属和失效策略，不能把内存 Store 当成共享状态。

## 错误、关闭与安全

配置错误、空 Token、空 Store 状态、关闭后调用和 Hook panic 有可判断的本地错误（`ErrInvalidConfig`、`ErrClosed`、`ErrNilRoom`、`ErrInvalidTokenOptions`、`*HookPanicError`）；SDK 错误使用 `%w` 保留原始语义。所有网络请求应使用非 nil context。`Subscription.Unsubscribe`、`RealtimeRoom.Close`、`Client.Close` 和 Store `Close` 都可重复调用。不要记录 API secret、signing key、Token 或完整 Authorization。

## 本地测试

普通测试不会依赖 LiveKit 服务。启动 `livekit-server --dev` 后运行真实链路：

```text
LIVEKITX_INTEGRATION=1 go test ./common/livekitx -run TestLiveKitDevServer -v
go test ./common/livekitx/...
go test -race ./common/livekitx/...
go vet ./common/livekitx/...
```

集成测试默认使用 `http://127.0.0.1:7880`、`devkey`、`secret`，也支持 `LIVEKIT_URL`、`LIVEKIT_API_KEY`、`LIVEKIT_API_SECRET`。它验证真实房间创建/查询/删除、参与者入会 Hook、SDK 原生聊天与 ChatTopic 数据聊天、RPC 请求 Hook 与真实调用往返、断开原因；未开启时跳过。Egress、Ingress、SIP、Agent 依赖额外服务，dev server 不提供可重复的真实环境，不能伪造为通过；Webhook 需要服务端配置推送地址，dev server 不推送，签名与分发逻辑由单元测试覆盖。实时媒体发布/订阅需要编解码与媒体环境，核心 Hook/Data/RPC 测试与媒体编解码测试分离。

排障时先确认服务地址、API 凭据、context deadline、服务端日志和 Twirp error code；再检查参与者 Token 的 room/identity/grant 是否匹配、聊天 Token 是否带 `CanPublishData`。