# Handler & Router

> **EXPERIMENTAL** — 此包尚未经过生产环境验证。

## Handler 接口

```go
// common/gnetx/handler.go:17-19
type Handler interface {
    Handle(ctx context.Context, sess *Session, msg any) (any, error)
}
```

- `ctx` — OTel trace context（每报文一个 span），可传入下游 RPC
- `reply` — 非 nil 框架编码回包（sync→`c.Write` / async→`AsyncWrite`）
- `err` — 进日志/`span.RecordError`，不 panic

### HandlerFunc

```go
// common/gnetx/handler.go:22-27
type HandlerFunc func(ctx context.Context, sess *Session, msg any) (any, error)
```

### 同步 vs 异步

| 类型 | 执行线程 | 回包方式 | 适用场景 |
|------|---------|---------|---------|
| 同步（默认） | event-loop | `c.Write`（on-loop） | 快操作：内存读写、简单计算 |
| 异步 | gnet worker pool | `AsyncWrite`（off-loop） | 重操作：DB/RPC/大计算 |

```go
// 标记异步
gnetx.Async(myHandler)
gnetx.AsyncFunc(func(ctx context.Context, s *Session, msg any) (any, error) {
    // 可做 IO/重操作
})
```

分发决策：`common/gnetx/handler.go:367-370`

```go
func isAsync(h Handler) bool {
    ah, ok := h.(AsyncHandler)
    return ok && ah.IsAsync()
}
```

## Router（opt-in 层）

按 `messageID` 路由的 Handler 容器。消息需实现 `Identifiable`（`MessageID() int`）。Router 本身实现 `Handler`，可直接传给 Server/Client。

`common/gnetx/router.go:23-29`

### 基本用法

```go
r := gnetx.NewRouter(defaultWorkerPool())

// 按 id 注册
r.RegisterFunc(1, handler1)
r.Register(2, handler2)

// 类型安全注册（避免 handler 内手写 type-assert）
gnetx.HandleTyped(r, 1, func(ctx context.Context, s *Session, msg MyMsg) (any, error) {

// 标记异步
r.Async(1)

// 兜底
r.FallbackFunc(fallbackHandler)

// 消息工厂（供 decoder 按 wire id 实例化）
r.RegisterType(1, func() any { return &MyMsg{} })
```

### 路由决策

`common/gnetx/router.go:46-68`（`Handle`）、`:71-95`（`lookup`）：

1. msg 未实现 `Identifiable` → fallback；无 fallback 返回 `ErrNoHandler`
2. id 未注册 → fallback；无 fallback 返回 `ErrNoHandler`
3. handler 有 `isAsync` + pool 非 nil → offload
4. 否则同步执行

### 无 pool 时异步回退

pool 传 `nil` 时 Async 标记不生效，handler 同步执行。
测试：`common/gnetx/router_test.go:160-175`

## 常见错误

| 错误 | 说明 |
|------|------|
| sync handler 做 IO/重操作 | 阻塞 event-loop |
| `HandleTyped` 类型与 wire id 不匹配 | 运行时 `typeMismatchErr` |
| Router 不传 fallback | 未知 id 返回 `ErrNoHandler`，静默丢弃 |
