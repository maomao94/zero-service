# 机巢 SDK 开发模板

> 基于 `common/djisdk` 提炼的机巢上云 MQTT SDK 开发模式。后续对接其他厂商机巢（如道通、科比特等）时，按此模板创建新 `common/xxx-sdk` 包。

## 一、整体架构

```
调用方（app/svc） → Client（构造 + Config + Options）
                  ├─ handler 上行分发（events/osd/state/status/requests/drc/up）
                  ├－ command 下行发送（services/property/drc/down）
                  └─ reply 应答路由（services_reply/property_set_reply）
                        ↓
                   mqttx.Client（MQTT 连接管理）
                        ↓
                   设备（机巢/无人机/负载）
```

**核心分层**：

| 层 | 包路径 | 职责 |
|----|-------|------|
| 协议模型 | `common/<sdk>/protocol*.go` | 全部消息结构体、序列化 |
| Topic 函数 | `common/<sdk>/topic.go` | MQTT Topic 构造函数 + 通配符 Pattern |
| 方法常量 | `common/<sdk>/method.go` | 下行/上行 method 字符串常量 |
| 业务 Client | `common/<sdk>/client.go` | Client struct、构造、命令发送 |
| Handler 分发 | `common/<sdk>/handler.go` | 上行消息解析 + 强类型分发 + 通用兜底 |
| Option 定义 | `common/<sdk>/option.go` | Config struct、WithXxx 函数、handlers 聚合 |
| 错误处理 | `common/<sdk>/error.go` | 厂商错误类型 + 平台错误码 |
| 会话管理 | `common/<sdk>/xxx.go` | DRC/实时通道生命周期（可选） |
| 应用层 Hook | `app/<service>/internal/hooks/` | 业务 Handler 闭包（DB 落库、推送等） |
| 应用层 Logic | `app/<service>/internal/logic/` | gRPC Handler → Client 命令调用 |

## 二、SDK 包实现步骤

### Step 1: 文件组织

按职责拆分文件，不堆砌单个 `client.go`：

```
common/<sdk>/
├── doc.go              # 包级 godoc，协议方向说明
├── client.go           # Client struct、Config、MustNewClient/NewClient、全部下行命令方法
├── option.go           # handlers 聚合 struct、ClientOption、全部 WithXxx 函数
├── handler.go          # 上行分发：HandleEvents/HandleOsd/...、tryDispatch*、SubscribeAll
├── protocol.go         # 公共消息结构体（请求/应答/事件/遥测）
├── protocol_xxx.go     # 特定通道协议结构体（如 DRC）
├── method.go           # method 字符串常量，按功能模块分组
├── topic.go            # Topic 构造函数和通配符 Pattern
├── error.go            # 厂商错误类型 + 平台错误码
└── error_descriptions.go  # 错误码中文描述（可选）
```

### Step 2: 定义协议结构体（`protocol.go`）

以 djisdk 为模板：

```go
// 下行请求壳
type ServiceRequest struct {
    Tid       string `json:"tid"`
    Bid       string `json:"bid"`
    Timestamp int64  `json:"timestamp"`
    Method    string `json:"method"`
    Data      any    `json:"data"`
}

// 上行应答壳
type ServiceReply struct {
    Tid       string           `json:"tid"`
    Bid       string           `json:"bid"`
    Method    string           `json:"method"`
    Data      ServiceReplyData `json:"data"`
}

type ServiceReplyData struct {
    Result int `json:"result"`
    Output any `json:"output,omitempty"`
}

// 设备主动上报事件壳
type EventMessage struct {
    Tid       string `json:"tid"`
    Bid       string `json:"bid"`
    Method    string `json:"method"`
    Gateway   string `json:"gateway,omitempty"`
    NeedReply int    `json:"need_reply,omitempty"`
    Data      any    `json:"data"`
}

// 遥测消息壳（osd/state 共用）
type TelemetryMessage struct {
    Tid       string `json:"tid"`
    Bid       string `json:"bid"`
    Timestamp int64  `json:"timestamp"`
    Gateway   string `json:"gateway,omitempty"`
    Data      any    `json:"data"`
}
```

**关键约定**：
- `Data` 字段使用 `any` 类型，按 method 在 handler 层强转
- 零值字段使用 `omitempty`，避免设备侧解析异常
- 工厂函数统一填充 `Timestamp`（`time.Now().UnixMilli()`）

