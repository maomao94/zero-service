# ISP 协议接入指南

> ISP（Inspection Substation Protocol）= 区域型变电站远程智能巡视系统技术规范，`common/isp` 包提供协议编解码和常量定义。

## 何时阅读

- 新增 `ispagent` 业务 handler
- 对接上级巡检系统（ISP 协议服务端）
- 扩展 ISP 协议常量或消息类型
- 开发 ispserver 服务端业务

## 协议概述

传输层 TCP，帧格式：

```
0xEB90(2B BE) + SendSeq(8B LE) + RecvSeq(8B LE) + SessionSource(1B) + XMLLength(4B LE) + XML(UTF-8) + 0xEB90(2B BE)
```

- 0xEB90 大端，SendSeq/RecvSeq/XMLLength 小端
- `messageId = (Type << 16) | Command`
- Command=0 为上报类，Command≠0 为指令类
- XML 根元素可配置 `PatrolHost` / `PatrolDevice`

## 包结构

```
common/isp/
├── config.go         # ClientConfig / ServerConfig + ApplyDefaults()
├── constants.go      # Type/Command/MessageID/状态码常量
├── errors.go         # ISP 协议错误 + 客户端本地运行错误
├── message.go        # Message 结构 + gnetx 接口实现 + NewResponse / NewSuccessResponse / NewItemsResponse / NewErrorResponse
├── client.go         # Client + ClientRouter + ClientOption 构造与生命周期管理
├── server.go         # Server + ServerRouter + rootNameCodec
├── wrapper.go        # IspHandler 类型 + wrapHandler / serverWrap / clientWrap + async 注册函数
├── xml.go            # XML 编解码（BuildXML/ParseXML）+ RootName 校验
├── serializer.go     # gnetx Codec 构造 + ISP Serializer
├── serializer_test.go
├── logging.go        # LogFields / LogInbound / LogOutbound / LogFallback / LogErrorResponse
├── model_types.go    # DevicePointModel / PatrolDeviceModel 结构定义
├── model_writer.go   # WriteDeviceModel / WritePatrolDeviceModel 流式 XML 生成
└── model_writer_test.go
```

## 编解码

使用 `gnetx.LengthPrefixCodec`，带 `leadingBytes=0xEB90`、`trailingBytes=0xEB90`：

```go
codec := isp.NewCodec(rootName, maxFrameLength, debug)
```

- `stripBytes=2`：只剥前导，保留 21B 头给 Serializer
- `lengthOffset=19`、`lengthAdjust=2`：XMLLength 不含尾缀
- `debug=true`：启用 `gnetx.DebugSerializer`（debug 级别输出 hex）

## Message 模型

- `Identifiable` → `MessageID()` 供 Router 路由
- `Correlatable` → `TID()` = SendSeq 供请求-响应匹配
- `Response` → `ResponseTID()` 仅 251-3/251-4 返回 RecvSeq，其余返回 ""（不能直接按 Response 接口丢弃所有消息）
- `SendSeq`/`RecvSeq` 对应协议帧 sendSerialNo/receiveSerialNo
- `RecvSeq` 为 ACK（上次收到的对端 SendSeq），出站时回执

## Item 约定

XML 中 `<Item attr="value"/>` 解析为 `map[string]string`，协议定义明确前保持动态。

## 应答构造

### NewResponse（`message.go`）

基于请求消息构造应答，自动处理 SendCode/ReceiveCode 互换和 RecvSeq 回执：

```go
resp := isp.NewResponse(req, isp.SessionSourceServer, isp.StatusSuccess,
    isp.CommandGenericResponseWithItems, items)
resp.SendSeq = conn.NextSendSeq()
```

| 参数 | 说明 |
|------|------|
| `sessionSource` | 本端会话源（SessionSourceClient / SessionSourceServer） |
| `code` | 应答状态码（200/400/500） |
| `command` | 应答指令（251-3 无 Item 或 251-4 有 Item） |
| `items` | 可选业务数据 |

直接使用 `NewResponse` 的调用方需自行填充 `SendSeq`（`conn.NextSendSeq()`），可按需覆盖 `RootName`。经 `ClientRouter` / `ServerRouter` 注册的 handler 返回响应由基础包装器统一填充 `SendSeq`。

## 日志

`common/isp/logging.go` 提供统一的 ISP 消息日志：

| 函数 | 用途 | 日志级别 |
|------|------|----------|
| `LogFields(msg)` | 返回 ISP 消息标准字段 | — |
| `LogInbound(ctx, msg)` | 入站消息（client→server） | `recv` info |
| `LogOutbound(ctx, msg)` | 出站消息（server→client） | `send` info |
| `LogFallback(ctx, msg)` | 未匹配消息 | `fallback` info |

