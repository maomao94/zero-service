# LiveKit RoomCallback 场景与字段说明

本文按场景说明 `github.com/livekit/server-sdk-go/v2` **v2.18.1** 原生 `RoomCallback` 的每个回调：函数签名、参数字段类型与含义、触发来源、典型用途和代码示例。所有字段均以锁定版本源码核对（本地工作树 `a41bea984454ccc958c238bde111022917806aaa`，即 `v2.18.1-28-ga41bea9`；Protocol 为 `v1.49.0`）。

配套代码：`common/livekitx` 的 `Client.JoinRoom(ctx, roomName, identity, opts...)`（以及 `CreateAndJoinRoom`）把回调 **原样透传给 SDK**（`lksdk.NewRoom(callback)`），不桥接、不合并、不合成事件。回调只来自单次加入的 `WithCallback` option：不传则 SDK 使用默认空回调（nil）。`JoinWithContext` 成功返回即表示连接已建立（SDK 原生没有 connected 回调），join token 由 client 的 APIKey/APISecret 内部生成，业务不签 token。返回 SDK 原生 `*lksdk.Room`，断开、重连、轨道、Data、聊天、RPC 等全部事件都在回调中直接处理。

```go
client, err := livekitx.New(
    livekitx.WithURL("http://127.0.0.1:7880"),
    livekitx.WithAPIKey("devkey", "secret"),
)
if err != nil { /* ... */ }
defer client.Close()

// 回调用 WithCallback 指定并原样透传；不传则 SDK 使用默认空回调。
// WithConnectOption 透传 SDK 连接选项。
room, err := client.JoinRoom(ctx, "demo", "user-a",
    livekitx.WithCallback(&lksdk.RoomCallback{
        OnParticipantConnected: func(rp *lksdk.RemoteParticipant) { /* ... */ },
        // ParticipantCallback 是嵌入字段，Data/轨道/参与者回调写在嵌套字面量里：
        ParticipantCallback: lksdk.ParticipantCallback{
            OnDataPacket: func(data lksdk.DataPacket, params lksdk.DataReceiveParams) { /* ... */ },
        },
    }),
    livekitx.WithConnectOption(lksdk.WithAutoSubscribe(true)),
)
```

`RoomCallback` 定义（`callback.go`）：房间级字段直接挂在 `RoomCallback` 上，参与者级字段通过嵌入的 `ParticipantCallback` 提供（`OnLocalTrackSubscribed` 在 `RoomCallback` 上）。

## 回调总览

| 回调 | 层级 | 触发时机 |
|---|---|---|
| `OnDisconnected` / `OnDisconnectedWithReason` | Room | 连接最终断开（含主动 `Disconnect`） |
| `OnParticipantConnected` / `OnParticipantDisconnected` | Room | 远端参与者入会 / 离会 |
| `OnActiveSpeakersChanged` | Room | 活跃说话人列表变化 |
| `OnRoomMetadataChanged` | Room | 房间 metadata 被更新 |
| `OnRecordingStatusChanged` | Room | 房间录制状态变化 |
| `OnRoomMoved` / `OnRoomMovedWithSID` | Room | 房间被迁移到新节点 |
| `OnReconnecting` / `OnReconnected` | Room | 信令重连开始 / 完成 |
| `OnLocalTrackSubscribed` | Room | 本地轨道被远端订阅 |
| `OnLocalTrackPublished` / `OnLocalTrackUnpublished` | Participant | 本地发布 / 取消发布轨道 |
| `OnTrackPublished` / `OnTrackUnpublished` | Participant | 远端发布 / 取消发布轨道 |
| `OnTrackSubscribed` / `OnTrackUnsubscribed` | Participant | 本地订阅 / 退订远端轨道 |
| `OnTrackSubscriptionFailed` | Participant | 订阅远端轨道失败 |
| `OnTrackMuted` / `OnTrackUnmuted` | Participant | 轨道静音 / 取消静音 |
| `OnMetadataChanged` | Participant | 参与者 metadata 变化 |
| `OnAttributesChanged` | Participant | 参与者属性变化 |
| `OnIsSpeakingChanged` | Participant | 参与者说话状态变化 |
| `OnConnectionQualityChanged` | Participant | 参与者连接质量变化 |
| `OnDataPacket` | Participant | 收到任意 DataPacket（Data/聊天/RPC 数据载体） |
| `OnTranscriptionReceived` | Participant | 收到语音转写分段 |