### Step 3: 定义 method 常量（`method.go`）

按官方文档的功能模块分组：

```go
// ==================== 航线功能（Wayline） ====================

const (
    MethodFlightTaskPrepare = "flighttask_prepare"
    MethodFlightTaskExecute = "flighttask_execute"
    // ...
)

// ==================== 设备管理（Device） ====================

const (
    MethodUpdateTopo = "update_topo"
)
```

**规则**：
- 每组分节注释标出 `Topic:` 和 `方向:`（up/down）
- 常量值对齐厂商官方 method 字符串
- 上行和下行 method 可放在同一文件，用分隔注释区分

### Step 4: 定义 Topic 函数（`topic.go`）

提供两个变体：精确 Topic（带设备 SN）和通配订阅 Pattern（用 `+`）：

```go
// OsdTopic 返回设备遥测数据上报 Topic。
// 路径格式: thing/product/{device_sn}/osd
// 方向: 设备 → 云平台
func OsdTopic(deviceSn string) string {
    return fmt.Sprintf("thing/product/%s/osd", deviceSn)
}

// OsdTopicPattern 返回 OSD Topic 的通配订阅模式。
func OsdTopicPattern() string {
    return "thing/product/+/osd"
}
```

**每个 Topic 函数注释应包含**：
1. 路径格式
2. 数据方向（设备→云 or 云→设备）
3. 用途说明

### Step 5: 定义 Config + Options（`option.go`）

```go
// Config SDK 级配置。
type Config struct {
    MqttConfig mqttx.MqttConfig
    PendingTTL time.Duration `json:",default=30s"`
    Reply      ReplyConfig   `json:",optional"`
    Drc        DrcConfig     `json:",optional"` // 零值禁用
}

// ReplyConfig 控制各类 _reply 的全局开关。
type ReplyConfig struct {
    EnableEventReply   bool `json:",default=true"`
    EnableStatusReply  bool `json:",default=true"`
    EnableRequestReply bool `json:",default=true"`
}

// ---------- Handler 聚合 ----------

// handlers 聚合全部上行 handler 回调，Client 和 clientOptions 均按值持有。
// 新增 handler 改 2 处：加字段 + WithXxx option 函数。
type handlers struct {
    onOsd    func(ctx context.Context, deviceSn string, data *OsdMessage) error
    onEvent  func(ctx context.Context, gatewaySn string, data *EventMessage) error
    // ... 更多 handler
}

// ---------- Option 模式 ----------

type ClientOption func(*clientOptions)

type clientOptions struct {
    handlers    handlers
    pendingTTL  time.Duration
    reply       ReplyConfig
    // 会话管理配置
    drcConfig     DrcConfig
    drcManagerOpts []drcManagerOption
}

func WithOsdHandler(h func(...) error) ClientOption {
    return func(o *clientOptions) { o.handlers.onOsd = h }
}
```

**关键原则**：
1. `ClientOption` 写入 `clientOptions` 而非 `Client`（coding-standards.md 约定）
2. handlers 按值聚合在未导出 struct，`buildClient` 一行赋值
3. 配置类 option（`WithPendingTTL`、`WithReplyConfig` 等）通过 `Config` struct 收束

### Step 6: 实现 Client 构造（`client.go`）

```go
type Client struct {
    mqttClient   mqttx.Client
    handlers     handlers
    pendingTTL   time.Duration
    reply        ReplyConfig
    drcManager   *drcManager   // nil 表示未启用
}

// MustNewClient go-zero 风格构造：Config 携带连接与 SDK 配置，opts 仅注册 handler。
func MustNewClient(cfg Config, opts ...ClientOption) *Client {
    opt := defaultClientOptions()
    if cfg.PendingTTL > 0 {
        opt.pendingTTL = cfg.PendingTTL
    }
    opt.reply = cfg.Reply
    opt.drcConfig = cfg.Drc
    for _, o := range opts {
        if o != nil { o(&opt) }
    }
    return buildClient(mqttx.MustNewClient(cfg.MqttConfig, replyRouters(opt.pendingTTL)...), &opt)
}

// NewClient 复用已有 mqttx.Client。
func NewClient(mqttClient mqttx.Client, opts ...ClientOption) *Client {
    opt := applyOptions(opts...)
    return buildClient(mqttClient, &opt)
}

func buildClient(mqttClient mqttx.Client, opt *clientOptions) *Client {
    c := &Client{
        mqttClient: mqttClient,
        handlers:   opt.handlers,  // 一行映射
        pendingTTL: opt.pendingTTL,
        reply:      opt.reply,
    }
    if opt.drcConfig.HeartbeatInterval > 0 {
        c.drcManager = newDrcManager(c, opt.drcConfig, opt.drcManagerOpts...)
    }
    return c
}
```

