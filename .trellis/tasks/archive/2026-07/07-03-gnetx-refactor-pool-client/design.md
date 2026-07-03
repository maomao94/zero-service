# gnetx 重构：技术设计

## 1. 架构变更概览

```
Before:                                     After:
┌─────────────────────┐                    ┌─────────────────────────┐
│ Server              │                    │ Server                  │
│  ├─ Session1 ─ pool1│                    │  ├─ replyPool (shared)  │
│  ├─ Session2 ─ pool2│  ──refactor──►    │  ├─ Session1            │
│  └─ Session3 ─ pool3│                    │  ├─ Session2            │
└─────────────────────┘                    │  └─ Session3            │
                                           └─────────────────────────┘

┌─────────────────────┐                    ┌─────────────────────────┐
│ Client (long conn)  │                    │ Client (long conn)      │
│  └─ Session ─ pool  │  ──refactor──►    │  ├─ replyPool (shared)  │
└─────────────────────┘                    │  └─ Session             │
                                           └─────────────────────────┘
                                           ┌─────────────────────────┐
                                           │ Dialer (new, short)     │
                                           │  ├─ replyPool (shared)  │
                                           │  └─ Session* (per dial) │
                                           └─────────────────────────┘
```

## 2. Session 变更

### 2.1 字段变更

```go
// 移除:
pool     atomic.Pointer[antsx.ReplyPool[any]]
poolOnce sync.Once

// 新增（非拥有型引用，指向共享池）:
replyPool *antsx.ReplyPool[any]

// 新增（Dialer 使用时的额外清理回调）:
extraClose func()
```

### 2.2 newSession 签名变更

```go
// Before:
func newSession(id string, conn gnet.Conn, codec Codec, mgr *SessionManager, isClient bool) *Session

// After:
func newSession(id string, conn gnet.Conn, codec Codec, mgr *SessionManager, isClient bool, replyPool *antsx.ReplyPool[any]) *Session
```

### 2.3 方法变更

| 方法 | 变更 |
|------|------|
| `ensurePool()` | **移除** |
| `Request(ctx, msg, ttl)` | 使用 `s.id + "\|" + msg.TID()` 作为复合 TID，操作 `s.replyPool` |
| `resolveResponse(tid, resp)` | 内部构造 `s.id + "\|" + tid` 作为复合 TID，操作 `s.replyPool` |
| `Close()` | **不再关闭池**（池由上层 Server/Client/Dialer 管理）；新增执行 `extraClose()` |

### 2.4 复合 TID 机制

```
Session.id = "127.0.0.1:54321"
msg.TID()  = "1"
────────────────────
registerKey = "127.0.0.1:54321|1"
resolveKey  = "127.0.0.1:54321|1"
```

- 分隔符 `"|"`，永不拆分，仅作 key
- `resolveResponse` 内部拼接，调用方无需感知

## 3. Server 变更

### 3.1 新增字段

```go
replyPool *antsx.ReplyPool[any]  // 在 NewServer 中创建
```

### 3.2 生命周期

```
NewServer()  → 创建 replyPool
  ├─ OnOpen → newSession(..., s.replyPool) 
  │            Session 持有非拥有型引用
  ├─ OnTraffic → sess.resolveResponse()
  │              内部用 sess.id + "|" + tid
  └─ Shutdown/Stop → replyPool.Close()
```

### 3.3 NewServer 变更

```go
func NewServer(opts ...ServerOption) (*Server, error) {
    // ... existing validation ...
    
    replyPool := antsx.NewReplyPool[any](
        antsx.WithName("gnetx-server-" + normalizeAddrForMetrics(o.Addr)),
        antsx.WithDefaultTTL(30*time.Second),
    )
    
    s := &Server{
        // ...
        replyPool: replyPool,
    }
    return s, nil
}
```

### 3.4 newSession 调用点

```go
// OnOpen:
sess := newSession(sessionIDForConn(c), c, s.opts.Codec, s.mgr, false, s.replyPool)
```

### 3.5 OnTraffic 中的 resolveResponse 调用不变

```go
if resp, ok := msg.(Response); ok {
    if sess.resolveResponse(resp.ResponseTID(), msg) {  // 不变
        continue
    }
}
```

`resolveResponse` 内部负责拼接复合 TID。

## 4. Client 变更（长连接）

### 4.1 新增字段

```go
replyPool *antsx.ReplyPool[any]  // 在 NewClient 中创建
```

### 4.2 生命周期

```
NewClient()  → 创建 replyPool
  ├─ OnOpen → newSession(..., c.replyPool)
  ├─ 断连重连 → 旧 Session 关闭但不关池（在途请求等 TTL 过期）
  │            新 Session 仍用 c.replyPool
  └─ Close() → replyPool.Close() + gcli.Stop()
```

### 4.3 newSession 调用点

```go
// Client.OnOpen:
sess := newSession(sessionIDForConn(c), c, c.opts.Codec, nil, true, c.replyPool)
```

## 5. Dialer 实现（新增）

### 5.1 核心思路

