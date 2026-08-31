package livekitx

import (
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"
)

// ChatTopic 是通过 DataChannel 发送聊天文本时的默认 topic 约定。
// 使用 UserDataPacket 携带聊天文本时必须使用该 topic，桥接才会把数据包
// 识别为聊天消息；SDK 原生 *livekit.ChatMessage 包不依赖 topic。
const ChatTopic = "chat"

// ChatMessageEvent 是交给业务处理的聊天消息；消息持久化、历史查询、未读数、
// 敏感词审核和业务会话归属由业务 handler 负责。
type ChatMessageEvent struct {
	RoomName  string // 消息所在房间
	SenderID  string // 发送者 identity
	MessageID string // 协议消息 ID（*livekit.ChatMessage 路径），UserData 路径为空
	Topic     string // 数据 topic；SDK 原生聊天消息固定为 ChatTopic
	Text      string // 聊天文本
	Payload   []byte // 原始 payload（UserData 路径）；SDK 原生聊天消息为空
	Timestamp int64  // 协议时间戳（毫秒），UserData 路径为 0
	Deleted   bool   // 是否为删除消息（*livekit.ChatMessage 路径）
	Generated bool   // 是否由 Agent 从音频转写生成（*livekit.ChatMessage 路径）
}

// RoomConnectionStatus 描述实时连接状态 Hook 事件的状态。
type RoomConnectionStatus string

const (
	// RoomConnectionConnected 表示入会成功（由 Connect 在连接建立后合成分发）。
	RoomConnectionConnected RoomConnectionStatus = "connected"
	// RoomConnectionReconnecting 表示信令重连中。
	RoomConnectionReconnecting RoomConnectionStatus = "reconnecting"
	// RoomConnectionReconnected 表示重连完成。
	RoomConnectionReconnected RoomConnectionStatus = "reconnected"
	// RoomConnectionDisconnected 表示连接已断开。
	RoomConnectionDisconnected RoomConnectionStatus = "disconnected"
)

// RoomConnectionEvent 是连接状态变化事件，用于会议状态同步。
type RoomConnectionEvent struct {
	RoomName       string                    // 房间名
	Status         RoomConnectionStatus      // 当前状态
	Reason         lksdk.DisconnectionReason // 断开时的 SDK 折叠原因；非断开状态为空
	ProtocolReason livekit.DisconnectReason  // 断开时的原始协议原因；非断开状态为 0
}

// RoomMetadataEvent 是房间 metadata 变化事件。
type RoomMetadataEvent struct {
	RoomName string // 房间名
	Metadata string // 新的 metadata
}

// RoomRecordingEvent 是房间录制状态变化事件。
type RoomRecordingEvent struct {
	RoomName    string // 房间名
	IsRecording bool   // 是否正在录制
}

// RoomMovedEvent 是房间迁移事件；迁移后 token 已由 SDK 内部更换。
type RoomMovedEvent struct {
	RoomName string // 迁移后的房间名
	RoomSID  string // 迁移后的房间 SID
	Token    string // 迁移后的接入 token
}

// ActiveSpeakersEvent 是活跃说话人列表变化事件。
type ActiveSpeakersEvent struct {
	RoomName     string              // 房间名
	Participants []lksdk.Participant // 当前活跃说话人快照
}

// ParticipantConnectedEvent 是远端参与者入会事件。
type ParticipantConnectedEvent struct {
	RoomName    string                   // 房间名
	Participant *lksdk.RemoteParticipant // 入会参与者
}

// ParticipantDisconnectedEvent 是远端参与者离会事件。
type ParticipantDisconnectedEvent struct {
	RoomName    string                   // 房间名
	Participant *lksdk.RemoteParticipant // 离会参与者
}

// ParticipantMetadataEvent 是参与者 metadata 变化事件。
type ParticipantMetadataEvent struct {
	RoomName    string            // 房间名
	Participant lksdk.Participant // 参与者
	OldMetadata string            // 变化前的 metadata
}