## 一、连接生命周期

### 1.1 断开：`OnDisconnected` / `OnDisconnectedWithReason`

签名：

```go
OnDisconnected           func()
OnDisconnectedWithReason func(reason DisconnectionReason)
```

字段与含义：

- `reason DisconnectionReason`（`callback.go`，string）：SDK 折叠后的断开原因枚举，用于业务可读分支：
  - `LeaveRequested`：用户主动离开（`Room.Disconnect()`）
  - `UserUnavailable`：远端用户不可达
  - `RejectedByUser`：被远端用户拒绝
  - `Failed`：连接房间失败（join failure / 信令关闭 / 状态不匹配）
  - `RoomClosed`：房间被关闭
  - `ParticipantRemoved`：被服务端移除（管理员踢人）
  - `DuplicateIdentity`：相同 identity 重复入会，旧连接被顶掉
  - `OtherReason`：其他（协议原因未折叠到的都归此）
- 原始协议原因：`room.DisconnectReason()` 返回 `livekit.DisconnectReason`（如 `ROOM_DELETED`、`SERVER_SHUTDOWN`、`CLIENT_INITIATED`），需要区分折叠值之外的原因时使用。**SDK 在触发 `OnDisconnected` 前已写入该值**，回调内可直接读取。

触发来源：连接最终断开时（主动 `Disconnect`、服务端踢人/关房、网络失败放弃重连等）。SDK 同时触发 `OnDisconnected` 与 `OnDisconnectedWithReason` 两个回调，业务任选其一，不要重复处理。

典型用途：更新会议状态为已结束、释放本地资源、告警、对账；配合 `common/livekitx` 的 `room.Disconnect()` 结束连接，业务状态在回调里做。

```go
var room *lksdk.Room // 来自 client.JoinRoom / CreateAndJoinRoom 的返回值

callback := &lksdk.RoomCallback{
    OnDisconnected: func() {
        // 原始协议原因：room.DisconnectReason()
        reason := room.DisconnectReason()
        // 折叠原因：lksdk.GetDisconnectionReason(reason)
        switch lksdk.GetDisconnectionReason(reason) {
        case lksdk.ParticipantRemoved, lksdk.DuplicateIdentity:
            // 被服务端移除或身份冲突：提示用户重新入会
        case lksdk.LeaveRequested:
            // 主动离开，静默处理
        }
    },
}
```

### 1.2 重连：`OnReconnecting` / `OnReconnected`

签名：

```go
OnReconnecting func()
OnReconnected  func()
```

字段：无参数。重连期间 SDK 自动处理信令/媒体恢复，本地轨道会重新发布（SDK 内部 `republishTracks`）。

触发来源：信令连接中断开始自动重连时触发 `OnReconnecting`；重连完成、房间状态恢复后触发 `OnReconnected`。**不保证重连成功**——最终失败会走到 `OnDisconnected`。

典型用途：展示"连接中…"状态、暂停依赖实时数据的 UI 更新、重连后刷新房间 metadata/参与者列表。

```go
OnReconnecting: func() { ui.SetConnectionState("reconnecting") },
OnReconnected:  func() { ui.SetConnectionState("connected") },
```

### 1.3 入会 / 离会：`OnParticipantConnected` / `OnParticipantDisconnected`

签名：

```go
OnParticipantConnected    func(participant *RemoteParticipant)
OnParticipantDisconnected func(participant *RemoteParticipant)
```

字段与含义（`*lksdk.RemoteParticipant`，`remoteparticipant.go` + `participant.go` 的 `Participant` 接口）：

- `SID() string`：参与者在房间内的唯一 SID（服务端分配）
- `Identity() string`：业务身份（join token 的 identity），同一房间内唯一
- `Name() string`：展示名（join token 的 name，可为空）
- `Kind() ParticipantKind`：参与者类型（标准/Ingress/Egress/Agent/SIP 等，来自协议 `ParticipantInfo.Kind`）
- `IsSpeaking() bool`、`AudioLevel() float32`：当前说话状态与音量（0~1）
- `Metadata() string`：参与者 metadata（业务自定义 JSON 等）
- `Attributes() map[string]string`：参与者属性
- `TrackPublications() []TrackPublication`：当前轨道发布集合
- `Permissions() *livekit.ParticipantPermission`：参与者的服务端权限

