# Server

`Server` 是 gnetx 的 TCP 服务端。实现 `gnet.EventHandler` + `go-zero service.Service`（`Start`/`Stop`），可接入 `service.NewServiceGroup()` 统一管理生命周期。

关键 source：`common/gnetx/server.go`

## 构造

```go
srv, err := gnetx.NewServer(
    gnetx.WithAddr(":9000"),                          // 必填
    gnetx.WithCodec(myCodec),                         // 必填
    gnetx.WithHandler(myHandler),                     // 必填
    gnetx.WithMaxFrameLength(1<<20),                  // 必填
    gnetx.WithIdleTimeout(5*time.Minute),
    gnetx.WithMulticore(true),
    gnetx.WithSessionListener(myListener),
)
```

必填项（`common/gnetx/options.go:164-178`）：`Addr`、`Codec`、`Handler`、`MaxFrameLength`

默认值（`common/gnetx/options.go:181-191`）：
- `SlowHandlerThreshold` → 50ms
- `BatchReadLimit` → 64
- `OnDecodeError` → `DecodeErrorClose`

## 共享 ReplyPool

Server 在 `NewServer` 中创建全局 `antsx.ReplyPool[any]`，所有 Session 共享（`common/gnetx/server.go:78-79`）：

```go
replyPool := antsx.NewReplyPool[any](
    antsx.WithName("gnetx-server-"+normalizeAddrForMetrics(o.Addr)),
    antsx.WithDefaultTTL(30*time.Second),
)
```

OnOpen 中传入：`newSession(..., s.replyPool)`（`server.go:178`）

Shutdown 中关闭：`defer s.replyPool.Close()`（`server.go:116`，用 defer 保证即使 eng.Stop 报错也清理）

## 生命周期

```
NewServer() → 配置校验 + SessionManager + replyPool
  srv.Run()  → gnet.Run(s, addr, opts...)  // 阻塞
  OnBoot     → 存 Engine、启动 idleSweeper
  OnOpen     → newSession(s.mgr, s.replyPool) + SetContext + mgr.add
  OnTraffic  → decode → Response.resolveResponse(共享池) → dispatch
  OnClose    → Session.Close
  OnShutdown → 停 idleSweeper
  srv.Shutdown(ctx) / srv.Stop() → replyPool.Close() + eng.Stop()
```

两种运行方式：
```go
// A：直接 Run
srv.Run()
// B：接入 go-zero service.Group
sg := service.NewServiceGroup(); sg.Add(srv); sg.Start()
```

## 线程模型

| 路径 | 线程 | 约束 |
|------|------|------|
| `OnTraffic` → sync handler | event-loop | 必须快（> 50ms 打 slow log） |
| `OnTraffic` → async handler | gnet worker pool | 可做 IO/重操作 |
| `Session.Send` | off-loop | `AsyncWrite` |
| `Session.Close` | off-loop | `conn.Close` |

## 半包/粘包处理

`OnTraffic` batch 循环（`common/gnetx/server.go:189-212`）：

```
for i := 0; i < batchLimit; i++ {
    msg, err := Codec.Decode(gconn, sess)
    if ErrIncompletePacket → break
    if 其他 error       → handleDecodeError
    consumed++
    // Response auto-route: resolveResponse(共享池)
    // 未命中 → dispatch
}
// consumed > 0 && InboundBuffered > 0 → Wake 重触发
```

同步 handler 返回 reply 时，Server 使用同一个 handler ctx 调用 `Codec.Encode(ctx, reply, conn)`。该 ctx 已由 dispatch 通过 `PacketContextProvider` 注入入站协议头（key=`PacketContextKey`），因此协议 Codec 可以从 ctx 中读取入站 seq 来填回复的 ack。dispatchAsync 同理（入池前完成注入）。

## 空闲扫描

`common/gnetx/idle.go` — `idleSweeper`

- 独立 goroutine，扫描周期 = `IdleTimeout / 2`（下限 1s）
- 不用 gnet `OnTick`（规避多核 N× 问题且无 per-loop 连接枚举 API）

## 常见错误

| 错误 | 说明 |
|------|------|
| on-loop handler 调 `Session.Request` | 阻塞 event-loop |
| `DecodeErrorLogOnly` 下自定义 Codec 不消费坏帧 | 无限循环 |
| OnTraffic 返回 `gnet.Close` 但未清理 Session | 残留泄漏 |
| Shutdown 中 replyPool 未 defer 关闭 | TimingWheel goroutine 泄漏 |
