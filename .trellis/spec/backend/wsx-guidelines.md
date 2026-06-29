# WebSocket 客户端规范

> `common/wsx/` 包是对 gorilla/websocket 的封装，提供带有状态机、自动重连、认证、心跳、token 刷新和 OTel 指标追踪的 WebSocket 客户端。

## When to read

- 创建或修改 WebSocket 客户端连接、配置、消息处理或重连逻辑。
- 使用 `SendJSON` / `Send` 发送消息，或通过 `WithOnMessage` 接收消息。
- 排查连接状态卡死、重连退避不合理、认证超时、心跳中断或并发写入竞态。
- 涉及服务间长连接推送（非 SocketIO、非 MQTT）时优先考虑此包。

## 包结构

```
common/wsx/
├── client.go      # Client 接口 + client 实现：连接、发送、心跳、重连、生命周期
├── config.go      # Config、ConnState（6 态）、ClientOption 函数式选项
├── errors.go      # 哨兵错误：ErrNotConnected, ErrAlreadyRunning, ErrAuthTimeout 等
└── client_test.go # 完整测试套件（状态转换、重连、认证、心跳、并发发送、token 刷新）
```

## 构造方式

```go
// 启动时推荐：panic 快速失败 + 自动注册关闭监听
cli := wsx.MustNewClient(cfg, opts...)

// 测试或延迟连接
cli, err := wsx.NewClient(cfg, opts...)
// ...
_ = cli.Connect(ctx)
```

`MustNewClient` 内部调用 `Connect` 并在失败时 panic，同时注册 `proc.AddShutdownListener`。业务层无需显式关闭。

### ClientOption

```go
cli := wsx.MustNewClient(wsx.Config{
    URL:                 "wss://example.com/ws",
    TokenRefreshInterval: 15 * time.Minute,
    MaxReconnectRetries:  5,
},  wsx.WithHeaders(http.Header{"Authorization": {"Bearer token"}}),
    wsx.WithAuthenticate(func(ctx context.Context) error {
        return cli.SendJSON(ctx, map[string]string{"type": "auth", "token": token})
    }),
    wsx.WithOnMessage(func(ctx context.Context, msg []byte) error {
        return handleMessage(ctx, msg)
    }),
    wsx.WithOnHeartbeat(func(ctx context.Context) ([]byte, error) {
        return []byte(`{"type":"ping"}`), nil
    }),
)
```

`Config` 字段全部有合理的默认值。`normalizeConfig` 在 `NewClient` 中自动调用。

## 状态机

```
Disconnected ──Connect()──→ Connecting ──dial()──→ Connected ──authenticate()──→ Authenticated
    ↑                           │                       │                             │
    │                           │                       │                             │
    └──── reconnect ────────────┘                       │                    (token refresh fail /
     (max retries / cancel)                              │                     connection drop)
           ↑                                             │                             │
           │                               Disconnected ←└── AuthFailed ◄── reconnect ──┘
           └──── StateReconnecting ◄──────────────┘             │
                                                          (reconnect disabled)
```

| 触发 | 源态 | 目标态 | 说明 |
|---|---|---|---|
| `Connect()` | Disconnected | Connecting | 启动 connectionManager goroutine |
| `dial()` 成功 | Connecting | Connected | WS 握手完成，启动 readLoop + heartbeatLoop |
| `authenticate()` 成功 | Connected | Authenticated | 认证完成，启动 token refresh |
| `authenticate()` 失败 | Connected | AuthFailed | 触发认证失败回调；若 `reconnectOnAuthFailed=false` 则断开 |
| 连接断开 | Authenticated | Disconnected | 清理 conn，进入重连判断 |
| `shouldReconnect()` 返回 true | Disconnected | Reconnecting | 退避等待后回到 Connecting |
| `Close()` / 生命周期结束 | 任意 | Disconnected | 清理所有 goroutine |

`State()` 是 O(1) 快照查询，组合 `running`、`lifeCtx`、`authenticated` 和 `conn` 指针：

```go
func (c *client) State() ConnState {
    if !c.running.load() || c.lifeCtx.Err() != nil {
        return StateDisconnected
    }
    if c.authenticated.load() {
        return StateAuthenticated
    }
    // ... check conn pointer
}
```

## 自动重连与指数退避

`connectionManager` 是核心循环。dial 失败或连接中途断开时：

1. `shouldReconnect()` 检查 `running` 状态、`lifeCtx` 取消和 `MaxReconnectRetries` 阈值。
2. `waitBeforeReconnect()` 计算退避延迟并阻塞等待。
3. 退避算法：`backoffDelay(attempt, min, max)` — 以 `MinReconnectDelay` 为基数，每次翻倍，上限 `MaxReconnectDelay`，最终叠加随机 jitter。

```go
func backoffDelay(attempt int, min, max time.Duration) time.Duration {
    base := min
    for i := 0; i < attempt; i++ {
        base *= 2
        if base > max {
            base = max
            break
        }
    }
    return time.Duration(rand.Int64N(int64(base))) // full jitter
}
```