触发来源：远端参与者加入 / 离开房间（服务端推送参与者更新）。自己（本地参与者）不触发。

典型用途：参会名单增删、入会/离会通知、时长统计、离会后清理该参与者的 UI 与缓存、踢人后的处理。

```go
OnParticipantConnected: func(rp *lksdk.RemoteParticipant) {
    roster.Add(rp.Identity(), rp.Name(), rp.Metadata())
},
OnParticipantDisconnected: func(rp *lksdk.RemoteParticipant) {
    roster.Remove(rp.Identity())
},
```

### 1.4 房间迁移：`OnRoomMoved` / `OnRoomMovedWithSID`

签名：

```go
OnRoomMoved        func(roomName string, token string)
OnRoomMovedWithSID func(roomName string, roomSID string, token string)
```

字段与含义：

- `roomName string`：迁移后的房间名（一般不变）
- `roomSID string`：迁移后的房间 SID（服务端分配）
- `token string`：SDK 内部更换后的接入 token（**不需要业务处理**，SDK 已自动用于重连）

触发来源：房间被迁移到另一个 LiveKit 节点（服务端 `MoveParticipant` 或自动迁移）。SDK 同时触发两个回调，通常用带 SID 的 `OnRoomMovedWithSID` 一次处理。

典型用途：感知迁移以刷新房间 SID 相关缓存、记录审计日志；业务无需自行重连。

## 二、活跃说话人与房间状态

### 2.1 活跃说话人：`OnActiveSpeakersChanged`

签名：

```go
OnActiveSpeakersChanged func(participants []Participant)
```

字段与含义：`participants []lksdk.Participant`——当前活跃说话人快照（含本地参与者；`Participant` 接口见 1.3）。通过 `p.IsSpeaking()` / `p.AudioLevel()` 读取状态；列表为空表示无人说话。

触发来源：房间内说话人集合变化（音量检测），可能高频触发。

典型用途：发言者高亮、字幕跟随、"当前谁在讲"展示；如需在每次变化时拿到完整状态，可用 `room.ActiveSpeakers()`。

```go
OnActiveSpeakersChanged: func(participants []lksdk.Participant) {
    for _, p := range participants {
        if p.IsSpeaking() {
            ui.HighlightSpeaker(p.Identity())
        }
    }
},
```

### 2.2 房间 metadata：`OnRoomMetadataChanged`

签名：

```go
OnRoomMetadataChanged func(metadata string)
```

字段与含义：`metadata string`——更新后的完整房间 metadata（业务自定义内容，如会议主题 JSON）。回调只给新值，旧值需业务自行保存。

触发来源：服务端更新房间 metadata（管理 API `UpdateRoomMetadata` 或参与者带权限更新）。

典型用途：会议主题/配置同步、审核留痕；初始值在入会成功后用 `room.Metadata()` 读取。

### 2.3 录制状态：`OnRecordingStatusChanged`

签名：

```go
OnRecordingStatusChanged func(isRecording bool)
```

字段与含义：`isRecording bool`——房间是否正在被 Egress 录制。

触发来源：Egress 开始/停止录制（服务端状态同步）。

典型用途：录制指示灯、录制计费/审计；注意管理 API 可发起 Egress 不代表录制已成功，最终以该回调为准。

## 三、轨道回调

### 3.1 本地轨道：`OnLocalTrackPublished` / `OnLocalTrackUnpublished` / `OnLocalTrackSubscribed`

签名：

```go
OnLocalTrackPublished   func(publication *LocalTrackPublication, lp *LocalParticipant)
OnLocalTrackUnpublished func(publication *LocalTrackPublication, lp *LocalParticipant)
OnLocalTrackSubscribed  func(publication *LocalTrackPublication, lp *LocalParticipant) // Room 级
```

字段与含义：