session 信息（`sessionID`、`local`、`remote`、`clientID`）由 gnetx 框架在 dispatch / Request 时通过 `logx.ContextWithFields` 注入 ctx，日志函数通过 `logx.WithContext(ctx)` 自动携带。日志字段沿用来源字段命名：Go 字段使用 lowerCamel（如 `sendSeq`、`recvSeq`、`sendCode`、`reqCode`），XML 字段保留 XML 原名（如 `SendCode`、`ReceiveCode`、`RootName`）。

## ISP Client/Server 与 Handler 注册

`common/isp` 提供 ISP 基础通信对象和方向明确的业务 handler 注册入口。业务系统只关心协议指令 handler，不直接管理 gnetx client/server、router、async/fallback 包装、SendSeq/RecvSeq 或 rootName codec：

```go
type IspHandler func(ctx context.Context, conn gnetx.Conn, req *Message) (*Message, error)
```

### Client API

```go
// 配置与构造
type ClientConfig struct {
    ServerAddr, SendCode, RegisterReceiveCode, RootName string
    HeartbeatInterval, RequestTimeout, ReconnectInterval time.Duration
    MaxFrameLength int
    DebugLog       bool
}
func (c *ClientConfig) ApplyDefaults()

type ClientOption func(*clientOptions)
func WithClientHandler(handler ClientHandler) ClientOption
func WithClientOnRegister(fn func(*Message)) ClientOption

func MustNewClient(cfg ClientConfig, opts ...ClientOption) *Client
func NewClient(cfg ClientConfig, opts ...ClientOption) (*Client, error)

// 生命周期
func (c *Client) Close()
func (c *Client) Context() context.Context

// 请求/应答
func (c *Client) Execute(ctx, typ, command, code, items) (*Message, error)
func (c *Client) requestOnSession(ctx context.Context, sess gnetx.ClientConn, msg *Message) (*Message, error) // 包内方法

// 响应构造（带客户端端点状态覆盖）
func (c *Client) NewItemsResponse(req, items) *Message   // 251-4
func (c *Client) NewSuccessResponse(req) *Message        // 251-3 code=200
func (c *Client) NewErrorResponse(req, err) *Message     // 251-3
func (c *Client) Response(ctx, req, err, items) *Message // 统一响应入口

// 状态查询
func (c *Client) IsRegistered() bool
func (c *Client) Connected() bool
func (c *Client) SendCode() string
func (c *Client) ReceiveCode() string
func (c *Client) RequestTimeout() time.Duration

// 注册成功后，common/isp.Client 自动将配置的 ClientConfig.SendCode
// 绑定到当前 gnetx.ClientConn；绑定失败不会标记为已注册。
// IsRegistered/Connected 仅在当前 Session 仍存在且绑定了 SendCode 时返回 true。
// NewClient 仅在底层 gnetx.Client 初始化成功后启动注册/心跳协程；
// 应用启动路径使用 MustNewClient，成功返回后 Client.transport 在生命周期内保持非空。

// Handler 注册
type ClientHandler func(*ClientRouter)
type ClientRouter
func (r *ClientRouter) Handle(messageID int, fn IspHandler)
func (r *ClientRouter) HandlePairs(pairs []MessageIDPair, fn IspHandler)
```

### Server API

```go
// 配置与构造
type ServerConfig struct {
    ListenAddr, RootName string
    MaxFrameLength, HeartbeatInterval, DeviceRunInterval, NestRunInterval, WeatherInterval, IdleTimeoutSeconds int
    DebugLog bool
}
func (c *ServerConfig) ApplyDefaults()

func NewServer(cfg ServerConfig, register ServerHandler) (*Server, error)

// 生命周期
func (s *Server) Start()
func (s *Server) Stop()
func (s *Server) Manager() *gnetx.SessionManager

// Handler 注册
type ServerHandler func(*ServerRouter)
type ServerRouter
func (r *ServerRouter) Handle(messageID int, fn IspHandler)
func (r *ServerRouter) Fallback(fn IspHandler)
```

### 配置约定

go-zero `internal/config` 应直接使用 `common/isp.ClientConfig` / `ServerConfig` 作为字段类型，不做配置字段转换 helper：

```go
// app/ispagent/internal/config/config.go
type Config struct {
    zrpc.RpcServerConf
    IspSetting isp.ClientConfig  // 直接使用 common/isp 类型
    ...
}

// app/ispserver/internal/config/config.go
type Config struct {
    zrpc.RpcServerConf
    IspConf isp.ServerConfig  // 直接使用 common/isp 类型
}
```

`internal/svc` 只负责依赖注入和传入业务 handler 注册函数：

