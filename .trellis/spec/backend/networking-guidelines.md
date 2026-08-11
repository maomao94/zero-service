# 网络通信规范

## 适用范围

修改 `common/netx`、`common/wsx`、`common/socketiox`、`common/ssex` 或任何服务的 HTTP/WebSocket/Socket.IO/SSE 通信层时读取。

## netx — HTTP 客户端

### 核心抽象

- **`Engine` 接口** — `Do(req *http.Request) (*http.Response, error)`，允许替换底层执行引擎。
- **`DefaultEngine`** — 包装 `*http.Client`。
- **`HTTPCEngine`** — 包装 go-zero `httpc.Service`（支持熔断、中间件）。
- **`Client`** — 主 HTTP 客户端，功能选项模式构建。

依据：`common/netx/client.go`、`common/netx/transport.go`。

### Client 构建

```go
c := netx.NewClient(
    netx.WithEngine(httpcEngine),       // 使用 go-zero httpc.Service
    netx.WithTLSConfig(tlsConfig),       // TLS 配置
    netx.WithHeaders(headers),            // 全局请求头
    netx.WithMaxResponseBytes(10<<20),   // 响应体限制
    netx.WithHTTPClientOption(func(c *http.Client) { ... }), // 额外 HTTP 客户端配置
)
```

- 默认限制: 响应 10MB，上传 32MB。
- `WithHTTPClientOption` 是灵活性兜底，允许直接修改 `*http.Client`。

依据：`common/netx/client.go`。

### Request/Response

- **`Request`** — 链式构建器 + 功能选项模式：
  - 方法: `Get`、`Post`、`Put`、`Delete`、`Patch`、`Head`、`Options`
  - Body 类型: raw bytes、JSON struct、form map、io.Reader
  - Query 参数、Header、FormData 均可配置
- **`Response`** — 永远不会返回 error：
  - 网络错误或超时都捕获在 `Response.Err` 字段中
  - 状态码: 504 (超时)、503 (网络错误)
  - 自动检测 Content-Type 并解码: `JSON(target)`、`XML(target)`、`Text()`
- 包级便捷函数 (`netx.Get()`、`netx.Post()` 等) 使用内部 `defaultClient`

**关键约定**: 调用方检查 `resp.Err` 而非 `err`。`Do()` 返回的 `error` 只表示请求构建失败，不表示网络执行失败。

依据：`common/netx/request.go`、`common/netx/response.go`。

### Transport

- `NewTransport(opts ...TransportOption)` — 创建带默认参数的 `http.Transport`（30s dial, 10s TLS, 90s idle, 100 max idle, HTTP/2 支持）。
- `NewHTTPClient(opts ...TransportOption)` — 使用上述 Transport 创建 `*http.Client`。
- `NewHTTPCService(name, opts ...TransportOption)` — 创建 go-zero `httpc.Service`。
- `NewHTTPEngine(svc httpc.Service)` — 包装 go-zero service 为 Engine。

依据：`common/netx/transport.go`。

### 编码工具

- `ValidateAndFlatten` — JSON → 扁平键值 map
- `EncodeURLEncoded` — JSON → URL-encoded 字符串
- `EncodeURLEncodedIfNeeded` — 智能判断：已编码则不处理
- `EncodeMultipart` — 键值 map → multipart/form-data

依据：`common/netx/encode.go`。

## wsx — WebSocket 客户端

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

## socketiox — Socket.IO 服务端

### Server 架构

- 基于 `github.com/doquangtan/socketio/v4`，封装房间管理、广播、统计上报。
- `Server` 持有 `eventHandlers map[string]EventHandler` 和 `sessions map[string]*Session`（`sync.RWMutex` 保护）。
- 钩子: `tokenValidator`、`tokenValidatorWithClaims`、`connectHook`、`disconnectHook`、`preJoinRoomHook`。
- `contextKeys` 从 JWT claims 提取到 session metadata。

依据：`common/socketiox/server.go`。

### 内置事件