- `publication *lksdk.LocalTrackPublication`：本地轨道发布。常用方法：`SID()`、`Name()`、`Kind()`（`lksdk.TrackKindAudio` / `TrackKindVideo`）、`Source()`（`livekit.TrackSource`：麦克风/摄像头/屏幕共享等）、`MimeType()`、`IsMuted()`、`TrackLocal()`（`webrtc.TrackLocal`）。
- `lp *lksdk.LocalParticipant`：本地参与者（`Identity()` 等见 1.3）。

触发来源：本地发布/取消发布轨道（`room.LocalParticipant.PublishTrack(...)`、`UnpublishTrack(...)`）；`OnLocalTrackSubscribed` 在远端订阅本地轨道时触发。

典型用途：本地画面/共享状态更新、发布失败提示、统计已发布轨道。

### 3.2 远端轨道发布/取消：`OnTrackPublished` / `OnTrackUnpublished`

签名：

```go
OnTrackPublished   func(publication *RemoteTrackPublication, rp *RemoteParticipant)
OnTrackUnpublished func(publication *RemoteTrackPublication, rp *RemoteParticipant)
```

字段与含义：

- `publication *lksdk.RemoteTrackPublication`：远端轨道发布。常用方法：`SID()`、`Name()`、`Kind()`、`Source()`、`MimeType()`、`IsMuted()`、`IsSubscribed()`、`TrackRemote()`（`*webrtc.TrackRemote`）、`Receiver()`；`SetSubscribed(bool)` 控制本地是否订阅该轨道，`IsEnabled()`/`SetEnabled(bool)` 控制是否接收。
- `rp *lksdk.RemoteParticipant`：发布该轨道的远端参与者。

触发来源：远端参与者发布/取消发布轨道（服务端同步）。

典型用途：更新远端画面/共享列表；对不需要的轨道（如他人的屏幕共享）调 `SetSubscribed(false)` 节省带宽。

### 3.3 订阅/退订：`OnTrackSubscribed` / `OnTrackUnsubscribed`

签名：

```go
OnTrackSubscribed   func(track *webrtc.TrackRemote, publication *RemoteTrackPublication, rp *RemoteParticipant)
OnTrackUnsubscribed func(track *webrtc.TrackRemote, publication *RemoteTrackPublication, rp *RemoteParticipant)
```

字段与含义：

- `track *webrtc.TrackRemote`：已建立的远端媒体轨道，可直接接播放器（如 pion/媒体框架的 `TrackRemote` 消费）。
- `publication *lksdk.RemoteTrackPublication`：见 3.2。
- `rp *lksdk.RemoteParticipant`：轨道所属参与者。

触发来源：本地完成对某条远端轨道的订阅（自动订阅或 `SetSubscribed(true)` 后建立媒体链路）/ 退订完成。

典型用途：订阅成功后把轨道接入渲染/播放管线；退订后释放对应资源。**播放失败（如编解码不支持）不会走这里，见 3.4。**

### 3.4 订阅失败：`OnTrackSubscriptionFailed`

签名：

```go
OnTrackSubscriptionFailed func(sid string, rp *RemoteParticipant)
```

字段与含义：`sid string`——失败的轨道 SID（不是参与者 SID，用于关联 publication）；`rp *lksdk.RemoteParticipant`——轨道所属参与者。

触发来源：订阅远端轨道失败（媒体协商失败、不支持等）。

典型用途：记录失败、提示用户、稍后重试 `SetSubscribed(true)`。

### 3.5 静音：`OnTrackMuted` / `OnTrackUnmuted`

签名：

```go
OnTrackMuted   func(pub TrackPublication, p Participant)
OnTrackUnmuted func(pub TrackPublication, p Participant)
```

字段与含义：

- `pub lksdk.TrackPublication`（接口）：`SID()`、`Name()`、`Kind()`、`Source()`、`MimeType()`、`IsMuted()`、`IsSubscribed()`、`TrackInfo()`（`*livekit.TrackInfo`）、`Track()`。
- `p lksdk.Participant`：轨道所属参与者（本地或远端）。

触发来源：参与者静音/取消静音（本地操作或远端状态同步）。

典型用途：麦克风/摄像头静音图标更新、静音状态落库。

## 四、参与者状态回调