```go
// ispserver
ispSrv, err := isp.NewServer(c.IspConf, ispserver.RegisterHandlers(c.IspConf))

// ispagent
c.Client = isp.NewClient(cfg,
    isp.WithClientHandler(c.registerHandlers),
    isp.WithClientOnRegister(c.onRegister),
)
```

### 包装器行为

包装器（`wrapper.go`）基于 `Client` / `Server` 统一处理每个业务 handler：
1. 消息类型断言（`*Message`）
2. 入站日志（`LogInbound`）
3. 业务处理
4. 客户端方向在 handler 返回后由 `clientWrap` 记录对端 SendSeq（`Client.trackRecvSeq`）
5. error 自动通过 `ResponseCode(err)` 转为 251-3 通用应答并记录错误日志（`LogErrorResponse`）
6. nil 响应自动转为 251-3 通用成功应答
7. 对最终返回的 `*Message` 统一填充 SendSeq（`conn.NextSendSeq`）

服务端方向默认使用 `SessionSourceServer` 构造应答。客户端方向若需要覆盖 `RootName` / `SendCode` / `ReceiveCode` 端点状态，handler 应通过 `Client.Response` 或 `Client.NewSuccessResponse` / `Client.NewItemsResponse` / `Client.NewErrorResponse` 显式构造响应。

`ispagent` 入站 handler 应返回 `*isp.Message, error`：空/nil 响应由包装器转为 251-3 成功；带业务 Item 的响应由 handler 显式构造 251-4（优先用 `Client.Response` / `Client.NewItemsResponse`）；错误通过 `Client.Response` / `Client.NewErrorResponse` 返回。fallback 由 `Client.connect()` 默认注册（返回 `ErrUnimplemented`），确保 gnetx `any` 返回值始终是协议消息对象。

## 错误边界

### 1. Scope / Trigger

- `common/isp` 构造协议应答、校验客户端调用或向 gnetx 发起请求时。
- 公共协议包不得依赖 gRPC；传输入口负责记录，grpc-go 负责最终序列化。

### 2. Signatures

```go
type IspError struct {
    Code string
    Msg  string
}

var ErrRetry, ErrReject, ErrInternal, ErrUnimplemented *IspError
var ErrInvalidMessageType, ErrClientNotRegistered error
var ErrSessionUnavailable, ErrRequestFailed, ErrUnexpectedResponse error

func NewIspError(code, msg string) *IspError
func ResponseCode(err error) string
func IsUnimplemented(err error) bool
```

### 3. Contracts

- `IspError` 表示写入 251-3 通用应答 `Code` 的协议错误；`ResponseCode` 使用 `errors.As` 提取。
- 客户端本地运行错误是 `errors.go` 中的哨兵错误，并通过 `%w` 保留底层 cause。
- `common/isp` 和 `common/gnetx` 禁止导入 `google.golang.org/grpc/codes` 或 `status`。
- `app/ispagent` 的 RPC logic 直接 `return nil, err`。现有 `LoggerInterceptor` 统一记录错误；grpc-go 对普通 error 默认返回 `Unknown`，对 context 取消/超时使用对应状态。
- `ErrUnimplemented` 表示 ISP 协议 `StatusError`，不等同于 gRPC `codes.Unimplemented`。

### 4. Validation & Error Matrix

| 条件 | ISP error |
|------|-----------|
| `typ <= 0` | 包装 `ErrInvalidMessageType` |
| ISP 客户端尚未注册 | `ErrClientNotRegistered` |
| TCP session 不可用 | `ErrSessionUnavailable` |
| gnetx request 失败/超时 | `ErrRequestFailed`，同时保留底层 cause |
| 响应不是 `*Message` | 包装 `ErrUnexpectedResponse` |
| handler 返回 `IspError` | `ResponseCode` 使用该错误的协议 Code |
| handler 返回普通 error | `ResponseCode` 返回 `StatusError` |

### 5. Good/Base/Bad Cases

- Good：`fmt.Errorf("%w: %w", ErrRequestFailed, cause)`，调用方可同时匹配语义错误和底层错误。
- Base：`ExecuteCommand` 原样返回 `Client.Execute` 的 error，由 RPC 拦截器记录。
- Bad：在 `common/isp/client.go` 中返回 `status.Error(codes.Unavailable, ...)`，或在每个 logic 中重复映射 gRPC code。

### 6. Tests Required

- `Execute` 参数非法、未注册、session 不可用分别断言对应 `errors.Is`。
- `ResponseCode` 对 nil、包装后的 `IspError`、普通 error 断言 200/指定 Code/500。
- 请求失败包装同时断言 `errors.Is(err, ErrRequestFailed)` 和 `errors.Is(err, cause)`。
- 搜索断言 `common/isp`、`common/gnetx` 不包含 gRPC status/codes 导入。

