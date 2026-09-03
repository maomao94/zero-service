# LiveKit 对接指南

本文以 2026-08-31 可核验的稳定发布为基线。稳定发布、用于本地源码检查的开发提交和 Protocol 依赖必须分开记录，不能把工作树的 pseudo-version 称为稳定版本。

## 版本基线与证据

| 项目 | 稳定版本 | 来源 | 本地验证状态 |
| --- | --- | --- | --- |
| LiveKit Server | `v1.13.6` | [GitHub release/tag](https://github.com/livekit/livekit/releases/tag/v1.13.6) | 本地工作树 `460014023fe5df504bdfff51cbb47ddbfdd1e6fd`，`git describe` 为 `v1.13.6-12-g46001402` |
| Go Server SDK | `v2.18.1` | [Go module/tag](https://github.com/livekit/server-sdk-go/releases/tag/v2.18.1) | 本地工作树 `a41bea984454ccc958c238bde111022917806aaa`，`git describe` 为 `v2.18.1-28-ga41bea9` |
| Protocol | 稳定 SDK `v2.18.1` tag 的 `go.mod` 为 `v1.49.0` | [SDK v2.18.1 go.mod](https://github.com/livekit/server-sdk-go/blob/v2.18.1/go.mod) | 本地开发工作树解析为 `v1.50.5-0.20260829123501-a469dd43727b`；这是开发依赖证据，不是稳定基线 |
| Go | 稳定 SDK `v2.18.1` tag 的 `go` 指令为 `1.26` | [Go SDK go.mod](https://github.com/livekit/server-sdk-go/blob/v2.18.1/go.mod) | 本地开发工作树指令为 `1.26.3`；业务构建仍应以项目工具链和依赖实际要求为准 |

查询日期为 2026-08-31。上述本地提交包含稳定 tag 之后的开发提交，因此只能作为源码/API 编译证据，不能替代稳定版本。升级时重新执行 `go list -m -json`、示例编译、配置检查和全文扫描。

## 边界与学习顺序

建议先理解 Server、客户端 SDK、Server SDK、实时参与者和 Egress/Ingress/SIP/Agent 的职责，再完成“签发 join token、客户端加入、建立媒体连接”的最小闭环，最后逐项接入 Webhook 和独立服务。

| 组件 | 负责 | 边界 |
| --- | --- | --- |
| LiveKit Server | 信令、SFU 媒体转发和房间实时状态 | 不负责 zero-service 的用户、会议单据和业务授权 |
| 客户端 SDK | 采集、发布、订阅和播放媒体 | 不得持有 API secret |
| Go Server SDK | 管理 API；也可作为实时参与者连接房间 | 不替代业务数据库，也不保证实时媒体服务已部署 |
| zero-service | 用户、会议、权限、审计和对外接口 | 不自行实现 SFU |
| Egress/Ingress/SIP/Agent worker | 录制、输入源、电话和 Agent 执行 | 管理 API 可调用不代表对应服务已部署或已连接 |

普通管理逻辑使用 `github.com/livekit/server-sdk-go/v2` 的 `LiveKitAPI`。只有 Webhook 验签、token 生成或确实需要生成的 Protocol client 时才直接依赖 `github.com/livekit/protocol`。

## Server SDK

稳定依赖为 `github.com/livekit/server-sdk-go/v2 v2.18.1`，其 tag 的 Protocol 依赖为 `v1.49.0`。SDK 的 `NewLiveKitAPI` 接受 `WithURL`、`WithAPIKey`、`WithToken` 等选项，并通过 `Room()`、`Egress()`、`Ingress()`、`SIP()`、`AgentDispatch()` 访问子客户端。API key/secret 模式由 SDK 为管理调用生成认证 token；`WithToken` 使用调用方已经签发的 token。Cloud failover 不能推导为自托管多区域重试。

下文的 Go 代码块是嵌入业务方法的 API 片段，不是可独立复制的完整文件；未在块内重复声明的 `ctx`、请求变量、配置变量和返回类型由调用方提供。使用时应补齐 imports（通常包括 `context`、`time`、`github.com/livekit/protocol/auth`、`github.com/livekit/protocol/livekit`、`github.com/livekit/protocol/webhook`、`github.com/livekit/server-sdk-go/v2` 及需要的 WebRTC 类型），并按本节锁定的 SDK 版本编译。

```go
// 该示例展示服务端管理 API 的正确构造方式；密钥应由配置或 secret manager 注入。
api, err := lksdk.NewLiveKitAPI(
    lksdk.WithURL(serverURL),
    lksdk.WithAPIKey(apiKey, apiSecret),
)
if err != nil {
    return err
}
// 统一入口暴露管理服务客户端；每个服务启动时构造一次并复用。
roomClient := api.Room()
egressClient := api.Egress()
ingressClient := api.Ingress()
sipClient := api.SIP()
agentClient := api.AgentDispatch()
_ = roomClient
_ = egressClient
_ = ingressClient
_ = sipClient
_ = agentClient
```

### 房间和参与者

`CreateRoomRequest.Name` 是房间标识；`EmptyTimeout` 和 `MaxParticipants` 是请求字段，业务应自行限制输入。房间、参与者、轨道管理应使用 SDK 对应方法，例如 `CreateRoom`、`ListRooms`、`DeleteRoom`、`ListParticipants`、`GetParticipant`、`RemoveParticipant`、`MutePublishedTrack`、`UpdateParticipant`、`UpdateSubscriptions`、`SendData`、`ForwardParticipant` 和 `MoveParticipant`。管理 token 必须只授予实际需要的 grant。

```go
// 创建房间；这是管理 API 调用，不是让客户端加入房间。
room, err := api.Room().CreateRoom(ctx, &livekit.CreateRoomRequest{
    Name:            roomName,
    Metadata:        metadataJSON,
    EmptyTimeout:    emptyTimeoutSeconds,
    MaxParticipants: maxParticipants,
})
if err != nil {
    return err
}
// 删除操作会结束服务端房间状态；业务侧仍需按自己的事务规则更新会议单据。
_, err = api.Room().DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: room.Name})
return err
```

#### 常用管理操作

以下操作都属于服务端管理 API，调用方需要使用 SDK 自动生成的管理凭据，并根据操作授予对应的最小权限。`TrackSid` 必须来自当前参与者的轨道信息，不能使用业务数据库中未经确认的旧值。

```go
// 查询房间和参与者，用于管理后台展示或业务对账。
rooms, err := api.Room().ListRooms(ctx, &livekit.ListRoomsRequest{
    Names: []string{roomName},
})
if err != nil {
    return err
}
participants, err := api.Room().ListParticipants(ctx, &livekit.ListParticipantsRequest{
    Room: roomName,
})
if err != nil {
    return err
}

// 踢出参与者会结束该参与者在当前房间的连接；业务侧应记录操作者和原因。
_, err = api.Room().RemoveParticipant(ctx, &livekit.RoomParticipantIdentity{
    Room: roomName, Identity: identity,
})
if err != nil {
    return err
}

// 服务端静音指定轨道；TrackSid 应从 ListParticipants 的 TrackInfo 获取。
_, err = api.Room().MutePublishedTrack(ctx, &livekit.MuteRoomTrackRequest{
    Room: roomName, Identity: identity, TrackSid: trackSID, Muted: true,
})
if err != nil {
    return err
}

// 更新参与者元数据、属性或权限时，必须确认业务方拥有该房间的管理权限。
_, err = api.Room().UpdateParticipant(ctx, &livekit.UpdateParticipantRequest{
    Room: roomName, Identity: identity,
    Metadata: metadataJSON,
    Attributes: map[string]string{"role": "presenter"},
})
if err != nil {
    return err
}
```

`SendData` 用于服务端向房间发送数据消息，`DestinationIdentities` 为空时按当前协议语义广播；可靠性由 `DataPacket_Kind` 和客户端能力共同决定，不能把 API 返回成功描述为客户端已经处理。

`UpdateParticipant` 也可更新目标参与者的权限；`UpdateSubscriptions` 控制服务端为该参与者设置的轨道订阅关系。下面只展示已确认的 SDK 请求入口，具体权限字段和轨道 SID 必须按锁定 Protocol 版本填充并做业务授权校验。

```go
// 这些管理调用均要求 RoomAdmin；TrackSids 应来自当前房间状态。
_, err = api.Room().UpdateParticipant(ctx, &livekit.UpdateParticipantRequest{
    Room: roomName, Identity: identity, Permission: permission,
})
if err != nil {
    return err
}
_, err = api.Room().UpdateSubscriptions(ctx, &livekit.UpdateSubscriptionsRequest{
    Room: roomName, Identity: identity, TrackSids: trackSIDs, Subscribe: subscribe,
})
```

房间元数据也是独立的管理操作，成功只表示 LiveKit 接受了更新；zero-service 仍应维护自己的会议单据和审计记录。

```go
// 用完整的业务元数据替换房间元数据。
_, err = api.Room().UpdateRoomMetadata(ctx, &livekit.UpdateRoomMetadataRequest{
    Room: roomName, Metadata: metadataJSON,
})
```

```go
// 向指定参与者发送业务数据；消息内容应由业务协议定义并限制大小。
_, err := api.Room().SendData(ctx, &livekit.SendDataRequest{
    Room: roomName,
    Data: []byte(`{"type":"chat","message":"hello"}`),
    Kind: livekit.DataPacket_RELIABLE,
    DestinationIdentities: []string{identity},
})
```

`ForwardParticipant` 是将参与者轨道转发到另一个房间，`MoveParticipant` 是把参与者从源房间移动到目标房间，两者不是同一操作；目标房间、参与者当前状态和管理权限都必须由服务端校验。

```go
// 转发保留源房间连接，并在目标房间产生转发轨道。
_, err := api.Room().ForwardParticipant(ctx, &livekit.ForwardParticipantRequest{
    Room: roomName, Identity: identity, DestinationRoom: targetRoom,
})
// 移动会让参与者离开源房间并加入目标房间。
_, err = api.Room().MoveParticipant(ctx, &livekit.MoveParticipantRequest{
    Room: roomName, Identity: identity, DestinationRoom: targetRoom,
})
```

### Egress 录制与输出

Egress 管理 API 只提交录制任务，不负责替代 Egress worker 执行任务。正式环境还需要 Egress 服务、可写的对象存储或文件输出配置、房间媒体以及 Webhook/查询对账。

```go
// 创建房间合成录制；具体输出类型和字段必须以锁定 Protocol 版本为准。
info, err := api.Egress().StartRoomCompositeEgress(ctx, &livekit.RoomCompositeEgressRequest{
    RoomName: roomName,
    Layout: "grid",
    FileOutputs: []*livekit.EncodedFileOutput{
        {FileType: livekit.EncodedFileType_MP4, Filepath: outputPath},
    },
})
if err != nil {
    return err
}
// EgressId 只标识本次录制任务，不能当作最终文件已落盘的证明。
egressID := info.EgressId

// 查询录制任务状态，用于展示和业务对账。
list, err := api.Egress().ListEgress(ctx, &livekit.ListEgressRequest{RoomName: roomName})
if err != nil {
    return err
}
_ = list

// 请求停止录制；最终文件状态仍应等待 egress_ended 事件或再次查询确认。
_, err = api.Egress().StopEgress(ctx, &livekit.StopEgressRequest{EgressId: egressID})
```

SDK 同时提供 `StartParticipantEgress`、`StartTrackCompositeEgress`、`StartTrackEgress`、`StartWebEgress`、`UpdateLayout` 和 `UpdateStream` 等管理方法。每种方法的请求 oneof、输出字段、存储配置和可用媒体格式必须以当前 Protocol 生成类型核验，不能从旧文章复制字段。

### Ingress 外部输入

Ingress 把 RTMP、WHIP 等外部输入接入 LiveKit 房间。创建或更新 Ingress 需要 `IngressAdmin` 权限；API 成功只表示输入实例被创建或更新，不能表示外部推流已经连接。

```go
// CreateIngressRequest 的输入类型、名称、房间和发布者字段以当前 Protocol 定义为准。
ingress, err := api.Ingress().CreateIngress(ctx, req)
if err != nil {
    return err
}
// 记录 IngressId，用 ListIngress 或 ingress_started/ingress_ended 对账生命周期。
ingressID := ingress.IngressId
_, err = api.Ingress().ListIngress(ctx, &livekit.ListIngressRequest{})
_ = ingressID
```

### SIP 电话接入

SIP 需要单独可用的 SIP 服务、运营商或 PBX、号码、Trunk 和网络策略。SIP API 的管理认证使用 `SIPGrant`；入站 Trunk、出站 Trunk、Dispatch Rule、外呼和转接是不同操作，不能混用请求类型。

```go
// 创建入站 Trunk；认证字段和号码格式必须由运营商及当前 Protocol 共同确认。
inbound, err := api.SIP().CreateSIPInboundTrunk(ctx, inboundReq)
if err != nil {
    return err
}

// 使用已经配置好的出站 Trunk 发起呼叫；等待接听不等于对端一定接通。
call, err := api.SIP().CreateSIPParticipant(ctx, &livekit.CreateSIPParticipantRequest{
    SipTrunkId: trunkID,
    SipCallTo: "+15555550100",
    RoomName: roomName,
    ParticipantIdentity: "sip-participant",
    WaitUntilAnswered: true,
})
if err != nil {
    return err
}
// 呼叫完成后使用 SIP 状态、Webhook 和管理 API 共同确认最终结果。
_ = inbound
_ = call
```

Dispatch Rule 的具体 oneof（如个人房间、主持房间或规则匹配）随 Protocol 演进，实际编码前必须直接阅读选定版本的生成类型；不要在业务规范中固定未经编译验证的 oneof 包装名。外呼失败时可使用 SDK 的 `SIPStatusFrom(err)` 提取运营商返回的 SIP 状态，再保留 Twirp 错误码和消息。

SDK 仍提供 Dispatch Rule 和 SIP 转接入口；由于它们的请求内 oneof/字段会随 Protocol 版本变化，示例保留请求构造交给调用方：

```go
// ruleReq 必须按选定 Protocol 版本设置 TrunkIds 和已核验的规则 oneof。
rule, err := api.SIP().CreateSIPDispatchRule(ctx, ruleReq)
if err != nil {
    return err
}
// 将当前 SIP 参与者转接到外部 SIP endpoint；目标和身份由业务校验。
_, err = api.SIP().TransferSIPParticipant(ctx, transferReq)
if err != nil {
    return err
}
_ = rule
```

### Agent Dispatch 与实时 Agent

Agent Dispatch 是向房间排队派遣 Agent job 的管理 API。Agent worker 还必须实际运行并注册；dispatch 创建成功不等于 worker 已接受任务，也不等于 AI 业务处理已完成。

```go
// 为指定房间派遣 Agent；RoomAdmin 权限只覆盖派遣管理，不替代 worker 的运行授权。
dispatch, err := api.AgentDispatch().CreateDispatch(ctx, &livekit.CreateAgentDispatchRequest{
    Room: roomName,
    AgentName: "meeting-agent",
    Metadata: `{"role":"note-taker"}`,
})
if err != nil {
    return err
}

// 查询和删除派遣记录，用于取消未执行任务或进行业务对账。
_, err = api.AgentDispatch().ListDispatch(ctx, &livekit.ListAgentDispatchRequest{Room: roomName})
if err != nil {
    return err
}
_, err = api.AgentDispatch().DeleteDispatch(ctx, &livekit.DeleteAgentDispatchRequest{
    DispatchId: dispatch.Id, Room: roomName,
})
```

`VideoGrant.Agent`、目标房间的 `RoomAdmin` 和 Cloud Agents 使用的独立管理授权属于三个边界。worker 的连接、媒体订阅、数据处理和退出策略应单独设计，并用可用的 Agent 管理 API 或业务侧状态对账；不要假定 Webhook 提供未经版本核验的 Agent job 事件。

### 实时参与者、媒体发布与接收

Go SDK 的实时参与者路径使用 `ConnectToRoom` 或 `ConnectToRoomWithToken`。它适合 Bot、文件发送器、媒体分析器等服务端参与者，需要 WebRTC、编解码、网络和可用房间；管理 API client 不能替代这条连接。

```go
// 使用后端预先签发的 participant token 连接房间，不向实时参与者暴露 API secret。
room, err := lksdk.ConnectToRoomWithToken(wsURL, participantToken, &lksdk.RoomCallback{
    ParticipantCallback: lksdk.ParticipantCallback{
        // 收到远端轨道后再决定是否订阅、解码或写入外部存储。
        OnTrackSubscribed: func(track *webrtc.TrackRemote, publication *lksdk.RemoteTrackPublication, participant *lksdk.RemoteParticipant) {
            _ = track
            _ = publication
            _ = participant
        },
    },
})
if err != nil {
    return err
}
defer room.Disconnect()
```

文件发布、PCM 音频、pacer、Opus 解码等能力属于媒体层，除了 SDK 版本还受 `pion/webrtc`、编解码器和本机原生媒体库影响。应优先参考稳定 SDK 仓库的 `examples/filesender`、`examples/filesaver` 和 `pkg/media`，先完成格式、采样率、轨道关闭和资源释放验证，再接入业务服务；文档不把媒体处理成功写成仅凭管理 API 可得。

## Webhook 事件

Webhook 是 LiveKit 发往业务服务的 HTTP POST 通知，不是业务数据库的唯一事实来源。请求的 `Content-Type` 是 `application/webhook+json`，`Authorization` 中的签名 JWT 包含原始 body 的校验信息；接收端必须在中间件消费 body 之前保留原始内容。

```go
// 使用与 LiveKit 签名配置匹配的服务端密钥创建验签 provider。
provider := auth.NewSimpleKeyProvider(apiKey, apiSecret)
event, err := webhook.ReceiveWebhookEvent(r, provider)
if err != nil {
    // 验签或解析失败时不要进入业务处理；响应策略要结合重试设计。
    http.Error(w, "invalid webhook", http.StatusBadRequest)
    return
}
// 先按 event.Id 持久化并去重，再把耗时处理投递到有界、可退出的异步流程。
switch event.Event {
case webhook.EventRoomStarted, webhook.EventRoomFinished,
    webhook.EventParticipantJoined, webhook.EventParticipantLeft,
    webhook.EventTrackPublished, webhook.EventTrackUnpublished,
    webhook.EventEgressStarted, webhook.EventEgressUpdated,
    webhook.EventEgressEnded, webhook.EventIngressStarted,
    webhook.EventIngressEnded:
    // 仅读取当前事件实际携带的 proto 字段，并执行幂等业务处理。
default:
    // 新版本可能增加事件；记录未知事件后安全忽略。
}
w.WriteHeader(http.StatusNoContent)
```

事件至少包含 `Id`、`CreatedAt` 和 `Event`，其他房间、参与者、轨道、Egress 或 Ingress 字段按事件类型提供。LiveKit 会对部分失败重试，但不保证业务 Exactly Once；事件可能重复、迟到或因队列限制放弃投递。消费者必须使用唯一约束或去重表，并通过管理 API 对关键状态主动对账。未经当前 Protocol 验证的事件常量不应写入稳定业务契约。

## Protocol 与 Twirp

普通 zero-service 管理逻辑优先使用 `server-sdk-go/v2` 的 `LiveKitAPI`。SDK 内部使用 Protocol 生成的 Protobuf client，并负责管理调用的认证、默认 Twirp client 选项和错误返回。确需直接使用 Protocol 时，必须锁定 SDK/Protocol 版本，阅读对应生成代码并编译验证 client 构造函数、请求字段、枚举和 oneof；不能凭历史文档拼接 `New*JSONClient` 或旧字段。

```go
// 管理调用失败时保留 Twirp 的 code 和 message，便于上层做准确映射与排障。
room, err := api.Room().CreateRoom(ctx, request)
if err != nil {
    var serverErr lksdk.ServerError
    if errors.As(err, &serverErr) {
        return fmt.Errorf("LiveKit %s: %s: %w", serverErr.Code(), serverErr.Msg(), err)
    }
    return fmt.Errorf("调用 LiveKit RoomService 失败: %w", err)
}
_ = room
```

排障先区分认证、请求契约、媒体网络和独立服务：

| 现象 | 优先检查 |
| --- | --- |
| `Unauthenticated` / `PermissionDenied` | API key/secret、token 有效期、grant 和房间范围 |
| `NotFound` / `InvalidArgument` | 房间、参与者、轨道 SID 和锁定 Protocol 字段 |
| 管理 API 正常但无媒体 | WebSocket、ICE、UDP/TCP fallback、NAT、公网地址和 TURN |
| Egress/Ingress/SIP/Agent 任务异常 | 独立服务、worker、存储、运营商连接和凭据 |
| Webhook 验签失败或重复异常 | 原始 body、Authorization、signing key、event ID 去重和重试响应 |

## 自托管部署边界

单节点开发可用本地 HTTP 地址；生产客户端需要可信 TLS 的 HTTPS/WSS 入口。WebRTC 媒体端口、TCP fallback、ICE、NAT、公网地址和 TURN 要按目标网络逐项配置。多节点部署需要按 Server 当前配置样例和官方文档配置 Redis、节点发现、负载均衡和 WebSocket；不能只开放 HTTP API 端口。Egress、Ingress、SIP 和 Agent worker 的独立进程、凭据、存储和网络要求必须单独验收。Cloud 专属区域发现、托管 Agent 或运营商能力不能推导为自托管能力。

不要把示例密钥、SIP 密码、固定媒体端口或未经当前 `config-sample.yaml` 核验的 YAML 当作生产配置。配置字段和默认值以 [Server 配置样例](https://github.com/livekit/livekit/blob/v1.13.6/config-sample.yaml) 与 [自托管部署文档](https://docs.livekit.io/transport/self-hosting/deployment/) 为准。

## zero-service 规范

- LiveKit 拥有房间、参与者、轨道和录制运行态；zero-service 拥有用户、会议单据、业务授权、审计和业务状态。
- ServiceContext 在服务启动时创建并复用 API client；secret 只从安全配置注入，不进入响应和日志。
- Webhook 只触发同步或对账，不是授权来源，也不是唯一状态来源；处理必须容忍重复、丢失、乱序和重试。
- 每个外部调用设置 context 超时，保留 Twirp/HTTP 错误语义；异步任务具备退出、重试、去重和关闭策略。
- 不在公共包固化业务 grant 组合；业务 Logic 根据授权策略签发最小权限 token。

## 验证与限制

可重复的 API 编译验证应在临时 module 中使用稳定 SDK `v2.18.1`，执行 `go mod tidy` 和 `go test ./...`；Protocol 版本由依赖解析并记录。还应执行 `git diff --check`、Markdown 链接/路径/版本/secret/个人路径扫描。

本次审计完成了文档全文静态核对、稳定 tag 与本地提交核对及本地 SDK/Protocol 源码核对；临时 module 已执行 `go mod tidy` 和 `go test ./...`，结果通过且无测试文件。由于示例同时直接导入 Protocol，临时 module 的 MVS 解析到了 `v1.50.4`，不能将该结果当作稳定 SDK tag 的唯一 Protocol 依赖证据。没有在本环境宣称完整 SDK 测试、真实 LiveKit Server、浏览器媒体、WebRTC/TURN、Redis 集群、Egress/Ingress/SIP 运营商或 Agent worker 的端到端通过；这些验证需要运行服务、凭据、网络和系统媒体依赖。

## 官方与源码索引

- [LiveKit Server v1.13.6](https://github.com/livekit/livekit/tree/v1.13.6)
- [Go Server SDK v2.18.1](https://github.com/livekit/server-sdk-go/tree/v2.18.1)
- [SDK LiveKitAPI 入口](https://github.com/livekit/server-sdk-go/blob/v2.18.1/livekitapi.go)、[房间管理 client](https://github.com/livekit/server-sdk-go/blob/v2.18.1/roomclient.go)、[实时参与者](https://github.com/livekit/server-sdk-go/blob/v2.18.1/room.go)
- [SDK Egress client](https://github.com/livekit/server-sdk-go/blob/v2.18.1/egressclient.go)、[Ingress client](https://github.com/livekit/server-sdk-go/blob/v2.18.1/ingressclient.go)、[SIP client](https://github.com/livekit/server-sdk-go/blob/v2.18.1/sipclient.go)、[Agent Dispatch client](https://github.com/livekit/server-sdk-go/blob/v2.18.1/agent_dispatch_client.go)
- [官方 Webhooks & events](https://docs.livekit.io/home/server/webhooks/)
- [官方 Room management](https://docs.livekit.io/home/server/managing-rooms/)
- [官方自托管部署](https://docs.livekit.io/transport/self-hosting/deployment/)
