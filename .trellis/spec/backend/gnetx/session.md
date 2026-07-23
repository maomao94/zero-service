# Session & SessionManager

`Session` 是每连接上下文，封装 `gnet.Conn` 并提供业务层 API。

关键 source：`common/gnetx/session.go`

## Session 结构

```go
// common/gnetx/session.go:36-52
type session struct {
    sessionID   string
    clientID    string
    gc          gnet.Conn
    localAddr   net.Addr
    remoteAddr  net.Addr
    codec       Codec
    mgr         *SessionManager
    created     time.Time
    lastActive  atomic.Int64
    sendSeq     atomic.Uint64
    attrs       sync.Map
    replyPool   *antsx.ReplyPool[any]   // 非拥有型引用，指向 Server/Client 共享池
    stateMu     sync.RWMutex
    closeOnce   sync.Once
    closed      bool
}
```

### 设计要点

- **`replyPool` 是非拥有型引用** — 指向 Server 或 Client 的共享池。Session 不创建也不关闭 ReplyPool。
- **连接地址在创建时快照** — `gnet.Conn.LocalAddr/RemoteAddr` 只能在 event-loop 使用；Session 对外返回不可变地址副本。
- **无 `ensurePool()` 方法** — 池生命周期由 Server/Client/Dialer 管理
- **`session` 是 unexported** — 外部通过 `Conn` / `ServerConn` / `ClientConn` 接口使用

## newSession

```go
// common/gnetx/session.go:54-66
func newSession(sessionID string, gc gnet.Conn, codec Codec, mgr *SessionManager, replyPool *antsx.ReplyPool[any], sequenceStart ...uint64) *session
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
| `Request(ctx, msg, ttl)` | off-loop ✅ | 复合 TID 注册 + 注入 session log fields + 阻塞等回包 |
| `Close()` | off-loop ✅ | 幂等；不关 replyPool（上层管理） |
| `SetAttribute`/`Attribute` | 并发 ✅ | `sync.Map` |
| `SessionID()` | 并发 ✅ | 返回框架分配的连接实例 ID |
| `ClientID()` | 并发 ✅ | 在 session 状态读锁下读取业务 ID，未绑定返回空串 |
| `BindClientID(clientID)` | 并发 ✅ | 校验并绑定业务 ID；冲突时新连接替换旧连接 |
| `LocalAddr()`/`RemoteAddr()` | 并发 ✅ | 返回 OnOpen 时保存的地址快照，不读取已释放的 gnet connection |

**禁止**在 event-loop（OnTraffic sync handler）中调 `Request()`。

## Scenario: 会话身份与客户端身份

### 1. Scope / Trigger

- 任何需要标识 TCP 连接、注册设备身份、按身份查询连接或处理重复登录的代码。
- `sessionID` 是连接实例身份；`clientID` 是注册后业务身份，两者禁止混用或统称 `alias`。

### 2. Signatures

```go
type Conn interface {
    SessionID() string
}

type ServerConn interface {
    Conn
    ClientID() string
    BindClientID(clientID string) error
}

type ClientConn interface {
    Conn
    ClientID() string
    BindClientID(clientID string) error
    Request(ctx context.Context, msg Correlatable, ttl time.Duration) (any, error)
}

func (m *SessionManager) GetBySessionID(sessionID string) Conn
func (m *SessionManager) GetByClientID(clientID string) Conn
```

### 3. Contracts

- `SessionID()` 从创建到关闭不变，由框架生成。
- `ClientID()` 在注册前为空；绑定、重新绑定和并发读取统一使用 session 状态锁。
- `mgr=nil` 的客户端 Session 不进入 SessionManager 索引；`closed` 和 `clientID` 由 session 状态锁保护，关闭完成后的绑定必须返回 `ErrSessionClosed`。
- Server/Client/Dialer 的 `OnClose` 只通过 `closeFromEventLoop` 更新 Session 状态；底层 gnet 已在执行关闭，不得再次调用 `gnet.Conn.Close`。
- 同一 session 重新绑定时，旧 client ID 索引必须删除。
- 新 session 绑定已占用的 client ID 时，新索引先生效，再关闭旧 session；旧 session 的关闭清理不得删除新索引。
- `SessionManager` 按 `m.mu -> session.stateMu` 顺序检查和更新 session；冲突淘汰旧 session 时，必须在 `m.mu` 内先标记旧 session 已关闭，阻止它在延迟 Close 前重新绑定。
- `add` 只能将未关闭 session 放入 `bySessionID`；若关闭先发生，后续 add 必须跳过。
- `BindClientID` 只接受仍由 `bySessionID` 管理的 session，禁止产生仅存在于 `byClientID` 的孤立索引。
- `GetBySessionID`、`GetByClientID`、`All` 和 `Count` 不应暴露已关闭 session。
- `GetBySessionID` 与 `GetByClientID` 是两个独立命名空间，即使字符串相同也返回各自索引的 session。

### 4. Validation & Error Matrix

| 条件 | 结果 |
|------|------|
| `clientID == ""` | `ErrInvalidClientID` |
| session 已关闭 | `ErrSessionClosed` |
| client ID 未找到 | `GetByClientID` 返回 nil |
| session ID 未找到 | `GetBySessionID` 返回 nil |
| client ID 冲突 | 新 session 可查询，旧 session 被关闭 |

### 5. Good/Base/Bad Cases

- Good：注册 handler 调 `serverConn.BindClientID(req.SendCode)` 并处理 error。
- Base：未注册连接只通过 `SessionID()` 管理，`ClientID()` 返回空串。
- Bad：用一个 `Get(idOrAlias)` 猜测调用方意图，或绕过 session 状态锁直接读写业务身份。

### 6. Tests Required

- 重新绑定：断言旧 client ID 查不到，新 client ID 指向当前 session。
- 冲突绑定：断言新 session 可查询、旧 session 已关闭。
- 命名空间：使用相同字符串，分别断言 session ID/client ID 查询结果。
- 并发绑定/读取：`go test -race` 无数据竞争，最终 client ID 索引只保留一个。
- 客户端 Session 并发绑定/关闭：关闭完成后再次绑定返回 `ErrSessionClosed`。
- 空 ID、关闭后绑定：分别断言 `ErrInvalidClientID`、`ErrSessionClosed`。

### 7. Wrong vs Correct

```go
// Wrong: 名称和查询来源不明确。
conn.Register(alias)
conn = manager.Get(idOrAlias)

