# Request-Response（opt-in，tid 响应式）

请求-响应是 gnetx 的 opt-in 层，仅在消息实现 `Correlatable`/`Response` 时启用。

关键 source：`common/gnetx/message.go`、`common/gnetx/session.go:103-120`

## 接口契约

```go
// common/gnetx/message.go:16-19
type Correlatable interface {
    TID() string
}

// common/gnetx/message.go:23-26
type Response interface {
    ResponseTID() string
}
```

## 使用范式

```go
type MyReq struct { Serial int }
func (m *MyReq) TID() string { return strconv.Itoa(m.Serial) }

type MyResp struct { RespSerial int }
func (m *MyResp) ResponseTID() string { return strconv.Itoa(m.RespSerial) }

resp, err := sess.Request(ctx, &MyReq{Serial: 1}, 10*time.Second)
typed := resp.(*MyResp)
```

**线程约束**：只能在业务 goroutine 或 AsyncHandler 内调用。严禁在 event-loop 同步 handler 中调用。

## 复合 TID 匹配

Server 和 Client 共享全局 `ReplyPool`，TID 需带上 session 前缀避免冲突：

```
Session.id = "127.0.0.1:54321"
msg.TID()  = "1"
────────────────────
registerKey = "127.0.0.1:54321|1"
```

- `Request()`: `compositeTID := s.id + "|" + msg.TID()` → `pool.Register`
- `resolveResponse()`: `compositeTID := s.id + "|" + tid` → `pool.Resolve`
- 调用方无需感知复合 TID，`Correlatable.TID()` 和 `Response.ResponseTID()` 接口不变

`common/gnetx/session.go:103-120`

## 回包匹配流程

`OnTraffic` 中（`common/gnetx/server.go:201-205` / `client.go:174-178`）：

```go
if resp, ok := msg.(Response); ok {
    if cn.resolveResponse(resp.ResponseTID(), msg) {
        continue  // 命中在途 → 完成 Promise → 跳过 handler
    }
}
// 未命中 → 进 handler 当意外报文
```

## ReplyPool 生命周期

| 组件 | 创建 | 销毁 | 说明 |
|------|------|------|------|
| Server | `NewServer()` | `Shutdown()` (defer) | 所有 server session 共享 |
| Client | `NewClient()` | `Close()` | 重连后新 session 复用同一池 |
| Dialer | **无** | — | 用 `antsx.Promise` 直接匹配 |

Client 断连重连时旧 Session 关闭但不关池，在途请求等 TTL 过期（默认 30s）。

## Server 端双向 Request

Server 可通过已连接 Session 主动向 client 发请求：
1. `srv.Manager().Get(alias)` 获取 Session
2. 业务 goroutine 中 `sess.Request(ctx, req, ttl)`
3. Session 内部用复合 TID（`sessionID + "|" + tid`）注册到 Server 共享池

测试：`common/gnetx/bidirectional_test.go`

## 常见错误

| 错误 | 说明 |
|------|------|
| on-loop handler 调 `sess.Request` | 阻塞 event-loop |
| `ResponseTID` 与 `TID` 不匹配 | 永不 resolve |
| `sess.Close()` 后仍等结果 | `Reject(ErrReplyClosed)` |
| ttl 过大 | 断连时在途请求残留等 TTL 过期 |