### Step 7: 实现命令发送

#### 阻塞模式（等 _reply）

```go
func (c *Client) SendCommand(ctx context.Context, gatewaySn, method string, data any) (string, error) {
    // 1. onlineChecker 在线预检
    if c.handlers.onlineChecker != nil && !c.handlers.onlineChecker(gatewaySn) {
        return "", fmt.Errorf("[sdk] device offline: sn=%s", gatewaySn)
    }
    // 2. UUID tid/bid
    tid := uuid.New().String()
    bid := uuid.New().String()
    // 3. 序列化 + Publish
    req := NewServiceRequest(tid, bid, method, data)
    payload, _ := json.Marshal(req)
    // 4. mqttx.RequestReply 阻塞等 _reply
    reply, err := mqttx.RequestReply[*ServiceReply](ctx, c.mqttClient,
        ServicesReplyTopicPattern(), tid,
        func() error { return c.mqttClient.Publish(ctx, ServicesTopic(gatewaySn), payload) },
        c.pendingTTL)
    if err != nil {
        return tid, fmt.Errorf("[sdk] command failed: method=%s err=%w", method, err)
    }
    if reply.Data.Result != 0 {
        return tid, NewVendorError(reply.Data.Result)
    }
    return tid, nil
}
```

#### 即发即忘（fire-and-forget）

```go
func (c *Client) SendCommandFireAndForget(ctx context.Context, gatewaySn, method string, data any) (string, error) {
    // ... 同上但不等待 reply
    return tid, c.mqttClient.Publish(ctx, ServicesTopic(gatewaySn), payload)
}
```

**业务方法封装**（薄封装，1-3 行）：

```go
func (c *Client) FlightTaskPrepare(ctx context.Context, gatewaySn string, data *FlightTaskPrepareData) (string, error) {
    return c.SendCommand(ctx, gatewaySn, MethodFlightTaskPrepare, data)
}
```

### Step 8: 实现上行分发（`handler.go`）

#### 通用事件分发

```go
func (c *Client) HandleEvents(ctx context.Context, payload []byte, topic string, _ string) error {
    var event EventMessage
    if err := json.Unmarshal(payload, &event); err != nil {
        return err
    }
    // 1. 强类型预分发（已知 method → OnXxx handler）
    handled, result := c.tryDispatchEventNotify(ctx, event.Gateway, event.Method, payload)
    // 2. 兜底（OnEvent handler）
    if !handled && c.handlers.onEvent != nil {
        err := c.handlers.onEvent(ctx, event.Gateway, &event)
        result = ResultFromError(err)
    }
    // 3. need_reply=1 时回复 events_reply
    if event.NeedReply == 1 && c.reply.EnableEventReply {
        return c.eventReply(ctx, event.Gateway, event.Tid, event.Bid, event.Method, result)
    }
    return nil
}
```

#### 强类型预分发模式

```go
func (c *Client) tryDispatchEventNotify(ctx context.Context, gatewaySn, method string, raw []byte) (bool, PlatformResult) {
    switch method {
    case MethodFlightTaskProgress:
        if c.handlers.onFlightTaskProgress != nil {
            var msg struct { Data struct { Output FlightTaskProgressEvent `json:"output"` } `json:"data"` }
            if err := json.Unmarshal(raw, &msg); err != nil {
                return true, PlatformResultHandlerError
            }
            if err := c.handlers.onFlightTaskProgress(ctx, gatewaySn, &msg.Data.Output); err != nil {
                return true, ResultFromError(err)
            }
            return true, PlatformResultOK
        }
    // ... 更多 case
    }
    return false, PlatformResultHandlerError
}
```

#### 订阅注册