// ParticipantAttributesEvent 是参与者属性变化事件；被删除的属性值为空字符串。
type ParticipantAttributesEvent struct {
	RoomName    string            // 房间名
	Participant lksdk.Participant // 参与者
	Changed     map[string]string // 变化的属性集合
}

// ParticipantSpeakingEvent 是参与者说话状态变化事件。
type ParticipantSpeakingEvent struct {
	RoomName    string            // 房间名
	Participant lksdk.Participant // 参与者
	IsSpeaking  bool              // 当前是否在说话
}

// ConnectionQualityEvent 是参与者连接质量变化事件。
type ConnectionQualityEvent struct {
	RoomName    string                         // 房间名
	Participant lksdk.Participant              // 参与者
	Update      *livekit.ConnectionQualityInfo // SDK 质量信息
}

// TrackEventKind 描述轨道事件类型。
type TrackEventKind string

const (
	// TrackEventPublished 表示远端发布轨道。
	TrackEventPublished TrackEventKind = "published"
	// TrackEventUnpublished 表示远端取消发布轨道。
	TrackEventUnpublished TrackEventKind = "unpublished"
	// TrackEventSubscribed 表示本地订阅远端轨道成功。
	TrackEventSubscribed TrackEventKind = "subscribed"
	// TrackEventUnsubscribed 表示本地退订远端轨道。
	TrackEventUnsubscribed TrackEventKind = "unsubscribed"
	// TrackEventSubscriptionFailed 表示订阅远端轨道失败。
	TrackEventSubscriptionFailed TrackEventKind = "subscription_failed"
	// TrackEventMuted 表示轨道静音。
	TrackEventMuted TrackEventKind = "muted"
	// TrackEventUnmuted 表示轨道取消静音。
	TrackEventUnmuted TrackEventKind = "unmuted"
	// TrackEventLocalPublished 表示本地发布轨道。
	TrackEventLocalPublished TrackEventKind = "local_published"
	// TrackEventLocalUnpublished 表示本地取消发布轨道。
	TrackEventLocalUnpublished TrackEventKind = "local_unpublished"
	// TrackEventLocalSubscribed 表示本地轨道被远端订阅。
	TrackEventLocalSubscribed TrackEventKind = "local_subscribed"
)

// TrackEvent 是轨道状态变化事件，用于会议轨道状态同步。
type TrackEvent struct {
	RoomName    string                  // 房间名
	Kind        TrackEventKind          // 事件类型
	Participant lksdk.Participant       // 事件参与者；本地事件为 nil
	Publication lksdk.TrackPublication  // 相关轨道发布；订阅失败事件为 nil
	Track       *webrtc.TrackRemote     // Subscribed/Unsubscribed 时的远端轨道；其余事件为 nil
	TrackSID    string                  // SubscriptionFailed 时的轨道 SID；其余事件为空
	Local       *lksdk.LocalParticipant // 本地事件（LocalPublished/LocalUnpublished/LocalSubscribed）的本地参与者；其余为 nil
}

// DataEvent 是收到任意 SDK DataPacket 的事件，包含原始包和解析出的 topic/payload。
type DataEvent struct {
	RoomName       string           // 房间名
	SenderIdentity string           // 发送者 identity
	SenderSID      string           // 发送者 SID；参与者信息未建立时为空
	Topic          string           // 数据 topic（UserDataPacket）；SDK 原生聊天消息为 ChatTopic
	Payload        []byte           // 数据 payload（UserDataPacket）；SDK 原生聊天消息为文本字节
	Packet         lksdk.DataPacket // SDK 原始数据包
}

// RPCRequestEvent 是经由 livekitx.RegisterRPC 注册的方法收到的 RPC 请求事件。
// 仅覆盖通过本包注册的方法；SDK 未注册方法由 SDK 自动返回 UnsupportedMethod。
type RPCRequestEvent struct {
	RoomName       string // 房间名
	CallerIdentity string // 调用方 identity
	RequestID      string // SDK 请求 ID，可关联响应与日志
	Method         string // 方法名
	Payload        string // 请求 payload
}
