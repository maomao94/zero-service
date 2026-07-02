package gnetx

import (
	"context"
	"time"
)

// Handler 是 gnetx 的统一消息处理入口。
// Server/Client 的 OnTraffic 解码后会调用 Handler.Handle。
// ctx 携带 OTel trace context（每报文一个 span），可传入下游 gRPC/RPC 调用串起全链路。
// 返回 (reply, err)：reply 非 nil 框架编码后回包；err 进日志/interceptor。
//
// 实现者：
//   - HandlerFunc：把函数适配为 Handler。
//   - Router：按 messageID 路由的可注册容器（实现 Handler）。
//   - 用户自定义：只要实现 Handle 方法即可。
type Handler interface {
	Handle(ctx context.Context, sess *Session, msg any) (any, error)
}

// HandlerFunc 把函数适配为 Handler。
type HandlerFunc func(ctx context.Context, sess *Session, msg any) (any, error)

// Handle 实现 Handler。
func (f HandlerFunc) Handle(ctx context.Context, sess *Session, msg any) (any, error) {
	return f(ctx, sess, msg)
}

// AsyncHandler 是一个可选标记接口：实现 IsAsync() 且返回 true 的 Handler，
// 会被框架 offload 到 gnet 自带的 goroutine worker pool 执行，回包走 AsyncWrite（off-loop 安全）。
// 默认（未实现 IsAsync 或返回 false）则 on-loop 同步执行，回包走 c.Write。
//
// 重活（DB/下游 RPC/大计算）的 handler 应用 Async/AsyncFunc 包装标记为异步，
// 避免阻塞 event-loop 拖死同 loop 上其他连接。
type AsyncHandler interface {
	IsAsync() bool
}

// asyncFlag 是一个便捷的 AsyncHandler 实现，包装任意 Handler 标记为异步。
type asyncFlag struct {
	Handler
}

// IsAsync 返回 true，表示包装的 handler 应被 offload。
func (asyncFlag) IsAsync() bool { return true }

// Async 把任意 Handler（含 HandlerFunc、Router 的子 handler）标记为异步执行。
// 返回的 Handler 同时实现 AsyncHandler 接口。
func Async(h Handler) Handler {
	return asyncFlag{Handler: h}
}

// AsyncFunc 把函数标记为异步 handler。
func AsyncFunc(fn HandlerFunc) Handler {
	return asyncFlag{Handler: fn}
}

// slowHandlerWarning 是 on-loop 同步 handler 的慢处理告警阈值。
// 超过此阈值的 handler 会打 logx 慢处理日志。可通过 options 调整。
const defaultSlowHandlerThreshold = 50 * time.Millisecond

// defaultReconnectInterval 是 Client 断线后的默认固定重连间隔。
const defaultReconnectInterval = 3 * time.Second
