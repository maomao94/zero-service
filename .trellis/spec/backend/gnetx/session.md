# Session & SessionManager

> **EXPERIMENTAL** — 此包尚未经过生产环境验证。

`Session` 是每连接上下文，封装 `gnet.Conn` 并提供业务层 API。

关键 source：`common/gnetx/session.go:25-211`

## Session 结构

```go
// common/gnetx/session.go:25-45
type Session struct {
    id       string           // 框架分配（远端地址派生）
    alias    string           // opt-in 业务 id（Register 时设置）
    conn     gnet.Conn
    codec    Codec            // Send/Request 编码用
    mgr      *SessionManager  // server 非 nil，client 为 nil
    isClient bool
    created  time.Time
    lastActive atomic.Int64   // unix nano，空闲扫描用
    attrs    sync.Map         // 业务属性
    pool     atomic.Pointer[antsx.ReplyPool[any]]  // 懒创建
    poolOnce sync.Once
}
```

## 关键方法

| 方法 | 线程安全 | 说明 |
|------|---------|------|
| `Send(msg)` | off-loop ✅ | 编码后 `AsyncWrite` |
| `Notify(ctx, msg)` | off-loop ✅ | `Send` 的语义别名 |
| `Request(ctx, msg, ttl)` | off-loop ✅ | 响应式请求，阻塞等回包 |
| `Close()` | off-loop ✅ | 幂等、从 mgr 移除、关 pool、关 conn |
| `SetAttribute`/`Attribute` | 并发 ✅ | `sync.Map` |
| `Register(alias)` | 并发 ✅ | alias 冲突踢旧 |
| `touch()` | on-loop | 更新 `lastActive`（atomic） |

**禁止**在 event-loop handler 同步路径调 `Session.Request`。

## SessionManager

管理所有活跃 Session，按 id 和 alias 查找。Server 持有；Client 不使用（mgr=nil）。

`common/gnetx/session.go:215-297`

```go
mgr := gnetx.NewSessionManager(listener)  // nil listener → noop

mgr.Get("id-or-alias")   // alias 优先，再查 byID
mgr.All()                 // 快照（只读锁）
mgr.Count()               // 只读锁
```

### Alias 冲突

`common/gnetx/session.go:273-284`

同 alias 重复注册时踢旧：`Unlock → old.Close() → Lock → 写新映射`。解锁-加锁避免死锁。

### Listener

```go
// common/gnetx/session.go:301-312
type SessionListener interface {
    OnCreated(s *Session)    // OnOpen 时
    OnRegistered(s *Session) // Register 时
    OnDestroyed(s *Session)  // OnClose/Close 时
}
```

嵌入 `noopSessionListener` 获全部空实现。

## Client Session 特殊性

Client Session 的 `mgr = nil`：`Register` 只设 `alias` 不写管理器；`Close` 跳 `mgr.remove`。

## ReplyPool

每 Session 一个 `antsx.ReplyPool[any]`，懒创建（首次 `Request` 时）。

`common/gnetx/session.go:160-173`

- `sync.Once` 防重
- 创建后查 `closed`：已关闭则立即 `pool.Close()` 防泄漏
- `atomic.Pointer` 存，与 `resolveResponse` 并发读无竞争

## 常见错误

| 错误 | 说明 |
|------|------|
| on-loop 调 `sess.Request` | 阻塞 event-loop |
| alias 冲突后仍用旧 Session 指针 | 旧 Session 已 Close |
| `SetContext` 在非 event-loop 线程调用 | 与 OnClose 数据竞争 |
