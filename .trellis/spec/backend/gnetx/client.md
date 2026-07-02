# Client

> **EXPERIMENTAL** — 此包尚未经过生产环境验证。

`Client` 是 gnetx 的**单连接 TCP 客户端**，对标 `mqttx`/`modbusx` 的 `MustNewClient` 模型。一个 Client = 一个远端连接。

关键 source：`common/gnetx/client.go:36-363`

## 核心设计

- **构造即拨号** — 无 `Start`/`Stop`（仅 `NewClient` + `Close`）
- **固定间隔重连** — 断线按 `ReconnectInterval`（默认 3s）自动重连，非指数退避
- **多连接 = 多 Client** — 多连接管理是独立职责，不耦合进 Client
- **不实现 `go-zero service.Service`** — 生命周期仅构造 + `Close`

## 构造

```go
// MustNewClient：失败 panic + 自动注册 proc 关闭监听
cli := gnetx.MustNewClient("tcp", "127.0.0.1:9000",
    gnetx.WithClientCodec(myCodec),
    gnetx.WithClientHandler(myHandler),
    gnetx.WithClientMaxFrameLength(1 << 20),
    gnetx.WithClientReconnectInterval(5 * time.Second),
    gnetx.WithClientOnReady(func(c *gnetx.Client) {
        log.Println("connected")
    }),
)
defer cli.Close()

// NewClient：返回首次拨号错误
cli, err := gnetx.NewClient("tcp", "...", opts...)
```

必填项（`common/gnetx/options.go:297-308`）：`Codec`、`Handler`、`MaxFrameLength`

默认值（`common/gnetx/options.go:311-324`）：
- `ReconnectInterval` → 3s（`common/gnetx/handler.go:63`）
- `SlowHandlerThreshold` → 50ms
- `BatchReadLimit` → 64

## 响应式 API

```go
// fire-and-forget 发送
cli.Send(msg)                          // → sess.Send(msg)

// 带 ctx 的 fire-and-forget
cli.Notify(ctx, msg)                   // → sess.Notify(ctx, msg)

// 响应式请求：发请求等匹配 tid 的回包
resp, err := cli.Request(ctx, req, ttl) // → sess.Request(ctx, req, ttl)
```

`common/gnetx/client.go:118-145`

未连接或重连中时均返回 `ErrSessionClosed`（非 panic）。

## 生命周期

```
MustNewClient() → NewClient → gnet.NewClient → gcli.Start → c.dial()
    ↓ 首次拨号成功
OnOpen → newSession → c.sess.Store → OnReady（仅首次）
    ↓
收发：OnTraffic → decode → Response 匹配 → dispatch
    ↓ 断连
OnClose → c.sess.CAS(nil) → startReconnect
    ↓ 固定间隔后
c.dial() → OnOpen → 收发恢复
    ↑ 重连 goroutine 退出（one-shot）；下次断开再触发
```

## 自动重连

`common/gnetx/client.go:241-268`

- 固定间隔重连（`ReconnectInterval`）
- 单实例保护：`reconnecting.CompareAndSwap` 防重入
- 重连成功即退，one-shot goroutine
- `Close()` → `close(reconnectCh)` → 立即退出

## OnReady 回调

`common/gnetx/client.go:165-173`

- 仅首次拨号成功触发（`ready.CompareAndSwap`）
- off-loop 调用（避免阻塞 event-loop）
- 重连不重复触发

## DialContext 安全

`NewClient` 和重连都使用 `gnet.Client.DialContext`，在 `OnOpen`（event-loop 线程）完成后才返回，保证 `dial()` 返回时 `c.sess` 已就绪，无 `SetContext` 竞争。

`common/gnetx/client.go:166`

## 常见错误

| 错误 | 说明 |
|------|------|
| on-loop handler 调 `cli.Request` | 阻塞 event-loop |
| 调 `cli.Send` 前不判 `cli.Session() == nil` | 返回 `ErrSessionClosed`（兜底非 panic） |
| 忘 `defer cli.Close()` | 重连 goroutine 泄漏 |
| 对同一远端重复建 Client | 不互扰，大概率是设计错误 |
