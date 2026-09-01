# common/livekitx

`livekitx` 是 LiveKit Server SDK v2.18.1 的机制层封装。它复用 SDK 的 Protocol 请求/响应类型，提供统一配置、Twirp 管理 API、Token、Room API（加入/创建/创建并加入/删除房间/踢人/邀请/音频视频静音/富媒体发送，实时事件由业务用 SDK 原生 `RoomCallback` 处理）和 Webhook 验签 KeyProvider。包不包含业务会议单据、数据库、Redis、消息历史或授权策略，也不包含任何事件分发/Hook 注册机制或连接状态存储。

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

`New` 成功后默认把 LiveKit SDK/protocol 的全局日志接到 go-zero `logx`（包级全局行为，幂等，业务可后续自行调用 SDK `SetLogger` 覆盖）。logx 没有 warn 级别，SDK 的 warn 按 info 输出、err 附加为 `error` 字段。

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

`client.JoinToken(roomName, identity, validFor)` 使用 client 配置的管理凭据签发指定房间和身份的入会 token，供业务在用户鉴权后发放给客户端 App，由客户端直接连接 LiveKit：

```go
// 用户鉴权通过后发放；token 只返回给该用户，不写入日志。
token, err := client.JoinToken("demo", "user-a", 2*time.Hour)
```

`NewSIPToken` 只支持 `SIPGrant.Admin` 和 `SIPGrant.Call`，用于 SIP 管理或呼叫。API 管理凭据、参与者 Token、Webhook signing key、SIP 凭据和 Agent worker 凭据必须分开保管。

## Room API 与原生回调

`livekitx` 的 Room API 提供加入/创建/创建并加入/删除房间与会议管理（踢人/邀请/音频视频静音/富媒体发送），实时连接全部返回 SDK 原生 `*lksdk.Room`：

```go
// 加入已有房间；join token 由 client 的 APIKey/APISecret 内部生成
//（SDK JoinWithContext + ConnectInfo），业务只传房间名与身份。
// opts 是 JoinRoomOption：WithCallback 覆盖本次回调、WithConnectOption
// 透传 SDK 连接选项（如 WithAutoSubscribe）。
room, err := client.JoinRoom(ctx, roomName, identity, opts...)
// 经管理 API 创建房间；返回协议房间信息。
info, err := client.CreateRoom(ctx, name)
// 先创建房间再加入；加入失败不自动删除房间（房间可能已有其他参与者），
// 由业务决定清理。
room, err := client.CreateAndJoinRoom(ctx, roomName, identity, opts...)
// 经管理 API 删除房间；断开房间内全部参与者并结束会议
//（结束会议统一使用本方法）。
err := client.DeleteRoom(ctx, name)
// 经管理 API 将指定参与者移出房间（踢人）；被移出者收到 PARTICIPANT_REMOVED。
err := client.RemoveParticipant(ctx, roomName, identity)
// 经管理 API 向房间内指定参与者发送一条可靠 Data 消息（邀请通知）；
// topic/payload 由业务定义，接收方在 OnDataPacket 按 topic 识别。
err := client.InviteParticipant(ctx, roomName, identity, topic, payload)
// 经管理 API 静音/取消静音指定参与者的全部音频轨道；
// 参与者不在房间或没有音频轨道时静默成功。
err := client.MuteParticipant(ctx, roomName, identity, muted)
// 经管理 API 关闭/恢复指定参与者的全部视频轨道（与 MuteParticipant 的
// 音频静音对称）；参与者不在房间或没有视频轨道时静默成功。
err := client.MuteParticipantVideo(ctx, roomName, identity, muted)
// 经管理 API 发送富媒体聊天（语音/图片等二进制数据）：destinations 为空
// 广播给全体参与者，否则定向发送；topic/payload 由业务约定，接收方
// OnDataPacket 按 topic 识别 UserDataPacket。
err := client.SendData(ctx, roomName, topic, payload, destinations...)
```

- **`JoinRoom`/`CreateAndJoinRoom` 成功返回即表示连接已建立**（SDK 原生没有 connected 回调）。连接建立期间 context 控制连接超时和取消。
- 回调只来自 `JoinRoom`/`CreateAndJoinRoom` 的 `WithCallback(callback *lksdk.RoomCallback)`（不传则 SDK 使用默认空回调，nil 也允许）；`WithConnectOption` 透传 SDK 连接选项；需要不同回调时为每次加入分别指定。
- 返回的 `*lksdk.Room` 是 SDK 原生对象：断开由业务调用 `room.Disconnect()`（`room.DisconnectWithReason(reason)` 可携带原因）；SDK Room 指针只在本进程保存，`Client.Close()` 只标记关闭，不管理连接资源。
- callback **原样透传给 SDK**：livekitx 不构造回调桥、不合并字段、不合成任何事件。入会、离会、轨道、Data、聊天、RPC、断开等全部事件都由 SDK 直接触发 callback 中的字段，业务在回调里自行处理。

```go
package meeting

import (
    "context"
    lksdk "github.com/livekit/server-sdk-go/v2"
    "zero-service/common/livekitx"
)

// client 启动时构造一次。
client, _ := livekitx.New(
    livekitx.WithURL("http://127.0.0.1:7880"),
    livekitx.WithAPIKey("devkey", "secret"),
)

// join 建立连接：callback 用 WithCallback 指定并原样透传给 SDK
//（不传则 SDK 使用默认空回调），join token 由 client 内部生成。
func join(ctx context.Context, client *livekitx.Client, room, identity string) (*lksdk.Room, error) {
    return client.JoinRoom(ctx, room, identity,
        livekitx.WithCallback(&lksdk.RoomCallback{
            OnParticipantConnected: func(rp *lksdk.RemoteParticipant) { /* 参会名单更新 */ },
            ParticipantCallback: lksdk.ParticipantCallback{
                OnDataPacket: func(data lksdk.DataPacket, params lksdk.DataReceiveParams) { /* 聊天/RPC 数据 */ },
            },
        }),
    )
}
```