### 4.1 参与者 metadata：`OnMetadataChanged`

签名：

```go
OnMetadataChanged func(oldMetadata string, p Participant)
```

字段与含义：`oldMetadata string`——变化前的 metadata（新值通过 `p.Metadata()` 读取）；`p lksdk.Participant`——参与者（本地或远端）。

触发来源：参与者 metadata 被更新（业务自定义信息，如头像、昵称）。

典型用途：用户资料同步、审核留痕。

### 4.2 参与者属性：`OnAttributesChanged`

签名：

```go
type ParticipantAttributesChangedFunc func(changed map[string]string, p Participant)
OnAttributesChanged ParticipantAttributesChangedFunc
```

字段与含义：`changed map[string]string`——本次变化的属性集合（**被删除的属性值为空字符串**）；`p lksdk.Participant`——参与者。

触发来源：参与者属性被更新（`LocalParticipant.SetAttributes(...)` 或远端同步）。

典型用途：会控指令回执、状态属性同步（如"正在共享"标记）。

### 4.3 说话状态：`OnIsSpeakingChanged`

签名：

```go
OnIsSpeakingChanged func(p Participant)
```

字段与含义：`p lksdk.Participant`——说话状态变化的参与者，用 `p.IsSpeaking()` 读取当前状态。

触发来源：参与者开始/停止说话（音量检测），与 `OnActiveSpeakersChanged` 互补（单参与者粒度）。

典型用途：麦位状态、发言统计、字幕开关。

### 4.4 连接质量：`OnConnectionQualityChanged`

签名：

```go
OnConnectionQualityChanged func(update *livekit.ConnectionQualityInfo, p Participant)
```

字段与含义（`*livekit.ConnectionQualityInfo`，protocol `v1.49.0`）：

- `ParticipantSid string`：参与者 SID
- `Quality livekit.ConnectionQuality`：`POOR` / `GOOD` / `EXCELLENT` / `LOST`
- `Score float32`：质量评分（0~1）

触发来源：参与者网络质量评估更新（周期性）。

典型用途：弱网告警、质量监控面板、`LOST` 时提示可能掉线。

## 五、Data 与转写

### 5.1 Data 接收：`OnDataPacket`

签名：

```go
OnDataPacket func(data DataPacket, params DataReceiveParams)
```

字段与含义：

- `data lksdk.DataPacket`（接口，`ToProto() *livekit.DataPacket`）。常见实现：
  - `*lksdk.UserDataPacket`：`Payload []byte`（任意字节）、`Topic string`（可选，业务约定主题）
  - `*livekit.ChatMessage`：SDK 原生聊天消息（字段见下文"聊天消息识别"）
  - `*livekit.SipDTMF`：SIP 按键音
- `params lksdk.DataReceiveParams`（`data.go`）：
  - `Sender *RemoteParticipant`：发送者（参与者信息未建立时可能为 nil）
  - `SenderIdentity string`：发送者 identity（优先用这个做发送方识别）
  - `Topic string`：**Deprecated**，使用 `UserDataPacket.Topic`

触发来源：收到任意 DataChannel 数据包（可靠/不可靠、任意 topic）。聊天与业务自定义 Data、RPC 都在这里出现。

典型用途：业务自定义数据分发、数据链路审计；聊天与 RPC 识别见专门章节。

```go
ParticipantCallback: lksdk.ParticipantCallback{
    OnDataPacket: func(data lksdk.DataPacket, params lksdk.DataReceiveParams) {
        switch msg := data.(type) {
        case *lksdk.UserDataPacket:
            // topic 区分业务数据类型；聊天识别章节有双路径说明
            _ = msg.Topic
        case *livekit.ChatMessage:
            // 聊天消息（见聊天识别章节）
        }
        _ = params.SenderIdentity
    },
},
```

发送 Data 使用 `room.LocalParticipant.PublishData(payload, lksdk.WithDataPublishTopic("topic"), lksdk.WithDataPublishReliable(true))` 或 `PublishDataPacket(pck, ...)`；发送聊天需要 join token 带 `CanPublishData: true`（`grant.SetCanPublishData(true)`）。服务端向房间广播/定向发送富媒体（语音、图片等二进制数据）可用 `common/livekitx` 的 `client.SendData(ctx, room, topic, payload, destinations...)`（管理 API，`destinations` 为空即广播），接收方同样在 `OnDataPacket` 按 topic 识别 `UserDataPacket`。