### 7. Wrong vs Correct

```go
// Wrong: 公共协议包绑定 gRPC 传输语义。
return nil, status.Error(codes.Unavailable, "isp tcp session unavailable")

// Correct: 返回包内语义错误，由入口和框架统一处理。
return nil, ErrSessionUnavailable
```

## RootName 校验

`common/isp/xml.go` 提供服务端 RootName 校验：

```go
func ValidateRootName(expected, actual string) error
func IsValidRootName(root string) bool
var ErrRootNameMismatch
```

服务端应包装 Codec 在 decode 后校验 RootName，不一致时返回错误断开连接：

```go
type rootNameCodec struct {
    inner    gnetx.Codec
    rootName string
}
func (c *rootNameCodec) Decode(gc gnet.Conn, conn gnetx.Conn) (any, error) {
    msg, err := c.inner.Decode(gc, conn)
    // ... isp.ValidateRootName(c.rootName, m.RootName)
}
```

## 注册响应

251-4 响应 Item 中的间隔属性均按秒解析；缺失或非法值保留默认/上一轮有效值，不应导致注册失败。

| 字段 | 用途 |
|------|------|
| `heart_beat_interval` | 只覆盖系统心跳间隔 |
| `patroldevice_run_interval` | 巡视装置运行数据周期上报间隔（→ `ReportCategoryPatrolDeviceRunData`） |
| `nest_run_interval` | 无人机机巢运行数据间隔（→ `ReportCategoryDroneNestRunData`） |
| `weather_interval` | 微气象数据间隔（→ `ReportCategoryEnvData`） |

心跳、上行周期上报、下游缓存新鲜度是独立概念，禁止用心跳超时直接替代上报间隔或缓存过期判断。

### Scenario: ISP 客户端注册状态原子发布

#### 1. Scope / Trigger

- 修改 `common/isp.Client` 的注册、重连、注册状态查询或 `Execute` 会话选择时适用。
- 这是 `common/isp` 客户端状态与 `gnetx.ClientConn` 身份绑定之间的跨包并发契约。

#### 2. Signatures

```go
func (c *Client) doRegister(sess gnetx.ClientConn)
func (c *Client) requestOnSession(ctx context.Context, sess gnetx.ClientConn, msg *Message) (*Message, error)
func (c *Client) Execute(ctx context.Context, typ, command int32, code string, items []Item) (*Message, error)
func (c *Client) IsRegistered() bool

type ClientConn interface {
    Conn
    ClientID() string
    BindClientID(clientID string) error
    Request(ctx context.Context, msg Correlatable, ttl time.Duration) (any, error)
}
```

#### 3. Contracts

- 注册请求开始前固定 `sess`，请求、失败关闭和成功绑定始终操作这个 Session，禁止重新获取 Session 后误操作重连产生的新连接。
- 网络请求期间不持有 `Client.mu`。注册响应成功后，必须在同一次 `Client.mu.Lock` 临界区内依次完成：确认 `transport.Session().SessionID()` 仍等于固定的 SessionID、调用 `sess.BindClientID(cfg.SendCode)`、按响应更新 `receiveCode`（SendCode 非空才覆盖）、`heartbeat` 和 `lastHeartbeat`。
- `Execute` 的 `Client.mu.RLock` 必须同时覆盖当前 Session/ClientID 注册校验和 `receiveCode` 快照；释放读锁后再执行网络请求。
- `IsRegistered` 使用同一把 `Client.mu.RLock` 覆盖当前 Session 查询与 ClientID 校验。这样 ISP Client API 不会观察到 ClientID 已绑定、端点状态尚未提交的中间状态。
- `onRegister` 回调和日志在释放 `Client.mu` 后执行，回调读取到的必须是已提交状态。
- 本场景不增加额外注册标志、SessionID 副本或新锁；注册事实由当前 Session 的 `ClientID() == cfg.SendCode` 表示。
- 无新增请求字段、响应字段或环境变量。注册响应沿用 `Message.SendCode` 和 `heart_beat_interval`。

#### 4. Validation & Error Matrix

| 条件 | 行为 |
|------|------|
| 注册请求失败 | 关闭固定的 `sess`，不提交注册状态 |
| `resp.Code != StatusSuccess` | 关闭固定的 `sess`，不提交注册状态 |
| 当前 Session 为空或 SessionID 已切换 | 丢弃旧响应，不绑定、不提交，也不关闭新的当前 Session |
| `BindClientID` 返回错误 | 先释放 `Client.mu`，再关闭固定的 `sess`；不提交端点状态 |
| `Execute` 查询时 Session 不存在或 ClientID 不匹配 | 返回 `ErrClientNotRegistered` |
| 注册提交期间并发调用 `Execute` / `IsRegistered` | 读操作等待写锁释放，只能看到提交前或提交后的完整状态 |

