# Request-Response（opt-in，tid 响应式）

> **EXPERIMENTAL** — 此包尚未经过生产环境验证。

请求-响应是 gnetx 的 opt-in 层，仅在消息实现 `Correlatable`/`Response` 时启用。纯推送协议完全不依赖此层。

关键 source：`common/gnetx/request.go:1-27`、`common/gnetx/session.go:135-184`

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
// 请求实现 Correlatable
type MyReq struct { Serial int }
func (m *MyReq) TID() string { return strconv.Itoa(m.Serial) }

// 回包实现 Response
type MyResp struct { RespSerial int }
func (m *MyResp) ResponseTID() string { return strconv.Itoa(m.RespSerial) }

// 业务 goroutine（或 AsyncHandler）中发请求等回包
resp, err := sess.Request(ctx, &MyReq{Serial: 1}, 10*time.Second)
if err != nil { /* 超时/取消/断连 */ }
typed := resp.(*MyResp)
```

**线程约束**：只能在业务 goroutine 或 AsyncHandler 内调用。**严禁**在 event-loop handler 同步路径调用——会阻塞 event-loop，框架无法拦截。

## 回包匹配

`OnTraffic` 中（`common/gnetx/client.go:206-210` / `server.go:218-222`）：

```go
if resp, ok := msg.(Response); ok {
    if sess.resolveResponse(resp.ResponseTID(), msg) {
        continue // 命中在途 → 完成 Promise → 跳过 handler
    }
}
// 未命中 → 进 handler 当意外报文
```

## ReplyPool 生命周期

- 每 Session 一个，懒创建（`common/gnetx/session.go:160-173`）
- 默认 TTL 30s
- `Session.Close` 时 `pool.Close()`，在途请求全部 `Reject(ErrReplyClosed)`

## Client 端

```go
// 方式 A：通过 Session
resp, err := cli.Session().Request(ctx, req, ttl)

// 方式 B：通过 Client（等价，内部判 nil）
resp, err := cli.Request(ctx, req, ttl)
```

`common/gnetx/client.go:139-145`

## Server 端双向 Request

Server 也可通过已连接 Session 主动向 client 发请求：

1. `srv.Manager().Get(alias)` 获取 client 的 Session
2. 业务 goroutine 中 `sess.Request(ctx, req, ttl)`
3. client handler 识别请求并回包
4. server OnTraffic 识别回包为 Response → resolveResponse

测试：`common/gnetx/bidirectional_test.go:19-89`

## 测试覆盖

- `common/gnetx/client_test.go:TestClientRequestResponse` — 正常流程
- `common/gnetx/client_test.go:TestClientRequestTimeout` — 超时
- `common/gnetx/client_test.go:TestClientRequestViaClient` — Client 级 API
- `common/gnetx/bidirectional_test.go:TestServerInitiatedRequest` — Server 主动请求

## 常见错误

| 错误 | 说明 |
|------|------|
| on-loop handler 调 `sess.Request` | 阻塞 event-loop |
| `ResponseTID` 与 `TID` 不匹配 | 永不 resolve，等超时 |
| `sess.Close()` 后仍等结果 | `Reject(ErrReplyClosed)` |
| ttl 过大或不设 | 断连时 goroutine 泄漏 |