### 5.2 旧 Data 回调（已废弃）：`OnDataReceived`

签名：`OnDataReceived func(data []byte, params DataReceiveParams)`。SDK 标记为 Deprecated，新代码一律使用 `OnDataPacket`。

### 5.3 语音转写：`OnTranscriptionReceived`

签名：

```go
OnTranscriptionReceived func(transcriptionSegments []*TranscriptionSegment, p Participant, publication TrackPublication)
```

字段与含义（`*lksdk.TranscriptionSegment`，`transcription.go`）：

- `ID string`：分段 ID
- `Text string`：转写文本
- `Language string`：语言
- `StartTime uint64` / `EndTime uint64`：相对时间（毫秒）
- `Final bool`：是否最终结果（false 表示中间结果，可能被后续分段修正）

`p lksdk.Participant`：说话者；`publication lksdk.TrackPublication`：对应音频轨道。

触发来源：Agent/转写服务推送的实时转写。

典型用途：实时字幕、会议纪要草稿（只保留 `Final` 分段）。

## 六、聊天消息识别（双路径）

`OnDataPacket` 里识别聊天消息有两条路径，字段差异必须分清：

### 6.1 SDK 原生：`*livekit.ChatMessage`

协议字段（`livekit_models.pb.go`，protocol `v1.49.0`）：

| 字段 | 类型 | 含义 |
|---|---|---|
| `Id` | `string` | 消息 ID（发送方生成，全局唯一，如 `MSG_xxx`） |
| `Timestamp` | `int64` | 发送时间戳（毫秒） |
| `EditTimestamp` | `*int64` | 编辑时间戳（仅编辑已有消息时存在） |
| `Message` | `string` | 聊天文本 |
| `Deleted` | `bool` | 是否删除消息（`true` 表示撤回） |
| `Generated` | `bool` | 是否由 Agent 从音频转写生成 |

发送：`room.LocalParticipant.PublishDataPacket(lksdk.ChatMessage(time.Now(), text))`，ID 与时间戳由 SDK 生成。

识别：

```go
case *livekit.ChatMessage:
    // msg.GetId() / msg.GetTimestamp() / msg.GetMessage() / msg.GetDeleted() / msg.GetGenerated()
```

### 6.2 兼容约定：`*lksdk.UserDataPacket` + 业务自定义 topic

`UserDataPacket` 只有 `Payload []byte` 与 `Topic string` 两个字段。topic 约定完全交业务：**业务定义自己的聊天 topic（如 `"chat"`），topic 等于该值的 UserDataPacket 视为聊天**。此路径没有消息 ID、时间戳、删除/生成标记，`params.SenderIdentity` 是发送者 identity。

识别：

```go
const chatTopic = "chat" // 业务自定义约定

case *lksdk.UserDataPacket:
    if msg.Topic == chatTopic {
        // 文本在 msg.Payload；无 ID/时间戳，业务自行决定是否补充
    }
```

### 6.3 两路径差异速查

| 维度 | `*livekit.ChatMessage` | `UserDataPacket` + 业务自定义 topic |
|---|---|---|
| 消息 ID | 有（`Id`） | 无 |
| 时间戳 | 有（毫秒） | 无 |
| 删除/撤回 | `Deleted` | 无 |
| Agent 生成标记 | `Generated` | 无 |
| 文本位置 | `Message` | `Payload` |
| 发送方式 | `PublishDataPacket(ChatMessage(...))` | `PublishData`/`PublishDataPacket` + `WithDataPublishTopic(topic)` |

## 七、RPC 收发

RPC 全部使用 SDK 原生 API（`common/livekitx` 不包一层）。

### 7.1 注册方法

```go
func (r *Room) RegisterRpcCtxMethod(method string, handler RpcHandlerCtxFunc) error
// RpcHandlerCtxFunc = func(ctx context.Context, data []byte) ([]byte, error)
```

