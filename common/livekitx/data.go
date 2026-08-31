package livekitx

import (
	"context"
	"time"

	lksdk "github.com/livekit/server-sdk-go/v2"
)

// PublishData 转发 SDK 的可靠/不可靠 Data API，不改变远端处理语义。
func (r *RealtimeRoom) PublishData(payload []byte, opts ...lksdk.DataPublishOption) error {
	if r == nil || r.room == nil || r.room.LocalParticipant == nil {
		return ErrNilRoom
	}
	return r.room.LocalParticipant.PublishData(payload, opts...)
}

// PublishDataPacket 发送 SDK DataPacket（含 *livekit.ChatMessage 等协议包）。
func (r *RealtimeRoom) PublishDataPacket(pck lksdk.DataPacket, opts ...lksdk.DataPublishOption) error {
	if r == nil || r.room == nil || r.room.LocalParticipant == nil {
		return ErrNilRoom
	}
	return r.room.LocalParticipant.PublishDataPacket(pck, opts...)
}

// PublishChatMessage 发送一条 SDK 原生聊天消息（*livekit.ChatMessage），
// 消息 ID 与时间戳由 SDK 生成；接收方通过 OnChatMessage Hook 收到事件。
func (r *RealtimeRoom) PublishChatMessage(text string, opts ...lksdk.DataPublishOption) error {
	return r.PublishDataPacket(lksdk.ChatMessage(time.Now(), text), opts...)
}

// PerformRPC 转发 SDK 原生 RPC 请求与错误。
// SDK v2.18.1 的实时 RPC 接口不接收 context；调用方应通过 params.Timeout
// 控制远端调用时限，不能误以为此方法支持调用方 context 取消。
func (r *RealtimeRoom) PerformRPC(params lksdk.PerformRpcParams) (*string, error) {
	if r == nil || r.room == nil || r.room.LocalParticipant == nil {
		return nil, ErrNilRoom
	}
	return r.room.LocalParticipant.PerformRpc(params)
}

// RegisterRPC 注册 SDK RPC handler。请求到达时先分发 RPCRequestEvent Hook
// 再调用业务 handler；业务 handler 的错误仍由 SDK 语义返回给远端。
func (r *RealtimeRoom) RegisterRPC(method string, handler lksdk.RpcHandlerCtxFunc) error {
	if r == nil || r.room == nil {
		return ErrNilRoom
	}
	if handler == nil {
		return ErrInvalidConfig
	}
	wrapped := func(ctx context.Context, data []byte) ([]byte, error) {
		r.dispatchRPCRequest(ctx, method, data)
		return handler(ctx, data)
	}
	return r.room.RegisterRpcCtxMethod(method, wrapped)
}

// UnregisterRPC 注销 SDK RPC handler。
func (r *RealtimeRoom) UnregisterRPC(method string) {
	if r != nil && r.room != nil {
		r.room.UnregisterRpcMethod(method)
	}
}

// dispatchRPCRequest 把 RPC 请求分发到 OnRPCRequest Hook；元数据来自 SDK 注入的 context。
func (r *RealtimeRoom) dispatchRPCRequest(ctx context.Context, method string, data []byte) {
	if r == nil || r.client == nil || r.client.hooks == nil {
		return
	}
	event := RPCRequestEvent{RoomName: r.roomName(), Method: method, Payload: string(data)}
	if meta := lksdk.RPCMetadataFromContext(ctx); meta != nil {
		event.CallerIdentity = meta.CallerIdentity
		event.RequestID = meta.RequestID
	}
	_ = r.client.hooks.rpc.dispatch(ctx, event)
}