```go
func (c *Client) SubscribeAll() error {
    topics := map[string]func(context.Context, []byte, string, string) error{
        EventsTopicPattern():   c.HandleEvents,
        OsdTopicPattern():      c.HandleOsd,
        StateTopicPattern():    c.HandleState,
        StatusTopicPattern():   c.HandleStatus,
        RequestsTopicPattern(): c.HandleRequests,
    }
    for topic, handler := range topics {
        if err := c.mqttClient.AddHandlerFunc(topic, handler); err != nil {
            return fmt.Errorf("[sdk] subscribe %s failed: %w", topic, err)
        }
    }
    return nil
}
```

### Step 9: 定义错误处理（`error.go`）

```go
// PlatformResult reply 包 data.result 取值。
type PlatformResult int

const (
    PlatformResultOK           PlatformResult = 0  // 成功
    PlatformResultHandlerError PlatformResult = 1  // 云侧错误/未注册
    PlatformResultTimeout      PlatformResult = 2  // 超时
)

// VendorError 厂商设备返回的业务错误。
type VendorError struct {
    Code    int
    Name    string
    Message string
}

func (e *VendorError) Error() string {
    return fmt.Sprintf("[sdk] device error: code=%d name=%s message=%s", e.Code, e.Name, e.Message)
}

// PlatformError 携带 reply result 码的业务错误，供 handler 精确控制 _reply.result。
type PlatformError struct {
    Code PlatformResult
    Err  error
}

func ResultFromError(err error) PlatformResult {
    var pe *PlatformError
    if errors.As(err, &pe) { return pe.Code }
    return PlatformResultHandlerError
}
```

### Step 10: DRC/实时通道管理（可选）

如果设备支持实时控制通道（如 DRC）：

```go
type DrcConfig struct {
    HeartbeatInterval time.Duration `json:",default=2s"`
    HeartbeatTimeout  time.Duration `json:",default=300s"`
}

// 两层防护：
// 1. DefaultDrcConfig() 返回公开默认值
// 2. newDrcManager 内部对零值自动填充默认值（防 time.NewTicker(0) panic）

func (c DrcConfig) normalized() DrcConfig {
    if c.HeartbeatInterval <= 0 { c.HeartbeatInterval = 2 * time.Second }
    if c.HeartbeatTimeout <= 0 { c.HeartbeatTimeout = 300 * time.Second }
    return c
}
```

**心跳管理**：内部 goroutine 定时发 heart_beat，设备回传时刷新存活时间。
**序列管理**：公开 `DrcNextSeq(gatewaySn)` 获取递增序号，供杆量控制等场景。

### Step 11: 包文档（`doc.go`）

```go
// Package <sdk> 封装 <厂商> 上云 MQTT 协议能力，提供 Topic 构造、云侧 Client、消息结构与常用负载模型。
//
// 官方资料：<文档链接>
//
// 行为约定：
//   - result 码：0=成功，1=云侧错误，2=超时
//   - events：强类型 method 优先分发到 OnXxx handler，未命中走 OnEvent 兜底
//   - property/set：仅云→设备写可写属性，设备→云 set_reply 回执
//   - 在线预检：WithOnlineChecker 注册后，SendCommand 发前检查
package <sdk>
```

## 三、应用层对接

### 服务上下文集成（`svc/servicecontext.go`）

```go
// 构造 djiClient
djiCli := djisdk.MustNewClient(c.Dji, handlerOpts...)
djiCli.SubscribeAll()
```

### Hook 注册（`app/<service>/internal/hooks/register.go`）

```go
func WithDjiClientOptions(o RegisterDjiClientOptions) []djisdk.ClientOption {
    var opts []djisdk.ClientOption
    if o.DB != nil {
        opts = append(opts, eventHandlerOptions(o.DB)...)
        opts = append(opts, telemetryHandlerOptions(o.DB, o.OnlineCache, o.PushCli)...)
    }
    if o.OnlineCache != nil {
        opts = append(opts, onlineCheckerOption(o.OnlineCache))
    }
    return opts
}
```

**每个 Handler 闭包职责**：
1. 解析 MQTT 消息
2. 写入数据库（Upsert 设备状态、快照、事件）
3. 推送 WebSocket（可选）
4. 刷新在线缓存

### gRPC Logic 层（`app/<service>/internal/logic/`）

