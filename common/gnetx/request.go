package gnetx

// 本文件是请求-响应（tid 响应式）opt-in 层的文档说明。
// 核心实现（Session.Request/ensurePool/resolveResponse）在 session.go 中，
// 关联引擎为 common/antsx.ReplyPool[any]（每 Session 一个，懒创建，TTL 默认 30s）。
//
// 使用范式：
//
//	// 请求侧消息实现 Correlatable（TID 提供关联 id）
//	type MyReq struct { SerialNo int }
//	func (m *MyReq) TID() string { return strconv.Itoa(m.SerialNo) }
//
//	// 回包侧消息实现 Response（ResponseTID 与请求 TID 对应）
//	type MyResp struct { RespSerialNo int }
//	func (m *MyResp) ResponseTID() string { return strconv.Itoa(m.RespSerialNo) }
//
//	// 业务 goroutine（或 AsyncHandler 内）发请求并等回包；严禁在同步 handler(on-loop) 调用
//	resp, err := sess.Request(ctx, &MyReq{SerialNo: 1}, 10*time.Second)
//	if err != nil { ... }
//	typed := resp.(*MyResp)
//
// 回包匹配流程：入站消息实现 Response → OnTraffic 中框架自动调
// sess.resolveResponse(msg.ResponseTID(), msg) → 命中则 ReplyPool.Resolve 完成 Promise，
// 跳过业务 handler；未命中（无在途或 pool 未建）则落到 handler 当意外报文处理。
//
// 生命周期：ReplyPool 与 Session 绑定，Session.Close 时自动 pool.Close，
// 所有在途请求被 Reject(ErrReplyClosed)。详见 antsx-replypool-guidelines.md。