- handler 的 `ctx` 携带 `*lksdk.RpcInvocationMetadata`，用 `lksdk.RPCMetadataFromContext(ctx)` 读取：
  - `RequestID string`：请求 ID（调用双方一致，用于日志关联）
  - `CallerIdentity string`：调用方 identity
  - `ResponseTimeout time.Duration`：调用方期望的响应时限
- 返回 `([]byte, error)` 回传调用方；handler 错误语义由 SDK 返回远端。
- 同名方法重复注册返回错误（先 `UnregisterRpcMethod(method)` 再注册）。

### 7.2 调用方法

```go
func (p *LocalParticipant) PerformRpc(params PerformRpcParams) (*string, error)
// PerformRpcParams{DestinationIdentity, Method, Payload string, ResponseTimeout *time.Duration}
```

- **SDK v2.18.1 的 `PerformRpc` 不接收 context**，用 `params.ResponseTimeout` 控制时限：小于 8000ms 会被钳制到 8000ms，默认 15000ms。
- 返回 `*string`（响应）或错误。
- **未注册方法由 SDK 自动返回 UnsupportedMethod**，不会触发任何业务回调。

### 7.3 完整示例

```go
// bob 侧注册
err := bobRoom.RegisterRpcCtxMethod("echo", func(ctx context.Context, data []byte) ([]byte, error) {
    meta := lksdk.RPCMetadataFromContext(ctx) // CallerIdentity / RequestID
    _ = meta
    return data, nil
})

// alice 侧调用
timeout := 10 * time.Second
resp, err := aliceRoom.LocalParticipant.PerformRpc(lksdk.PerformRpcParams{
    DestinationIdentity: "bob",
    Method:              "echo",
    Payload:             "ping",
    ResponseTimeout:     &timeout,
})
```

## 八、Webhook 接收（服务端推送）

Webhook 由 LiveKit 服务器向业务 HTTP 端点推送，与实时回调无关，但同样"直接复用 SDK"：

```go
import "github.com/livekit/protocol/webhook"

event, err := webhook.ReceiveWebhookEvent(req, livekitx.NewWebhookKeyProvider(cfg.WebhookKey))
if err != nil {
    // 验签失败：返回错误，不处理任何业务
    return err
}
```

- `NewWebhookKeyProvider(signingKey string) auth.KeyProvider`：对任意 API key claim 返回同一个 signing key（服务器用自身 API key 签名，业务只持 secret；**不要用 `auth.NewSimpleKeyProvider("", key)`，空 key 永远验签失败**）。
- `*livekit.WebhookEvent` 关键字段：`Event`（`room_started`/`room_finished`/`participant_joined`/`participant_left`/`track_published`/`track_unpublished`/`egress_*`/`ingress_*` 等）、`Id`（**事件唯一 ID**）、`CreatedAt`（毫秒）、`Room`、`Participant`、`Track`、`EgressInfo`、`IngressInfo`、`Sip`。
- **幂等要求**：事件可能重复、迟到、乱序或丢失。业务必须持久化 `event.GetId()` 去重，关键状态用管理 API 主动对账；未知事件安全记录并忽略。
- Webhook 只用于状态同步/对账，不授予权限，也不替代管理 API。

## 九、生命周期与注意事项

- `common/livekitx` 的 `JoinRoom`/`CreateAndJoinRoom` 成功返回即已连接（SDK 无 connected 回调）；回调只来自 `WithCallback`（单次）指定，不传则 SDK 空回调（nil）；入会初值（房间 metadata、参与者列表）用 `room.Metadata()`、`room.GetRemoteParticipants()` 读取。
- 连接结束由业务调用 `room.Disconnect()`（SDK 原生方法，幂等）；覆盖 `OnDisconnected` 不影响 `Disconnect` 的清理。
- 断开原因读取：回调内 `room.DisconnectReason()` 已写入（见 1.1）；SDK 同时触发 `OnDisconnected` 与 `OnDisconnectedWithReason`，只处理一次。
- 请求/回调内的网络调用要自建超时 context，不要在回调里阻塞读循环。
- 本指南所有签名以 SDK v2.18.1（本地工作树 `v2.18.1-28-ga41bea9`）与 protocol v1.49.0 为准；升级 SDK 后重新核对字段与枚举。