```go
func (l *FlightTaskPrepareLogic) FlightTaskPrepare(in *pb.FlightTaskPrepareReq) (*pb.CommonRes, error) {
    tid, err := l.svcCtx.DjiClient.FlightTaskPrepare(l.ctx, in.DeviceSn, data)
    if err != nil {
        return errRes(tid, err), nil
    }
    return okRes(tid), nil
}
```

## 四、测试策略

### SDK 层测试
- `client_test.go`：Config 构造、option 组合、handler 注入
- `protocol*_test.go`：JSON 序列化/反序列化、DrcUnmarshalUpData dispatch

### 应用层测试
- `hooks/register_test.go`：使用 `setupSQLiteDB` 的集成测试，覆盖 handler 闭包的 DB 读写行为
- `hooks/<topic>_up_test.go`：Mock MQTT payload，验证解析、DB 落库、缓存刷新

## 五、代码注释规约

### 文件级注释（`doc.go`）

见 Step 11 包文档模板。

### Topic 函数注释

每个 Topic 函数注释 **必须** 包含三段：`路径格式`、`方向`、`用途`。

```go
// OsdTopic 返回设备遥测数据（OSD）上报 Topic。
// 路径格式: thing/product/{device_sn}/osd
// 方向: 设备 → 云平台
// 用途: 设备定期推送定频遥测数据。
func OsdTopic(deviceSn string) string { ... }

// OsdTopicPattern 返回 OSD Topic 的通配订阅模式。
// 路径格式: thing/product/+/osd
// 方向: 设备 → 云平台（云平台侧订阅）
// 用途: 云平台使用该模式订阅所有设备的遥测数据。
func OsdTopicPattern() string { ... }
```

### Method 常量注释

每个常量 **必须** 包含两行：名称描述 + 方向描述。

```go
const (
    // MethodFlightTaskPrepare 航线任务准备（Flighttask Prepare）
    // 云平台 → 设备（Services），下发航线任务准备指令，设备进行航线预检查
    MethodFlightTaskPrepare = "flighttask_prepare"

    // MethodFlightTaskReady 航线任务就绪通知（Flighttask Ready）
    // 设备 → 云平台（Events），设备通知云平台航线任务已准备就绪可执行
    MethodFlightTaskReady = "flighttask_ready"
)
```

### 分组注释

每个功能模块用 `====` 分隔线标注，必须包含 `参考`（API 文档链接）、`Topic`、`方向`：

```go
// ==================== 航线功能（Wayline） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/.../wayline.html
// Topic: thing/product/{gateway_sn}/services | events
// 方向: services 为云平台 → 设备（下发指令），events 为设备 → 云平台（上报进度）。
```

### Handle 函数注释

统一格式：一行功能描述 + 最多三行参数说明。

```go
// HandleEvents 处理 thing/.../events 上行事件。
//   - ctx: 请求上下文
//   - payload: MQTT 消息原始字节
//   - topic: 消息来源的 MQTT 主题
func (c *Client) HandleEvents(ctx context.Context, payload []byte, topic string, _ string) error {
```

`HandleOsd`、`HandleState`、`HandleStatus`、`HandleRequests`、`HandleDrcUp` 等全部统一此格式。

## 六、SDK 内部错误日志

**原则**：SDK 发命令出错时，SDK 层自己打 `logx.ErrorContextf` 日志，调用方（gRPC Logic）只判断 `err`，不重复打印。

### client.go 实现

所有下发方法（`SendCommand`、`SendCommandFireAndForget`、`publishDrcDown`、属性设置等）在返回 `error` 前 **必须** 调用 `logx.WithContext(ctx).Errorf`：

```go
// SendCommand — 超时/网络错误路径
reply, err := mqttx.RequestReply(...)
if err != nil {
    err = fmt.Errorf("[sdk] command failed: sn=%s method=%s err=%w", gatewaySn, method, err)
    logx.WithContext(ctx).Errorf("%v", err)
    return tid, err
}

// SendCommand — 设备拒绝路径
if reply.Data.Result != 0 {
    err = NewVendorError(reply.Data.Result)
    logx.WithContext(ctx).Errorf("[sdk] command rejected: sn=%s method=%s err=%v", ...)
    return tid, err
}
```

### 调用方（Logic 层）

只需判断 err，不写 `l.Errorf`：

