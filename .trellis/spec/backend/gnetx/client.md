# Client

> **EXPERIMENTAL** — 此包尚未经过生产环境验证。

`Client` 是 gnetx 的**单连接 TCP 客户端**，对标 `mqttx`/`modbusx` 的 `MustNewClient` 模型。一个 Client = 一个远端连接。

关键 source：`common/gnetx/client.go`

## 核心设计

- **构造即拨号** — 无 `Start`/`Stop`（仅 `NewClient` + `Close`）
- **固定间隔重连** — 断线按 `ReconnectInterval`（默认 3s）自动重连
- **多连接 = 多 Client**
- **不实现 `go-zero service.Service`** — 生命周期仅构造 + `Close`
- **共享 ReplyPool** — Client 级 `antsx.ReplyPool[any]`，所有重连 session 共享

## 构造

```go
// MustNewClient：失败 panic + 自动注册 proc 关闭监听
cli := gnetx.MustNewClient("127.0.0.1:9000",
    gnetx.WithClientCodec(myCodec),
    gnetx.WithClientHandler(myHandler),       // 统一 Handler 类型
    gnetx.WithClientMaxFrameLength(1 << 20),
    gnetx.WithClientReconnectInterval(5 * time.Second),
    gnetx.WithClientOnConnect(func(ctx context.Context, c *gnetx.Client) {
        log.Println("connected")
    }),
)
defer cli.Close()

// NewClient：首次拨号失败时返回 client 并启动后台重连
cli, err := gnetx.NewClient("127.0.0.1:9000", opts...)
```

必填项（`common/gnetx/options.go:296-307`）：`Codec`、`Handler`、`MaxFrameLength`

默认值（`common/gnetx/options.go:310-323`）：
- `ReconnectInterval` → 3s
- `SlowHandlerThreshold` → 50ms
- `BatchReadLimit` → 64

## ClientConn 接口

```go
type ClientConn interface {
    Conn
    Request(ctx context.Context, msg Correlatable, ttl time.Duration) (any, error)
}
```

`cli.Session()` 返回 `ClientConn`（nil if 未连接）。

## 响应式 API

```go
// fire-and-forget 发送
cli.WriteAsync(ctx, msg)                  // → sess.WriteAsync(ctx, msg)

// 响应式请求：发请求等匹配 tid 的回包
resp, err := cli.Request(ctx, req, ttl)   // → sess.Request(ctx, req, ttl)
```

`common/gnetx/client.go:107-121`

`WriteAsync`/`Request` 在未连接或重连中均返回 `ErrSessionClosed`（非 panic）。

## 生命周期

```
MustNewClient() → NewClient → gnet.NewClient → gcli.Start → c.dial()
    ↓ 首次拨号成功
OnOpen → newSession(UUID) → c.sess.Store → OnConnect（每次连接，含重连）
    ↓
收发：OnTraffic → decode → Response 匹配（resolveResponse）→ dispatch
    ↓ 断连
OnClose → c.sess.CAS(nil) → cn.Close() → startReconnect
    ↓ 固定间隔后
c.dial() → OnOpen → 收发恢复
    ↑ 重连 goroutine 退出（one-shot）；下次断开再触发
```

## 自动重连

`common/gnetx/client.go:206-232`

- 固定间隔重连（`ReconnectInterval`，默认 3s）
- 单实例保护：`reconnecting.CompareAndSwap` 防重入
- 重连成功即退，one-shot goroutine
- `Close()` → `close(reconnectCh)` → 立即退出

## OnConnect 回调

`common/gnetx/client.go`

- 每次连接成功后触发（含自动重连）。
- off-loop 调用（避免阻塞 event-loop）。
- 配置 `ConnectTimeout` 时，回调收到带超时的 ctx。

## ReplyPool（共享）

Client 级的 `antsx.ReplyPool[any]`，构造时创建，所有重连 session 共享。request 使用复合 TID（`sessionID + "|" + msg.TID()`）隔离不同重连期间的同 TID 请求。

## 常见错误

| 错误 | 说明 |
|------|------|
| on-loop handler 调 `cli.Request` | 阻塞 event-loop |
| 调 `cli.WriteAsync` 前不判 `cli.Session() == nil` | 返回 `ErrSessionClosed`（兜底非 panic） |
| 忘 `defer cli.Close()` | 重连 goroutine 泄漏 |
| 对同一远端重复建 Client | 不互扰，大概率是设计错误 |

## 心跳（Heartbeat）

`common/gnetx/client.go:148-165` — 通过 gnet `OnTick` 定时发送应用层心跳；`client.go:339` 在 `buildGnetOptions` 中开启 `gnet.WithTicker(true)`：

```go
cli := gnetx.MustNewClient("tcp", "127.0.0.1:9000",
    gnetx.WithClientCodec(myCodec),
    gnetx.WithClientHandler(myHandler),
    gnetx.WithClientMaxFrameLength(1 << 20),
    gnetx.WithClientHeartbeatInterval(30 * time.Second),
    gnetx.WithClientHeartbeatMessage(func() any {
        return &MyHeartbeatMsg{Type: "ping"}
    }),
)
```

- `HeartbeatInterval` ≤ 0 时不启用心跳
- `HeartbeatMessage` 为 nil 时不发送
- 心跳报文通过 `Codec.Encode` 编码后走 `AsyncWrite` 非阻塞发送
- 心跳没有入站请求上下文，编码使用 `context.Background()`
- 内部自动调用 `gnet.WithTicker(true)` 开启 gnet 定时器
- 未连接时（session nil/closing）静默跳过，按间隔重试