- `MaxReconnectRetries = 0`（默认）表示无限重连，直到 `Close()` 或 `lifeCtx` 取消。
- 重连期间 `State()` 返回 `StateReconnecting`，`onStateChange` 回调对应的状态。

## 认证与 token 刷新

### 认证（`WithAuthenticate`）

Dial 成功后调用，context 携带 `AuthTimeout` 超时：

```go
wsx.WithAuthenticate(func(ctx context.Context) error {
    return cli.SendJSON(ctx, authPayload)
})
```

超时、取消或返回错误分别触发 `StateAuthFailed` 回调 + 不同的哨兵错误日志。`reconnectOnAuthFailed` 控制是否重连（默认 true）。

### Token 刷新（`WithOnTokenRefresh`）

认证通过后启动独立 goroutine，按 `TokenRefreshInterval` 周期执行。失败时：

```go
if c.opts.reconnectOnTokenExpire {
    c.cancelConn()  // 触发连接断开 → 重连 → 重新认证
}
```

`reconnectOnTokenExpire` 默认为 `true`，因此 token 过期会触发完整重连+重新认证周期。

## 心跳机制

`heartbeatLoop` goroutine 在 `setConnection` 时启动，与 `readLoop` 并存。支持两种模式：

1. **自定义心跳**（`WithOnHeartbeat`）：按间隔发送 `TextMessage` 帧，payload 由回调决定。
2. **默认 WS Ping**：发送 `websocket.PingMessage` 帧，服务端返回 Pong 帧保持连接。

```go
// 自定义心跳
wsx.WithOnHeartbeat(func(ctx context.Context) ([]byte, error) {
    return json.Marshal(map[string]string{"type": "ping", "ts": fmt.Sprint(time.Now().Unix())})
})

// 默认 Ping（不传 WithOnHeartbeat）
```

心跳仅在 `authenticated` 状态发送。写操作持有 `writeMu` 锁，与 `Send` / `SendJSON` 互斥。

readLoop 在 `SetPongHandler` 中重置 `ReadDeadline`，确保 Pong 响应不会触发读超时。

## 并发安全：双互斥锁模式

| 锁 | 保护范围 | 持有者 |
|---|---|---|
| `mu` | conn 指针、connCancel、reconnectIdx、running 标志 | `Connect`, `Close`, `write`, `connectionManager`, `cancelConn`, `cleanupConnection` |
| `writeMu` | 所有 WriteMessage 调用（Send、SendJSON、心跳、关闭帧） | `write`, `heartbeatLoop`, `Close` |

```go
// write 方法 — 双锁顺序
func (c *client) write(ctx context.Context, msgType int, data []byte) error {
    c.writeMu.Lock()
    defer c.writeMu.Unlock()

    c.mu.Lock()
    conn := c.conn    // 快照指针
    c.mu.Unlock()

    if conn == nil || !c.running.load() {
        return ErrNotConnected
    }
    return conn.WriteMessage(msgType, data)
}
```

`writeMu` 在 `mu` 之前获取，避免与心跳争夺写资源。`Close` 中先释放 `mu` 再获取 `writeMu` 发送关闭帧——这是故意的，因为关闭时 conn 已快照，不再需要 `mu` 保护。

## 常见反模式

### 1. Connect 前调用 Send / SendJSON

```go
cli, _ := wsx.NewClient(cfg)
cli.Send(ctx, msg) // ❌ ErrNotConnected
cli.Connect(ctx)   // ✅ 先 Connect
```

### 2. Close 后继续发送

```go
cli.Close()
cli.Send(ctx, msg) // ❌ ErrNotConnected（write 检查 running + conn）
```

### 3. 不处理状态转换

`onStateChange` 默认是 no-op。业务应至少记录关键转换：

```go
wsx.WithOnStateChange(func(ctx context.Context, s wsx.ConnState, err error) {
    logx.Infof("[wsx] state: %s, err: %v", s, err)
})
```

### 4. 在外层持有锁后调用 Client 方法

`client` 内部有 `mu` 和 `writeMu`。不要在外部调用方的同步原语中调用 `Send` / `Close`，避免意外锁嵌套。

### 5. 忽略 Send 的错误

Send 在 conn 断开时返回 `ErrNotConnected`。不检查会导致消息静默丢失：

```go
// ❌ 错误丢弃
cli.Send(ctx, msg)

// ✅ 检查
if err := cli.Send(ctx, msg); err != nil {
    logx.Errorf("send failed: %v", err)
}
```

## 参考文件

- `common/wsx/client.go` — 核心接口和实现（562 行）
- `common/wsx/config.go` — Config、ClientOption、ConnState 定义
- `common/wsx/errors.go` — 10 个哨兵错误
- `common/wsx/client_test.go` — 覆盖所有路径的测试套件
