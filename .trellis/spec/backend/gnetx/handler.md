# Handler & Router

## Handler 接口

```go
// common/gnetx/handler.go:7-15
type Handler interface {
    Handle(ctx context.Context, conn Conn, msg any) (any, error)
}

type HandlerFunc func(ctx context.Context, conn Conn, msg any) (any, error)
```

第二个参数是 `Conn` 接口（`common/gnetx/session.go`），提供 `ID()`、`Write()`、`WriteAsync()`、`Close()` 等方法。Server/Client/Dialer 传入的 `*session` 满足该接口。

- `ctx` — OTel trace context，若入站消息实现了 `PacketContextProvider`，ctx 中额外注入 `PacketContextKey` 对应的协议包头
- `reply` — 非 nil 框架编码回包（sync→`c.Write` / async→`AsyncWrite`）；回包编码使用的 ctx 与 handler 收到的相同，`Codec.Encode` 可从 `ctx.Value(PacketContextKey)` 读取入站协议头填 ack
- `err` — 进日志/`span.RecordError`

### 同步 vs 异步

| 类型 | 执行线程 | 回包方式 | 适用场景 |
|------|---------|---------|---------|
| 同步（默认） | event-loop | `Conn.Write`（底层 `gnet.Conn.Write`） | 快操作 |
| 异步 | gnet worker pool | `Conn.WriteAsync`（底层 `gnet.Conn.AsyncWrite`） | 重操作 |

```go
gnetx.Async(myHandler)
gnetx.AsyncFunc(func(ctx context.Context, conn Conn, msg any) (any, error) { ... })
```

分发决策：`common/gnetx/handler.go` `isAsync()`

## Router（opt-in 层）

按 `messageID` 路由的 Handler 容器。消息需实现 `Identifiable`（`MessageID() int`）。Router 本身实现 `Handler` 和 `RouteResolver`。

Router 只负责消息到 handler 的匹配，不负责开 goroutine、不负责写回包、不直接调用 worker pool。sync/async 是 handler 自身属性，由业务侧注册 `Async(handler)`、`HandleTypedAsync`、`FallbackFuncAsync` 或 `Router.Async(id)` 声明；Server/Client 的 dispatch 统一判断并调度。

`common/gnetx/router.go`

```go
r := gnetx.NewRouter()
r.RegisterFunc(1, handler1) // 同步
r.Register(2, gnetx.Async(handler2))
gnetx.HandleTyped(r, 1, func(ctx context.Context, conn Conn, msg MyMsg) (any, error) { ... })
r.Async(1)             // 将已注册 handler 包装为 Async(handler)
gnetx.HandleTypedAsync(r, 3, handler)  // HandleTyped + Async 一步完成
r.FallbackFunc(f)    // 兜底（同步）
r.FallbackFuncAsync(f) // 兜底（异步）
```

### 分发决策

1. Server/Client `dispatch` 取根 handler。
2. 根 handler 实现 `RouteResolver` 时先 `Resolve(msg)` 得到业务 handler。
3. `isAsync(handler)` 为 true → `dispatchAsync` 入 worker pool，回包走 `WriteAsync`。
4. 否则 → `dispatchSync` 在 event-loop 执行，回包走 `Write`。

`Router.Handle` 仅用于直接把 Router 当普通 Handler 调用：内部 `Resolve(msg)` 后调用匹配 handler 的 `Handle`，不做 async 调度。

## 常见错误

| 错误 | 说明 |
|------|------|
| sync handler 做 IO/重操作 | 阻塞 event-loop |
| `HandleTyped` 类型与 wire id 不匹配 | 运行时 `typeMismatchErr` |
| Router 不传 fallback | 未知 id 返回 `ErrNoHandler` |
| 在 Router 内部写 reply 或提交 pool | 破坏 reply owner；应由 Server/Client dispatch 统一写 |