**每个回调场景的签名、字段含义、触发来源与代码示例见 [docs/livekit-callbacks-guide.md](../docs/livekit-callbacks-guide.md)**。

## Webhook

`NewWebhookKeyProvider(signingKey string)` 返回校验 LiveKit Webhook 签名所需的 `auth.KeyProvider`，业务自行调用 SDK 的 `webhook.ReceiveWebhookEvent`：

```go
import "github.com/livekit/protocol/webhook"

// 收到 LiveKit 服务端推送的 HTTP 请求后：
event, err := webhook.ReceiveWebhookEvent(req, livekitx.NewWebhookKeyProvider(cfg.WebhookKey))
if err != nil {
    // 验签失败直接返回签名错误，不调用任何业务代码。
    return err
}
// 持久化 event.GetId() 做幂等；处理重试、迟到、乱序、未知事件和对账。
```

签名方式与 livekit-server 一致：body SHA256 放入 API token 的 Sha256 claim。KeyProvider 对任意 key claim 返回同一个 signing key（服务器用自己的 API key 签名，业务只持有 secret）。业务必须自行持久化 event ID 做幂等，本包不承诺 Exactly Once，也不内置永久去重；未知事件保留 event ID/type 安全交付，由业务决定处理策略。

## Data、聊天与 RPC

实时链路的 Data、聊天与 RPC 全部使用 SDK 原生方法（`*lksdk.Room` / `room.LocalParticipant`）：

- 发送 Data：`LocalParticipant.PublishData(payload, lksdk.WithDataPublishTopic(...), ...)`；发送任意 DataPacket（含 `*livekit.ChatMessage`）：`LocalParticipant.PublishDataPacket(pck, ...)`。
- 发送聊天：`LocalParticipant.PublishDataPacket(lksdk.ChatMessage(time.Now(), text))`；接收方在 `OnDataPacket` 回调中识别 `*livekit.ChatMessage` 数据包。
- 发送富媒体（语音/图片等）：`ChatMessage` 只支持文本，二进制内容用服务端管理 API `client.SendData(ctx, room, topic, payload, destinations...)` 广播/定向投递，接收方在 `OnDataPacket` 识别 `UserDataPacket`（topic 由业务约定）；也可用 SDK 原生 `PublishData`/`PublishDataPacket` 从参与者侧发送。
- RPC：`room.RegisterRpcCtxMethod(method, handler)` 注册、`room.UnregisterRpcMethod(method)` 注销、`room.LocalParticipant.PerformRpc(params)` 调用。SDK v2.18.1 的 `PerformRpc` 不接收 context，通过 `params.ResponseTimeout` 控制时限（<8000ms 会被钳制到 8000ms，默认 15000ms）；未注册方法由 SDK 自动返回 UnsupportedMethod。
- 聊天接收双路径：SDK 原生 `*livekit.ChatMessage`，或 topic 为业务自定义约定的 `UserDataPacket`；业务在 `OnDataPacket` 内自行识别（字段差异见 [docs/livekit-callbacks-guide.md](../docs/livekit-callbacks-guide.md) 的"聊天消息识别"一节）。

发送聊天需要 Token 带 `CanPublishData: true`。可靠消息由 SDK/底层 DataChannel 提供传输语义，不等于业务持久化或业务确认。RPC response 是请求级回执，不代表业务落库成功。

## 错误、关闭与安全

配置错误、空 Token 和关闭后调用有可判断的本地错误（`ErrInvalidConfig`、`ErrClosed`、`ErrInvalidTokenOptions`）；SDK 错误使用 `%w` 保留原始语义。所有网络请求应使用非 nil context。`Client.Close` 可重复调用。不要记录 API secret、signing key、Token 或完整 Authorization。

## 本地测试

普通测试不会依赖 LiveKit 服务。启动 `livekit-server --dev` 后运行真实链路：

```text
LIVEKITX_INTEGRATION=1 go test ./common/livekitx -run TestLiveKitDevServer -v
go test ./common/livekitx/...
go test -race ./common/livekitx/...
go vet ./common/livekitx/...
```

集成测试默认使用 `http://127.0.0.1:7880`、`devkey`、`secret`，也支持 `LIVEKIT_URL`、`LIVEKIT_API_KEY`、`LIVEKIT_API_SECRET`。它验证真实房间创建/加入/删除（Room API）、原生回调收到入会/离会、SDK 原生聊天与自定义 topic 的 UserData 聊天、RPC 真实往返、房间内可靠 Data 投递（InviteParticipant）、踢人（RemoveParticipant）与断开原因、DeleteRoom 结束会议；未开启时跳过。Egress、Ingress、SIP、Agent 依赖额外服务，dev server 不提供可重复的真实环境，不能伪造为通过；Webhook 需要服务端配置推送地址，dev server 不推送，签名与分发逻辑由单元测试覆盖。实时媒体发布/订阅需要编解码与媒体环境，回调/Data/RPC 测试与媒体编解码测试分离。

排障时先确认服务地址、API 凭据、context deadline、服务端日志和 Twirp error code；再检查参与者 Token 的 room/identity/grant 是否匹配、聊天 Token 是否带 `CanPublishData`。