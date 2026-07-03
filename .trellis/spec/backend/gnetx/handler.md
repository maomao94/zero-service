# Handler & Router

## Handler 接口

```go
// common/gnetx/handler.go:7-15
type Handler interface {
    Handle(ctx context.Context, conn Conn, msg any) (any, error)
}

type HandlerFunc func(ctx context.Context, conn Conn, msg any) (any, error)
```

第二个参数是 `Conn` 接口（`common/gnetx/session.go:22-33`），提供 `ID()`、`Send()`、`Close()` 等方法。Server/Client/Dialer 传入的 `*session` 满足该接口。

- `ctx` — OTel trace context
- `reply` — 非 nil 框架编码回包（sync→`c.Write` / async→`AsyncWrite`）
- `err` — 进日志/`span.RecordError`

### 同步 vs 异步

| 类型 | 执行线程 | 回包方式 | 适用场景 |
|------|---------|---------|---------|
| 同步（默认） | event-loop | `c.Write`（on-loop） | 快操作 |
| 异步 | gnet worker pool | `AsyncWrite`（off-loop） | 重操作 |

```go
gnetx.Async(myHandler)
gnetx.AsyncFunc(func(ctx context.Context, conn Conn, msg any) (any, error) { ... })
```

分发决策：`common/gnetx/handler.go` `isAsync()`

## Router（opt-in 层）

按 `messageID` 路由的 Handler 容器。消息需实现 `Identifiable`（`MessageID() int`）。Router 本身实现 `Handler`。

`common/gnetx/router.go`

```go
r := gnetx.NewRouter(defaultWorkerPool())
r.RegisterFunc(1, handler1)
gnetx.HandleTyped(r, 1, func(ctx context.Context, conn Conn, msg MyMsg) (any, error) { ... })
r.Async(1)           // 标记异步
r.FallbackFunc(f)    // 兜底
r.RegisterType(1, func() any { return &MyMsg{} })
```

### 路由决策（`common/gnetx/router.go:40+`）

1. msg 未实现 `Identifiable` → fallback；无则 `ErrNoHandler`
2. id 未注册 → fallback
3. handler 有 `isAsync` + pool 非 nil → offload
4. 否则同步执行

## 常见错误

| 错误 | 说明 |
|------|------|
| sync handler 做 IO/重操作 | 阻塞 event-loop |
| `HandleTyped` 类型与 wire id 不匹配 | 运行时 `typeMismatchErr` |
| Router 不传 fallback | 未知 id 返回 `ErrNoHandler` |