Dialer 不持有 ReplyPool，短连接场景下直接用 `antsx.Promise` 做请求-响应匹配：
- 一次 dial → 发一个请求 → 等回包 → 关连接
- ctx 控制超时，Promise.Await(ctx) 阻塞等待
- 无需 TimingWheel、无需 TTL、无需复合 TID

### 5.2 结构

```go
// Dialer 是短连接 TCP 客户端，类似 HTTP client 的 Dial 模式。
//
// 用法 A — Request 一步完成：
//   dialer := gnetx.NewDialer(gnetx.WithClientCodec(myCodec))
//   resp, err := dialer.Request(ctx, "tcp", "127.0.0.1:8080", req)
//
// 用法 B — Dial 拿到 Session 手动控制：
//   sess, err := dialer.Dial(ctx, "tcp", "127.0.0.1:8080")
//   defer sess.Close()
//   sess.Send(msg)  // fire-and-forget（Dial 返回的 Session 不支持 Request）
type Dialer struct {
    opts   ClientOptions
    closed atomic.Bool
}
```

### 5.3 方法

```go
func NewDialer(opts ...ClientOption) *Dialer
func (d *Dialer) Dial(ctx context.Context, network, address string) (*Session, error)
func (d *Dialer) Request(ctx context.Context, network, address string, msg Correlatable) (any, error)
func (d *Dialer) Close() error
```

### 5.4 Dial 实现

```
Dial() 流程:
1. 创建 dialAdapter（含 Promise）
2. 创建 gnet.Client + Start
3. gnet.Client.DialContext() → OnOpen 创建 Session，发送请求消息
4. 返回 Session（replyPool=nil，仅支持 Send/Notify）
5. Session.extraClose = func() { gcli.Stop() }
```

### 5.5 Request 实现（基于 Promise，无 ReplyPool）

```
Request() 流程:
1. promise := antsx.NewPromise[any]()
2. 创建 dialAdapter{ promise: promise, reqMsg: msg }
3. 创建 gnet.Client(da) + Start + DialContext
4. OnOpen: 编码 msg 并发送；存 Session
5. OnTraffic: 解码 → 若为 Response 且 TID 匹配 → promise.Resolve(resp)
6. OnClose: promise.Reject(err)（若未 resolve）
7. 阻塞 promise.Await(ctx) 等待结果
8. 无论成功失败: sess.Close() + gcli.Stop()
```

不需要 ReplyPool，不需要 TTL，ctx 控制整体超时。

### 5.6 dialAdapter

```go
type dialAdapter struct {
    gnet.BuiltinEventEngine
    opts    ClientOptions
    promise *antsx.Promise[any]   // 回包时 resolve
    tid     string                 // 期望匹配的 TID
    sess    *Session
    once    sync.Once
}

// OnOpen: 创建 Session，编码并发送请求
// OnTraffic: 解码帧，若 Response.ResponseTID() == tid → resolve promise
// OnClose: reject promise（若未 resolve）
```
```

## 6. ReplyPool 所有权总结

| 结构 | 创建时机 | 销毁时机 | 传递给 Session |
|------|---------|---------|---------------|
| `Server.replyPool` | `NewServer()` | `Shutdown()`/`Stop()` | 所有 server Session |
| `Client.replyPool` | `NewClient()` | `Client.Close()` | 该 Client 的 Session |
| `Dialer` | **无** | — | Session.replyPool=nil，不支持 Request |

Dialer 短连接直接用 `antsx.Promise` 做单次请求-响应，无需 ReplyPool。

## 7. 线程安全

- Server/Client/Dialer 的 `replyPool` 在构造后不变，无需锁保护
- Session 的 `replyPool` 在 `newSession` 中设置，后续只读，无需原子操作
- `antsx.ReplyPool` 自身内部是并发安全的（mutex + atomic）
- `resolveResponse` 在 on-loop 调用，与 off-loop 的 Request goroutine 通过 `antsx.ReplyPool` 同步

## 8. 兼容性

### 公开 API 不变
- `Correlatable`、`Response` 接口不变
- `Session.Request(ctx, msg, ttl)` 签名不变
- `Session.Send(msg)` 签名不变
- `Client.Request/MustNewClient/NewClient` 签名不变
- `Server.NewServer/Start/Stop/Manager` 签名不变
- `ServerOptions`/`ClientOptions` 不变

### 内部变更
- `newSession` 签名变更（但为 unexported）
- `ensurePool` 移除（unexported）
- `resolveResponse` 内部逻辑变，签名不变

### Dialer 向后兼容
- Dialer 使用 `ClientOption` 系列选项，无需新增选项类型
- 若需要 Dialer 特定选项，可后续添加 `DialerOption`

## 9. 风险与回滚

- **风险1**: 复合 TID 碰撞。sessionID 从 `RemoteAddr().String()` 得来（如 "127.0.0.1:54321"），加上 `|` + 业务 TID，碰撞概率极低
- **风险2**: 断连重连时旧 TID 条目残留。设计选择让 TTL 自然过期（30s 默认），不主动清理
- **回滚**: 如果共享池有问题，可恢复 `ensurePool()` 机制，只需回退 `session.go` 和 `server.go`/`client.go` 的 `newSession` 调用