// Correct: 明确业务身份和查询命名空间。
if err := conn.BindClientID(clientID); err != nil { return err }
conn = manager.GetByClientID(clientID)
```

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
Session.sessionID = "127.0.0.1:54321"
msg.TID()  = "1"
────────────────────
registerKey = "127.0.0.1:54321|1"
resolveKey  = "127.0.0.1:54321|1"
```

```go
// common/gnetx/session.go:103-112
func (s *session) Request(ctx context.Context, msg Correlatable, ttl time.Duration) (any, error) {
    compositeTID := s.sessionID + "|" + msg.TID()
    return antsx.RequestReply[any](ctx, s.replyPool, compositeTID, ...)
}

// common/gnetx/session.go:114-120
func (s *session) resolveResponse(tid string, resp any) bool {
    compositeTID := s.sessionID + "|" + tid
    return s.replyPool.Resolve(compositeTID, resp)
}
```

- 分隔符 `"|"`，永不拆分，仅作 key
- `resolveResponse` 内部拼接，调用方无需感知
- 重连后旧 session 的复合 TID 与新 session 不冲突（session ID 不同）

## SessionManager

管理所有活跃 Session，分别按 session ID 和 client ID 查找。Server 持有；Client/Dialer 不使用（mgr=nil）。

`common/gnetx/session.go:138-213`

```go
mgr := NewSessionManager(listener)  // nil listener → noop
mgr.GetBySessionID(sessionID)
mgr.GetByClientID(clientID)
mgr.All()                           // 快照
mgr.Count()
```

### Client ID 重复绑定与冲突

`common/gnetx/session.go:191-201`

同一 session 重新绑定 client ID 时，在锁内删除旧索引并写入新索引。不同 session 绑定相同 client ID 时，锁内先令新索引生效，解锁后关闭旧 session；`remove` 只在索引仍指向自身时删除，防止旧连接清理新映射。

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
| `Server` | `NewServer()` | `Shutdown()`；`Run()` 启动失败时同步关闭 | 所有 server session 共享 |
| `Client` | `NewClient()` | `Close()` | 该 client 的 session 共享 |
| `Dialer` | **无** | — | nil（不支持 Request） |

Client 断连重连时旧 Session 关闭但不关池，新 Session 仍用同一个池。在途请求等 TTL 过期（默认 30s）。

## 常见错误

| 错误 | 说明 |
|------|------|
| on-loop 调 `sess.Request` | 阻塞 event-loop |
| client ID 冲突后仍用旧 Session 指针 | 旧 Session 已 Close |
| 使用模糊 `Get(idOrAlias)` | session ID 与 client ID 字符串相同时查询语义不确定；必须选明确方法 |
| `SetContext` 在非 event-loop 线程调用 | 与 OnClose 数据竞争 |
| Dialer session 上调 `Request()` | replyPool 为 nil，返回 `ErrSessionClosed` |

## Session Log Fields 注入

`injectSessionLogFields`（`session.go:243`）在 dispatch 和 `Request` 时将 session 信息注入 context：

```go
func injectSessionLogFields(ctx context.Context, s *session) context.Context {
    ctx = logx.ContextWithFields(ctx,
        logx.Field("sessionID", s.sessionID),
        logx.Field("local", s.LocalAddr().String()),
        logx.Field("remote", s.RemoteAddr().String()),
    )
    if clientID := s.ClientID(); clientID != "" {
        ctx = logx.ContextWithFields(ctx, logx.Field("clientID", clientID))
    }
    return ctx
}
```

**注入时机：**
- Server/Client `dispatchSync` / `dispatchAsync` — handler 的 ctx 含 session 字段
- `session.Request()` — request-reply 发送路径的 ctx 含 session 字段

Handler 内部使用 `logx.WithContext(ctx)` 即可自动携带 `sessionID`、`local`、`remote`、`clientID`。不需要在各 handler 中重复注入。
