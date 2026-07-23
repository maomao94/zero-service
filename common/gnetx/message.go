package gnetx

// 本文件定义 gnetx 的 opt-in 消息接口。Core 层（Codec/Handler/Session）不强制
// 消息实现这些接口；仅在启用 Router 或请求-响应（tid 响应式）时由框架按需断言。
//
// 设计原则：纯推送/遥测协议可以完全不实现这些接口，只用 Codec + Handler；
// 需要按 id 路由或请求-响应关联的协议，让消息类型实现对应接口即可启用相应能力。

// Identifiable 用于 Router 按 message id 路由。
// 消息实现此接口后，Router 会用 MessageID() 查找注册的 handler。
type Identifiable interface {
	MessageID() int
}

// Correlatable 用于请求-响应关联的请求侧。
// 消息实现此接口后，Session.Request 会用 TID() 作为关联 id 注册到 ReplyPool。
type Correlatable interface {
	TID() string
}

// Response 用于请求-响应关联的回包侧。
// 消息实现此接口后，框架在 OnTraffic 中自动识别为回包，
// 用 ResponseTID() 匹配在途请求并完成对应 Promise，不进业务 handler。
type Response interface {
	ResponseTID() string
}

// ClientIdentifiable 用于设备注册场景，提供业务身份（设备号等）。
// 消息实现此接口后，可配合 ServerConn/ClientConn.BindClientID 绑定业务 ID。
// 非必需：无设备身份概念的协议可不实现，Session 使用框架分配的会话 ID。
type ClientIdentifiable interface {
	ClientID() string
}

// PacketContextProvider 用于协议头上下文传递。
// 消息实现此接口后，框架在 dispatch 阶段读出 PacketContext 并注入 handler/reply 的
// context.Context（key=PacketContextKey）。Codec.Encode 从 ctx 中取回该值用于回包
// 时填写 ack/reply-seq 等协议头字段。
//
// PacketContext 的生命周期与消息一致（请求级），不存在连接级共享状态问题。
//
// 用法：
//
//	// Codec.Decode 返回带协议头的消息:
//	type myMsg struct { Seq uint32; Ack uint32; Body string }
//	func (m *myMsg) PacketContext() any { return &myCtx{Seq: m.Seq, Ack: m.Ack} }
//
//	// Codec.Encode 从 ctx 取回:
//	pc, _ := ctx.Value(PacketContextKey).(*myCtx)
type PacketContextProvider interface {
	PacketContext() any
}

type packetContextKeyType struct{}

// PacketContextKey 是协议包头在 context.Context 中的标准存取键。
//
// 框架在 dispatch 阶段：若消息实现了 PacketContextProvider，则
//
//	ctx = context.WithValue(ctx, PacketContextKey, msg.PacketContext())
//
// Codec.Encode 从 ctx 中取出 PacketContextKey 对应的值，用于回包时填写 ack/reply-seq
// 等协议头字段。如果 ctx 中不存在该值（主动发起的请求，没有入站请求上下文），
// Codec.Encode 生成新协议头。
//
// 读取时做类型断言：
//
//	pc, _ := ctx.Value(PacketContextKey).(*myPacketContext)
var PacketContextKey = &packetContextKeyType{}
