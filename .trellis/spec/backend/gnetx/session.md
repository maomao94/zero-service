# Session & SessionManager

`Session` 是每连接上下文，封装 `gnet.Conn` 并提供业务层 API。

关键 source：`common/gnetx/session.go`

## Session 结构

```go
// common/gnetx/session.go:36-52
type session struct {
    id          string
    alias       string
    gc          gnet.Conn
    codec       Codec
    mgr         *SessionManager
    created     time.Time
    lastActive  atomic.Int64
    sendSeq     atomic.Uint64
    attrs       sync.Map
    replyPool   *antsx.ReplyPool[any]   // 非拥有型引用，指向 Server/Client 共享池
    closeOnce   sync.Once
    closed      atomic.Bool
    closeFunc   func()                  // Dialer 额外清理（当前未使用）
}
```

### 设计要点

- **`replyPool` 是非拥有型引用** — 指向 Server 或 Client 的共享池。Session 不创建也不关闭 ReplyPool。
- **无 `ensurePool()` 方法** — 池生命周期由 Server/Client/Dialer 管理
- **`session` 是 unexported** — 外部通过 `Conn` / `ServerConn` / `ClientConn` 接口使用

## newSession

```go
// common/gnetx/session.go:54-66
func newSession(id string, gc gnet.Conn, codec Codec, mgr *SessionManager, replyPool *antsx.ReplyPool[any], sequenceStart ...uint64) *session
```

| 参数 | Server 调用 | Client 调用 | Dialer 调用 |
|------|-----------|------------|------------|
| `mgr` | `s.mgr` | `nil` | `nil` |
| `replyPool` | `s.replyPool` | `c.replyPool` | `nil` |

Dialer 的 session 无 replyPool，`Request()` 方法返回 `ErrSessionClosed`。

## 关键方法

| 方法 | 线程 | 说明 |
|------|------|------|
| `Write(ctx, msg)` | on-loop ✅ | 编码后同步 `gnet.Conn.Write`，用于 sync handler 回包 |
| `WriteAsync(ctx, msg)` | off-loop ✅ | 编码后 `gnet.Conn.AsyncWrite`，用于业务 goroutine / async handler |
| `NextSendSeq()` | 并发 ✅ | 返回当前发送序号并递增 |
| `Request(ctx, msg, ttl)` | off-loop ✅ | 复合 TID 注册 + 阻塞等回包 |
| `Close()` | off-loop ✅ | 幂等；不关 replyPool（上层管理） |
| `SetAttribute`/`Attribute` | 并发 ✅ | `sync.Map` |
| `Register(alias)` | 并发 ✅ | alias 冲突踢旧 |

**禁止**在 event-loop（OnTraffic sync handler）中调 `Request()`。

## 连接级发送序号

每个 Session 持有独立的 `sendSeq atomic.Uint64`。`ServerOptions.SequenceStart` / `ClientOptions.SequenceStart` 在新建 session 时写入初始值；不配置时默认为 0。

```go
func (s *session) NextSendSeq() uint64 {
    return s.sendSeq.Add(1) - 1
}
```

- 第一次调用返回 `SequenceStart`。
- 后续调用按连接递增。
- 框架只提供序号分配；Seq/TID/Ack/CorrelationID 如何映射由 Codec 决定。
- `Conn.Write` / `Conn.WriteAsync` 会把调用方 ctx 直接传给 `Codec.Encode`。

## 复合 TID 机制

为支持 Server/Client 全局共享 `ReplyPool`，`Request` 和 `resolveResponse` 使用复合 TID：

```
Session.id = "127.0.0.1:54321"
msg.TID()  = "1"
────────────────────
registerKey = "127.0.0.1:54321|1"
resolveKey  = "127.0.0.1:54321|1"
```

```go
// common/gnetx/session.go:103-112
func (s *session) Request(ctx context.Context, msg Correlatable, ttl time.Duration) (any, error) {
    compositeTID := s.id + "|" + msg.TID()
    return antsx.RequestReply[any](ctx, s.replyPool, compositeTID, ...)
}

// common/gnetx/session.go:114-120
func (s *session) resolveResponse(tid string, resp any) bool {
    compositeTID := s.id + "|" + tid
    return s.replyPool.Resolve(compositeTID, resp)
}
```

- 分隔符 `"|"`，永不拆分，仅作 key
- `resolveResponse` 内部拼接，调用方无需感知
- 重连后旧 session 的复合 TID 与新 session 不冲突（id 不同）

## SessionManager

管理所有活跃 Session，按 id 和 alias 查找。Server 持有；Client/Dialer 不使用（mgr=nil）。

`common/gnetx/session.go:138-213`

```go
mgr := NewSessionManager(listener)  // nil listener → noop
mgr.Get("id-or-alias")              // alias 优先
mgr.All()                           // 快照
mgr.Count()
```

### Alias 冲突

`common/gnetx/session.go:191-201`

同 alias 重复注册时踢旧：`Unlock → old.Close() → Lock → 写新映射`。

### SessionListener

```go
type SessionListener interface {
    OnCreated(s ServerConn)
    OnRegistered(s ServerConn)
    OnDestroyed(s ServerConn)
}
```

## ReplyPool 所有权

| 结构 | 创建 | 销毁 | Session.replyPool |
|------|------|------|------------------|
| `Server` | `NewServer()` | `Shutdown()` | 所有 server session 共享 |
| `Client` | `NewClient()` | `Close()` | 该 client 的 session 共享 |
| `Dialer` | **无** | — | nil（不支持 Request） |

Client 断连重连时旧 Session 关闭但不关池，新 Session 仍用同一个池。在途请求等 TTL 过期（默认 30s）。

## 常见错误

| 错误 | 说明 |
|------|------|
| on-loop 调 `sess.Request` | 阻塞 event-loop |
| alias 冲突后仍用旧 Session 指针 | 旧 Session 已 Close |
| `SetContext` 在非 event-loop 线程调用 | 与 OnClose 数据竞争 |
| Dialer session 上调 `Request()` | replyPool 为 nil，返回 `ErrSessionClosed` |