| 事件 | 方向 | 用途 |
|------|------|------|
| `__connection__` | 系统 | 连接建立，加入初始房间 |
| `__disconnect__` | 系统 | 断开，清理 session |
| `__up__` | 上行 | 主上行事件，路由到 `EventUp` handler |
| `__join_room_up__` | 上行 | 加入房间（经过 preJoinRoomHook） |
| `__leave_room_up__` | 上行 | 离开房间 |
| `__rooms_page_up__` | 上行 | 分页获取房间列表 |
| `__room_broadcast_up__` | 上行 | 房间广播 |
| `__global_broadcast_up__` | 上行 | 全局广播 |
| `__stat_down__` | 下行 | 每分钟推送统计（Nps、房间数、metadata） |
| `__down__` | 下行 | 下游事件推送 |

### 响应模式

- **Ack 优先 + ReplyDown 兜底**: 客户端带 Ack 回调用 Ack 响应；无 Ack 时通过 `__down__` 事件下行推送。
- 响应码: `200`（成功）、`400`（参数错误）、`500`（业务错误）。
- `__down__` 是保留事件名，不能用于广播。

依据：`common/socketiox/server.go`、`common/socketiox/handler.go`。

### Session 管理

- `Session` 提供房间操作 (`JoinRoom`、`LeaveRoom`)、多种 Emit 方法 (`EmitAny`、`EmitString`、`EmitDown`、`EmitEventDown`、`ReplyEventDown`)。
- Session 元数据: `GetMetadata(key)`、`AllMetadata()`、`SetMetadata(key, val)`。
- 查询: `GetSession(id)`、`GetSessionByDeviceId()`、`GetSessionByUserId()`、`GetSessionByKey()`。

### 多节点 (SocketContainer)

- `SocketContainer` 管理 gRPC 客户端池 (`map[string]socketgtw.SocketGtwClient`)。
- 支持三种服务发现: Direct endpoint、Etcd discovery、Nacos discovery。
- Nacos: 订阅服务变更 + 60s 轮询兜底。
- Etcd: go-zero `discov.Subscriber` + subset 采样（最多 32 节点）。
- Nacos gRPC 端口从 `metadata["gRPC_port"]` 提取。

依据：`common/socketiox/container.go`。

### 反模式 (socketiox)

- 使用 `__down__` 作为广播事件名（会被拒绝）。
- 绕过 session 直接操作底层 socket 连接。
- 在 `preJoinRoomHook` 中进行耗时操作（阻塞连接流程）。
- 不检查 Ack 就直接使用 ReplyDown 模式（可能重复推送）。

## ssex — SSE 写入器

### Writer API

```go
w, _ := ssex.NewWriter(responseWriter)
w.WriteData("hello")           // data: hello\n\n
w.WriteEvent("custom", "msg")  // event: custom\ndata: msg\n\n
w.WriteJSON(struct{...})       // data: {"...\n\n
w.WriteDone()                  // data: [DONE]\n\n  (OpenAI 兼容)
w.WriteKeepAlive()             // : keepalive\n\n
w.WriteComment("debug info")   // : debug info\n\n
```

- `Writer` 实现 `io.Writer` 接口，支持流式 A2UI 输出。
- 行缓冲: `Write()` 累积字节直到 `\n`，按行发送 `data: {line}\n\n`。
- 线程安全: 所有公开方法使用 `sync.Mutex` 串行化。
- 向后兼容别名: `LineWriter` = `Writer`、`NewLineWriter` = `NewWriter`。
- `ResponseWriter()` 暴露底层 writer。

依据：`common/ssex/writer.go`。

### 反模式 (ssex)

- 不设置 Content-Type header（SSE 需要 `text/event-stream`）。
- 使用 `Write()` 后不调用 `BufferFlush()` 发送缓冲区残余。
- 在 SSE handler 中调用 `WriteDone()` 后继续写入。

## 验证

- netx: 验证 Engine 切换、错误捕获为 Response.Err、超时状态码、编解码完整性。
- wsx: 验证状态机转换、重连、心跳、认证、优雅关闭、并发 Send。
- socketiox: 验证连接/断开/房间/广播/Ack/ReplyDown、多节点 gRPC 转发。
- ssex: 验证行缓冲、全部写方法、线程安全、OpenAI `[DONE]` 格式。