```go
tid, err := l.svcCtx.DjiClient.FlightTaskPrepare(l.ctx, in.DeviceSn, data)
if err != nil {
    return errRes(tid, err), nil // 不打印 Errorf
}
```

## 七、命名规约（四层对齐）

以厂商 **原始 method 字面值** 为唯一锚点，四层命名字面一致可追踪：

| 层 | 规则 | 示例 |
|---|---|---|
| 厂商 method 值 | 原始 snake_case | `flighttask_undo` |
| SDK 常量 | `Method` + CamelCase | `MethodFlightTaskUndo` |
| SDK 客户端方法 | 常量去 `Method` 前缀 | `FlightTaskUndo` |
| Proto RPC | = 客户端方法名 | `FlightTaskUndo` |
| Proto Message | RPC 名 + `Req`/`Res` | `FlightTaskUndoReq` |

**禁止**引入与厂商原始 method 不一致的别名（如 `CancelFlightTask` 替代 `FlightTaskUndo`、`FloatUp` 替代 `UIResourceUpload`）。

协议无关的平台自有接口（`IsDeviceOnline`、`ListDevices` 等）不适用此规则，用业务命名即可。

## 八、对接新机巢 Checklist

- [ ] **Step 1**: 创建 `common/<vendor>-sdk/`，复制模板文件结构
- [ ] **Step 2**: 实现 `protocol.go`（消息结构体）、`topic.go`（Topic 函数）、`method.go`（方法常量）
- [ ] **Step 3**: 实现 `option.go`（Config、handlers、WithXxx）、`client.go`（构造 + 命令方法）
- [ ] **Step 4**: 实现 `handler.go`（上行分发 + SubscribeAll）
- [ ] **Step 5**: 实现 `error.go`（厂商错误码映射）
- [ ] **Step 6**: 按「五、代码注释规约」统一所有注释格式（Topic 三段式、Method 两行式、分组三要素、Handle 统一格式）
- [ ] **Step 7**: 按「六、SDK 内部错误日志」在 client.go 下发方法中加入 `logx.ErrorContextf`，调用方 Logic 删掉冗余 `l.Errorf`
- [ ] **Step 8**: 按「七、命名规约」检查四层命名以厂商 method 值为锚点对齐
- [ ] **Step 9**: `go build ./common/<vendor>-sdk/...` 通过
- [ ] **Step 10**: 创建 `app/<vendor>cloud/`，实现 proto + hooks + logic + server
- [ ] **Step 11**: `go build ./app/<vendor>cloud/...` 通过
- [ ] **Step 12**: 编写 SDK 层 + 应用层单元/集成测试
- [ ] **Step 13**: 创建 spec `<vendor>cloud-*.md` 到 `.trellis/spec/backend/`

## 九、常见陷阱

1. **Data 字段类型**：使用 `any` 而非 `map[string]any`，由 handler 按 method 强转，避免协议升级时字段遗漏
2. **Config 零值语义**：`ReplyConfig` 零值 `{false,false,false}` 全部禁用 _reply；`DrcConfig` 零值不创建 drcManager
3. **handlers 聚合**：新增 handler 改 2 处（struct 加字段 + WithXxx 函数），不要同时改 Client/clientOptions
4. **goroutine 泄漏**：DRC/心跳等后台 goroutine 需要 CancelFunc 清理 + closeOnce 防重
5. **日志前缀不一致**：SDK 层统一用 `[<vendor>-sdk]` 首级前缀，应用层用 `[<vendor>-cloud]`，不要混用或私有化前缀
6. **need_reply 处理**：`need_reply=1` 时必须回复 _reply，`need_reply=0` 时跳过
7. **onlineChecker 非阻塞**：仅做轻量缓存查询，不做网络 IO
8. **Topic 函数注释缺段**：每个 Topic/Pattern 函数必须含 `路径格式`、`方向`、`用途` 三段注释，不可遗漏
9. **Method 常量缺方向**：每个常量必须有第二行注释标明方向（`// 云平台 → 设备（Services）` 或 `// 设备 → 云平台（Events）`）
10. **调用方冗余 Errorf**：SDK 内部已打 `logx.ErrorContextf`，Logic 层调用方不要再写 `l.Errorf`，只判断 err 即可
11. **命名别名**：不得在四层链中引入与厂商原始 method 不一致的别名，必须字面一致可追踪