#### 5. Good/Base/Bad Cases

- Good：成功响应后，在一把 `Client.mu` 写锁内校验相同 Session、绑定 ClientID 并提交全部注册状态。
- Base：未注册或重连中的客户端，`IsRegistered` 返回 false，`Execute` 返回 `ErrClientNotRegistered`。
- Bad：先调用 `BindClientID`，之后才获取 `Client.mu` 更新 `receiveCode`；并发调用方可能将客户端判定为已注册，却使用旧的端点编码发送请求。

#### 6. Tests Required

- `TestClientRegistrationPublishesStateUnderClientLock`：测试持有 `Client.mu.RLock` 并释放服务端注册响应时，断言 `sess.ClientID()` 不得提前可见；释放读锁后再断言 `IsRegistered()` 为 true 且 `ReceiveCode()` 已更新。
- `TestClientRegistrationBindsClientID`：成功注册后断言当前 Session 绑定配置的 SendCode。
- `TestClientRegistrationFailureDoesNotCloseReplacementSession`：旧 Session 的延迟失败不得关闭重连后的 Session。
- `TestClientRegistrationRequiresBoundCurrentSession`：无当前 Session 时 `IsRegistered` 和 `Connected` 都返回 false。
- 并发相关变更至少运行 `go test -race ./common/isp/... ./common/gnetx/...`。

#### 7. Wrong vs Correct

```go
// Wrong: ClientID 先对外可见，端点状态稍后才提交。
if err := sess.BindClientID(c.cfg.SendCode); err != nil {
    return
}
c.mu.Lock()
c.receiveCode = resp.SendCode
c.mu.Unlock()

// Correct: 绑定身份与注册状态通过同一把 Client 锁整体发布。
c.mu.Lock()
current := c.transport.Session()
if current == nil || current.SessionID() != sess.SessionID() {
    c.mu.Unlock()
    return
}
if err := sess.BindClientID(c.cfg.SendCode); err != nil {
    c.mu.Unlock()
    _ = sess.Close()
    return
}
if resp.SendCode != "" {
    c.receiveCode = resp.SendCode
}
c.heartbeat = hb
c.lastHeartbeat = time.Now()
c.mu.Unlock()
```

## 应答约定

所有 service→client 消息需回复 251-3 通用应答（Code 区分 `100/200/400/500`）。

## 常见错误

| 错误 | 说明 |
|------|------|
| Command=0 时输出 `<Command>0</Command>` | 应省略（`xmlMessage.Command` 使用 `omitempty`） |
| SendSeq/RecvSeq 字节序用混 | SendSeq/RecvSeq/XMLLength 一律小端 |
| 混淆 Type 共用 | Type=1 既是巡视设备状态上报也是机器人本体指令，由 Command 区分 |
| 251-3/251-4 未匹配时回 251-3 | gnetx 框架已处理：`ResponseTID() != ""` 未匹配直接丢弃 |
| 忘记设置 `resp.SendSeq` | wrapper 统一在返回前调用 `conn.NextSendSeq()` 填充；仅直接调用 `NewResponse` 不经过 wrapper 的场景（如 register handler 显式构造 251-4）需自行填充 |
| 业务 handler 直接使用 `gnetx.NewServer`/`gnetx.NewClient` | 应使用 `isp.NewServer` / `isp.NewClient`，让 common/isp 管理 codec、rootName 校验、router 和 async 包装 |
| config 中定义 `type IspSetting = isp.ClientConfig` 别名 | 直接使用 `isp.ClientConfig` / `isp.ServerConfig` 字段类型 |

## ispserver 服务

`app/ispserver/` 是基于 ISP 标准协议的 TCP 服务端，对标 Java `SipEndpoint`。用于上级巡检系统接受下级设备的注册、心跳和数据上报。

### 目录结构

```
app/ispserver/
├── ispserver.go                # main 入口：zrpc + isp.Server → serviceGroup
├── ispserver.proto             # gRPC 管理接口（ListSessions/DisconnectSession）
├── etc/ispserver.yaml           # 配置：TCP 端口、心跳间隔、RootName 校验
├── internal/
│   ├── config/config.go         # Config + isp.ServerConfig
│   ├── svc/servicecontext.go    # ServiceContext：持有 Config + *isp.Server
│   ├── server/ispserverserver.go # gRPC Server 实现
│   ├── logic/                    # gRPC 业务逻辑
│   ├── ispserver/
│   │   └── router.go            # RegisterHandlers(conf) → isp.ServerHandler
│   └── handler/
│       ├── register.go          # 251-1 注册（唯一真正实现的 handler）
│       ├── heartbeat.go         # 251-2 心跳（async，返回 251-3 code=200）
│       ├── unimplemented.go     # HandleUnimplemented + HandleFallbackUnimplemented
│       └── names.go             # 工具函数
```

