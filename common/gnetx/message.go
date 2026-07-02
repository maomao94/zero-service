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
// 消息实现此接口后，可配合 Session.Register 把会话按业务 id 纳入 SessionManager。
// 非必需：无设备身份概念的协议可不实现，Session 用框架分配的 id。
type ClientIdentifiable interface {
	ClientID() string
}
