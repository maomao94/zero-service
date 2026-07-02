# Server

> **EXPERIMENTAL** — 此包尚未经过生产环境验证。

`Server` 是 gnetx 的 TCP 服务端。实现 `gnet.EventHandler` + `go-zero service.Service`（`Start`/`Stop`），可接入 `service.NewServiceGroup()` 统一管理生命周期。

关键 source：`common/gnetx/server.go:48-396`

## 构造

```go
srv, err := gnetx.NewServer(
    gnetx.WithAddr(":9000"),                          // 必填
    gnetx.WithCodec(myCodec),                         // 必填
    gnetx.WithHandler(myHandler),                     // 必填
    gnetx.WithMaxFrameLength(1 << 20),                // 必填
    gnetx.WithIdleTimeout(5 * time.Minute),           // 可选
    gnetx.WithMulticore(true),                        // 可选
    gnetx.WithSessionListener(myListener),            // 可选
    gnetx.WithOnDecodeError(gnetx.DecodeErrorLogOnly), // 可选
)
```

必填项（`common/gnetx/options.go:157-171`）：`Addr`、`Codec`、`Handler`、`MaxFrameLength`

默认值（`common/gnetx/options.go:174-184`）：
- `SlowHandlerThreshold` → 50ms
- `BatchReadLimit` → 64
- `OnDecodeError` → `DecodeErrorClose`

## 生命周期

```
NewServer() → 配置校验 + SessionManager + stat.Metrics
    ↓
srv.Run()  → gnet.Run(s, addr, opts...)        // 阻塞
    ↓ (gnet 内部事件)
OnBoot     → 存 Engine、启动 idleSweeper
OnOpen     → newSession + SetContext + mgr.add
OnTraffic  → decode → Response 匹配 → dispatch
OnClose    → Session.Close
OnShutdown → 停 idleSweeper
    ↑
srv.Shutdown(ctx) / srv.Stop()                  // 优雅停止
```

两种运行方式：
```go
// A：直接 Run
srv.Run()

// B：接入 go-zero service.Group
sg := service.NewServiceGroup()
sg.Add(srv)
sg.Start()
```

## 线程模型

| 路径 | 线程 | 约束 |
|------|------|------|
| `OnTraffic` → sync handler | event-loop | 必须快（> 50ms 打 slow log） |
| `OnTraffic` → async handler | gnet worker pool | 可做 IO/重操作 |
| `Session.Send` | off-loop | `AsyncWrite` 安全 |
| `Session.Close` | off-loop | `conn.Close` 安全 |

## 半包/粘包处理

`OnTraffic` batch 循环（`common/gnetx/server.go:197-233`）：

```
for i := 0; i < batchLimit; i++ {
    msg, err := Codec.Decode(conn, sess)
    if ErrIncompletePacket → break（等下次可读事件，不 Wake）
    if 其他 error      → handleDecodeError
    consumed++
    // Response auto-route → dispatch
}
// consumed > 0 && InboundBuffered > 0 → Wake 重触发
```

## 空闲扫描

`common/gnetx/idle.go:8-59` — `idleSweeper`

- 独立 goroutine，扫描周期 = `IdleTimeout / 2`（下限 1s）
- 不用 gnet `OnTick`（规避多核 N× 问题且无 per-loop 连接枚举 API）

## Metrics

构造时自动创建 `stat.Metrics`（每分钟输出一次）。

## 常见错误

| 错误 | 说明 |
|------|------|
| on-loop handler 调 `Session.Request` | 阻塞 event-loop |
| `DecodeErrorLogOnly` 下自定义 Codec 不消费坏帧 | 无限循环——内置 Codec 已消费帧字节 |
| OnTraffic 返回 `gnet.Close` 但未清理 Session | 残留泄漏 |