### Handler 方向

**仅注册 Client→Server 上行消息：**

| 方向 | 消息 | 处理器 |
|------|------|--------|
| Client→Server | 251-1 注册 | register.go |
| Client→Server | 251-2 心跳 | heartbeat.go |
| Client→Server | 251-3/251-4 | gnetx OnTraffic 通过 Response 接口匹配在途请求，未匹配静默丢弃 |
| Client→Server | 上报类（1-0~5-0, 11-0, 21-0, 41-0, 61-0~64-0, 67-0, 81-0, 10004-0, 20001-0） | unimplemented.go |
| Fallback | 未匹配 | HandleFallbackUnimplemented |

**下行指令（Server→Client）不在 router 注册**，由未来 `SendCommand` 方法主动下发。

### 构造约定

```go
// app/ispserver/internal/ispserver/router.go
func RegisterHandlers(conf isp.ServerConfig) isp.ServerHandler {
    return func(r *isp.ServerRouter) {
        r.Handle(isp.MessageIDRegister, handler.NewRegisterHandler(conf))
        r.Handle(isp.MessageIDHeartbeat, handler.HandleHeartbeat)
        // ... 其余 Handle / HandleFallback 注册
    }
}

// app/ispserver/internal/svc/servicecontext.go
ispSrv, err := isp.NewServer(c.IspConf, ispserver.RegisterHandlers(c.IspConf))
```

Server 构造由 `isp.NewServer` 内部完成：创建 ISP codec、包装 rootNameCodec 校验 RootName、初始化 gnetx router、向 gnetx server 注入 handler。业务服务不再直接构造 gnetx server。

### 与 ispagent 的关系

| | ispagent | ispserver |
|---|---|---|
| 角色 | ISP 协议客户端（代理） | ISP 协议服务端 |
| gnetx | Client（长连接） | Server（多连接） |
| gRPC | 上报/指令透传接口 | 管理接口（ListSessions/DisconnectSession） |
| Handler 方向 | Server→Client（接收指令） | Client→Server（接收上报） |
| 生命周期 | `serviceGroup.Add(rpc)`, `proc.AddShutdownListener(func() { client.Close() })` 注册 Client 关闭 | `serviceGroup.Add(rpc)`, `serviceGroup.Add(tcpServer)` |

## ispagent 服务约定

### 巡视装置上报缓存

`SendPatrolDeviceRunData`、`SendPatrolDeviceStatusData`、`SendPatrolDeviceCoordinates`、`SendDroneNestRunData`、`SendEnvData` gRPC 调用只写入本地内存缓存并返回本地受理成功，不同步等待上级 ISP 应答。

**当前已注册的 ReportCategory（`app/ispagent/internal/ispclient/reporting.go`）：**

| 常量 | messageId | 对应 gRPC |
|------|-----------|-----------|
| `ReportCategoryPatrolDeviceRunData` | 2-0 | `SendPatrolDeviceRunData` |
| `ReportCategoryPatrolDeviceStatusData` | 1-0 | `SendPatrolDeviceStatusData` |
| `ReportCategoryPatrolDeviceCoordinates` | 3-0 | `SendPatrolDeviceCoordinates` |
| `ReportCategoryDroneNestRunData` | 10004-0 | `SendDroneNestRunData` |
| `ReportCategoryEnvData` | 21-0 | `SendEnvData` |

