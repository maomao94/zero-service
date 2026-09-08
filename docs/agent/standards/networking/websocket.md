# WebSocket Client

修改 `common/wsx` 的连接状态、认证、心跳、重连、发送或关闭流程时读取。

### 状态机

WebSocket 连接生命周期通过 `atomic.Int32` 状态机管理：

```
Disconnected → Connecting → Connected → Authenticated → (ready)
                  ↑              ↓
                  └── AuthFailed ←┘
                  ↑              ↓
                  └── Reconnecting ←┘
```

- `Send()` / `SendJSON()` 只在 `StateAuthenticated` 时才允许发送。
- `conn` 使用 `atomic.Pointer[websocket.Conn]` 无锁读取。

依据：`common/wsx/client.go`、`common/wsx/config.go`。

### 配置与生命周期

```go
cfg := wsx.Config{
    URL: "wss://...",
    DialTimeout: 10 * time.Second,
    AuthTimeout: 5 * time.Second,
    HeartbeatInterval: 30 * time.Second,
    ReconnectInterval: 1 * time.Second,
}
cli := wsx.MustNewClient(cfg,
    wsx.WithOnAuthenticate(func(ctx, conn) error { ... }),
    wsx.WithOnMessage(func(ctx, msgType, data) { ... }),
    wsx.WithOnStateChange(func(old, new wsx.ConnState) { ... }),
)
```

- `MustNewClient` 注册 `proc.AddWrapUpListener` 实现优雅关闭。
- 认证阶段在连接建立后执行，认证失败触发重连。
- Token 刷新定时器 (默认 30min)，刷新失败触发重连。
- 心跳支持自定义回调（文本/JSON）或 WebSocket Ping。
- 写入串行化使用 `sync.Mutex`，非 channel。

### 可观测性

- 每条接收消息创建 OTel trace span。
- `stat.Metrics` 集成吞吐/丢弃统计。
- URL MD5 哈希用于 metrics 命名和会话标识。
- go-zero `logx` 结构化日志，包含 `url` 和 `session` 字段。

依据：`common/wsx/client.go`。

### 反模式 (wsx)

- 认证阶段未完成就调用 `Send()`（会被拒绝）。
- 在回调中长时间阻塞（阻塞消息读取循环）。
- 不使用 `MustNewClient` 或手动注册 shutdown hook（资源泄露）。
- 重连间隔设得太短，形成连接风暴。

验证状态转换、重连、心跳、认证、优雅关闭和并发发送。