- 周期上报由 `app/ispagent/internal/ispclient.IspClient` 在注册成功后按各上报类别自己的间隔发送 `CommandReport`；禁止把心跳间隔当作业务上报间隔。
- `patroldevice_run_interval` 只驱动巡视装置运行数据；`nest_run_interval` 驱动机巢运行数据；`weather_interval` 驱动环境数据。未在注册响应中给出的字段保持当前间隔不变。
- 默认间隔：坐标 2 秒（`noFreshCheck=true`），其余类别 1 分钟。可通过 `newReportManager` 选项覆盖，也可运行时通过 `SetInterval` 覆盖。
- 缓存模型必须按 report category 复用：category 映射 Type/Command，缓存 code/items/update time/expired/last sent。
- 缓存 key 由 `keyAttrsByCategory[category]` 定义：运行/状态/环境使用 `patroldevice_code + type`，坐标使用 `patroldevice_code`，机巢使用 `nest_code + type`。
- 上报时同一 XML `Code` 下聚合当前 category 的所有最新 Item。
- 发送前必须 snapshot 缓存，禁止持锁执行 TCP 请求。
- 定时上报可用 `threading.NewTaskRunner(4)` 做有界并发，但 `reportTick` 必须等待本轮 `dueReports` 快照全部发送完成后再返回；禁止 fire-and-forget 投递，否则上一轮未 `markSent` 前下一轮 tick 会再次取到同一批快照并重复上报。
- 下游长时间未刷新时按 `freshnessTimeout(report interval)` 判定 expired；非 `noFreshCheck` 类别在 2s tick 扫描时收集过期 key，释放读锁后短写锁删除，删除前必须按 `updatedAt` 二次校验，避免误删并发刷新。
- `noFreshCheck` 类别不做新鲜度清理，继续按类别间隔上报缓存旧值；当前巡视设备坐标属于此类。
- 新 `itemKey` 写入缓存时必须将对应 `category+Code` 的 `lastSent` 置零，使下一次 2s tick 立即上报完整快照；已存在 key 的刷新不能重置 `lastSent`，避免破坏上报间隔控频。
- 过期清理独立于上报间隔：即使 `lastSent` 尚未到期，也要在 tick 扫描时清理非 `noFreshCheck` 过期 item；清理后空的 `category+Code` 缓存槽应删除。
- 新增机巢、环境、微气象等上报类型时复用同一套 category + cache + interval + freshness 生命周期。

#### 巡视上报缓存测试契约

- 新 key 写入后，即使距离上次发送不足 interval，`dueReports` 也应返回该 `category+Code` 的快照。
- 已存在 key 刷新后，`dueReports` 在原 interval 未到期前应继续返回空。
- 非 `noFreshCheck` 过期 item 不应出现在 snapshot 中，并应从 `itemByKey` 清理。
- 同一 `category+Code` 下全部 item 过期后，应删除对应 `cachedReport`。
- `noFreshCheck` 类别即使超过 freshness timeout，也不应删除缓存 item。

#### 构造选项

`newReportManager(opts ...ReportManagerOption)` 支持按类别自定义初始间隔，零值使用默认：

```go
reports: newReportManager(
    ispclient.WithRunDataInterval(10 * time.Second),
    ispclient.WithCoordInterval(5 * time.Second),
    ispclient.WithNestRunInterval(30 * time.Second),
    ispclient.WithEnvDataInterval(30 * time.Second),
),
```

#### 运行时接口

`*IspClient` 对外暴露的上报控制接口（`app/ispagent/internal/ispclient/reporting.go`）：

| 方法 | 用途 |
|------|------|
| `CacheReport(ctx, category, code, items)` | gRPC 上报入口，非法 category 返回 error |
| `SetInterval(category, d)` | 运行时覆盖上报间隔，非正值忽略 |
| `SetNoFreshCheck(category, skip)` | 控制是否跳过新鲜度检查 |
| `ReportIntervals()` | 返回所有类别的当前间隔 |
| `CategoryNoFreshCheck(category)` | 查询类别是否跳过新鲜度检查 |

#### 上报链路关键模式

**`dueReports` 两阶段锁**（`reporting.go:233`）：
- RLock 扫描所有 category × code，收集过期 key 并 clone 到期快照
- RUnlock 释放读锁
- 有残留时短暂 Lock 调用 `deleteExpired` 清理（`updatedAt` 二次校验防并发误删）

**`freshItems` 元组返回**（`reporting.go:363`）：
```go
func freshItems(items, code, now, timeout) ([]isp.Item, []expiredReportItem)
```
一次遍历同时返回未过期 item 的 clone 和过期 key 列表。

**`markSent` 快照校验**（`reporting.go:316`）：
```go
func markSent(category, code, sentAt, snapLastSent)
// snapLastSent 是快照时刻的 lastSent，如果被并发 update 重置为零则跳过更新
if !snapLastSent.IsZero() && report.lastSent.IsZero() { return }
```

**周期上报有界并发**（`client.go:135`）：
```go
const reportTaskConcurrency = 4

for _, report := range c.reports.dueReports(now) {
    report := report
    c.reportRunner.Schedule(func() {
        c.sendReport(now, report)
    })
}
c.reportRunner.Wait()
```

`sendReport` 保持原有语义：每条上报使用 `context.WithTimeout(c.Context(), c.RequestTimeout())`，只有 `Execute` 成功后才调用 `markSent`。失败不更新 `lastSent`，由后续 tick 继续重试。

**新鲜度公式**（`reporting.go:344`）：
```go
freshnessTimeout = max(interval * 2, interval + 10s)
```

#### 新增 proto RPC

| RPC | 表 | 说明 |
|-----|-----|------|
| `SendDroneNestRunData` | O.40 | 机巢运行数据（→ `ReportCategoryDroneNestRunData`，10004-0） |
| `SendEnvData` | J.41 | 环境/微气象数据（→ `ReportCategoryEnvData`，21-0） |
| `ListReportIntervals` | — | 返回 `ReportCategoryInfo` 列表（含 category、name、interval、type、command、key_attrs） |

### 客户端生命周期

`app/ispagent/internal/ispclient/client.go` 是 TCP 长连接客户端，**不是 go-zero service**：

- `NewClient(cfg isp.ClientConfig, taskStore, db, uploader, provider, opts ...ClientOption) *IspClient` — 构造即建连、启动轮询 goroutine
- `Close()` — `*isp.Client.Close()` 取消 context + 关闭 gnetx client
- 通过 `proc.AddShutdownListener(func() { client.Close() })` 注册 go-zero shutdown close，**不放入 `serviceGroup`**
- `serviceGroup` 只放 RPC server 和 `crontask.Scheduler`（二者都实现 `Start/Stop`）

**goroutine 拆分**：

`*isp.Client` 内部 `run()` goroutine（2s ticker）负责注册检查 + 心跳。`*IspClient` 在 `NewClient` 末尾启动独立的 `go c.reportLoop()` goroutine（2s ticker）负责上行缓存上报，防止 TCP 超时阻塞注册/心跳。

- `*isp.Client.run()` — 注册检查 + 心跳（2s ticker）
- `*IspClient.reportLoop()` — 上行缓存上报（独立 2s ticker）

**构造选项**（`ClientOption func(*ClientOptions)`，遵循 `coding-standards.md`）：

```go
NewClient(cfg, store, db, uploader, nil,
    WithReportOption(WithNoFreshCheck(categories...)),
    WithReportOption(WithCoordInterval(5*time.Second)),
)
```

**TCP handler 注册**（`registerHandlers`）：

全部入站 handler 使用 `isp.ClientRouter.Handle` / `HandlePairs` 注册，由 common/isp 的 `clientHandleAsync` / `clientFallbackAsync` 隐藏 gnetx 包装细节，gnet worker pool offload 执行，不阻塞 eventloop。

`IspClient` 嵌入 `*isp.Client`，最终响应契约是 `*isp.Message`：业务 handler 返回 `*isp.Message, error`，wrapper 统一处理 nil→251-3 和 error→251-3。带客户端端点状态的响应通过 `c.Response(ctx, req, err, items)` 构造（内部调 `c.NewSuccessResponse` / `c.NewItemsResponse` / `c.NewErrorResponse`）。gnetx handler 的 `any` 返回值始终是 `*isp.Message`。

**注册响应校验**（`tick()` / `doRegister(sess)`）遵循上文“ISP 客户端注册状态原子发布”场景；应用层不得绕过 `common/isp.Client` 自行绑定 ClientID 或维护第二份注册状态。

最后回执状态使用 `ackState{sessionID, recvSeq}` 保存在 `Client.sessionAck` 中，通过 `atomic.Value` CAS 单调递增，不占用 `Client.mu`。Session 切换时由 `tick` 原子重置为新 Session；`trackRecvSeq` 只接受相同 SessionID 的更新，旧 Session 延迟完成的异步 handler 不得覆盖新 Session 的回执状态，重连后的新 Session 也不得携带旧 Session 的回执序号。

### 汉化映射

所有 ISP 协议指令中文名称统一放在 `handler/names.go`，禁止散落在各 handler 文件中：

```go
// names.go
var taskControlName = map[int32]string{...}    // 任务控制指令名
var robotBodyName = map[int32]string{...}      // 机器人本体指令名
var modelTypeName = map[string]string{...}      // 模型类型名
```

`modelSyncCommandName` 引用 `modelTypeName` 避免重复定义。

### 巡视任务持久化

Cron 触发和 `HandleTaskControl` 都通过 `handler.UpsertPatrolTask` 写入 `GormIspPatrolTask`。使用 `FirstOrCreate` + `Assign` 模式，禁止 `clause.OnConflict`。

状态使用 `gormmodel.PatrolTaskStateXxx` 常量，禁止裸字符串 `"1"`/`"2"`。

源文件：
- `app/ispagent/internal/svc/cron_handler.go` — cron 持久化
- `app/ispagent/internal/handler/task.go` — 任务控制持久化

### carbon 时间格式化

`time.Time.Format("2006-01-02 15:04:05")` → `carbon.CreateFromStdTime(t).ToDateTimeString()`
`time.Time.Format("20060102150405")` → `carbon.CreateFromStdTime(t).Format("YmdHis")`
