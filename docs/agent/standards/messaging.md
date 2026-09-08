# 客户端与消息规范

## 适用范围

修改 `common/netx`、`common/wsx`、`common/mqttx`、MQTT 请求应答、上传下载或其他长连接 client 时读取。业务 topic/事件契约还要读取本层对应的领域规范。

## 通用 client 规则

- client option 只构造配置；连接、状态、锁和统计由构造函数初始化。
- 所有 I/O 接收 context 或由连接级 context 管理，遵守已有 deadline；不能用 client 默认超时覆盖更短的调用 deadline。
- 请求 body、响应 body、上传下载和帧大小使用现有上限；流式 reader 的读取错误必须返回，不能变成部分成功。
- 调用方负责检查 transport error、状态码/协议结果和解码错误，这三者不能合并成“有 Response 即成功”。
- 长连接组件明确状态转换、认证、心跳、重连和 Stop/Close 所有权；回调在锁外执行并能观察取消。

依据：`common/netx/client.go`、`common/netx/response.go`、`common/netx/upload.go`、`common/netx/download.go`、`common/wsx/client.go`、`common/wsx/config.go`。

## HTTP `netx`

- 复用 `Request` builder 表达 header、query、JSON、form、raw 或 reader body；同一请求只选择一种 body 语义。
- request header 覆盖 client default header；构造时克隆外部 map/slice，避免调用方后续修改造成数据竞争。
- 解码 helper 在非成功状态、nil body、大小超限或格式错误时返回 error；不要绕过 `Response` 约束直接无限读取。
- 上传使用 streaming pipe 时传播生产端错误；下载文件先遵守大小限制和 context，再写目标路径。

## MQTT 请求应答

- `mqttx.ReplyRouter` 只管理关联与解码，业务协议包拥有 topic、method、payload 和结果语义。
- 请求必须按“生成 TID -> 在响应 topic 注册 -> publish -> await -> 清理”的顺序，发送失败立即移除/拒绝 pending。
- router 的 decoder 只提取协议中稳定的关联 ID；解析业务结果由 typed handler/协议层完成。
- 同一 topic 的 response handler 只注册一次，并与 router 生命周期一致；不要为每个请求重复注册 broker handler。

依据：`common/mqttx/reply_router.go`、`common/mqttx/request_replyer.go` 及测试。

## 场景: MQTT 广播集群（`common/mqttx/broadcast`）

### 1. Scope / Trigger

- 集群实例间「命令广播 + 按实例 ack」的跨节点调用（当前用例：ieccaller IEC104 命令集群广播、oryxserver StreamRelayStop 分布式停止）适用；单一请求-应答优先用 `mqttx.RequestReply` 直连。
- 修改 `app/oryxserver`、`app/ieccaller`、新广播场景时读取；SDK 通用层改造读本场景第 1-7 节。

### 2. Signatures

```go
// 主题（djisdk 风格，Prefix 由 app 提供：oryxserver "oryx/server"、ieccaller "iec"）
func BroadcastTopic(prefix string) string          // {prefix}/broadcast
func BroadcastTopicPattern(prefix string) string   // 同值（具体订阅）
func BroadcastAckTopic(prefix, instanceID string) string // {prefix}/broadcast_reply/{id}
func BroadcastAckTopicPattern(prefix string) string // {prefix}/broadcast_reply/+（全景订阅/监控）

type Executor func(ctx context.Context, method string, payload []byte) ([]byte, error)

type Broadcaster interface {
    Broadcast(ctx context.Context, method string, payload []byte) error      // fire-and-forget
    BroadcastReply(ctx context.Context, method string, payload []byte, timeout time.Duration) ([]byte, error) // 等待 ack
    AddBroadcastHandler() error
    AddExecutor(method string, fn Executor)
}

func NewBroadcaster(c mqttx.Client, instanceID string, opts ...BroadcasterOption) Broadcaster
func WithPrefix(prefix string) BroadcasterOption
func WithReplyTTL(ttl time.Duration) BroadcasterOption
func NewAckReplyRouter(ttl time.Duration, name string) *mqttx.ReplyRouter[*BroadcastAckBody]
func RegisterErrorKind(src error, kind string, dst func(msg string) error)
func NormalizeErrorKind(err error) string
func ErrorFromKind(kind, msg string) error
var ErrSkipAck = errors.New(...) // executor 返回：处理成功但不回 ack
```

### 3. Contracts

- **协议中立**：`BroadcastBody{Tid,AckTopic,Method,Body}` 与 `BroadcastAckBody{Tid,Method,Success,ResponseBody,Error,ErrorKind}` 只承载关联/路由字段；业务数据一律经 opaque `Body`/`ResponseBody`（string）由业务 executor 约定格式（ieccaller protojson、oryxserver taskId 原文/JSON）。**SDK 内禁止出现业务字段/业务常量**（如 taskId、iec_rejected）。
- **发送隔离**：调用方只传 `method + payload []byte`，不得构造 `BroadcastBody`、拼接主题（Tid/AckTopic/Method 由 SDK 填充；`BroadcastReply` 成功只返回业务结果字节，失败按 errorKind 还原领域错误）。
- 防回环：消费侧忽略 `Body.AckTopic == BroadcastAckTopic(prefix, 本实例ID)` 的消息。
- ack 绑定：`mqttx.Client` 创建时必须经 `mqttx.WithReplyRouter(BroadcastAckTopic(prefix, instanceID), NewAckReplyRouter(ttl,".."))` 注册（Runtime 无事后入口）；默认 TTL 10s。
- errorKind：内置 `timeout`(=ErrReplyExpired)、`duplicate`(=ErrDuplicateID)、`unknown` 兜底；业务类别（如 ieccaller `iec_rejected`）由 app 声明常量并 `RegisterErrorKind` 注册双向映射——SDK 不定义业务 kind。
- 骨架行为：未注册 method → 回 ack `Success=false, Error="unknown method", ErrorKind=unknown`；executor 失败 → `NormalizeErrorKind` 归一后回 ack；`ErrSkipAck` → 不回 ack（oryxserver 非 owner 节点语义、ieccaller ClearPointMappingCache）。
- **装配约定（闭环在 NewServiceContext）**：cluster 分支创建 `Broadcaster` 后依次「业务执行器包 `RegisterExecutors(broadcaster)` → `AddBroadcastHandler()`（失败 `logx.Must`）」；业务执行器包（`app/*/mqtt`）构造**只收窄依赖**（`*relay.Manager`、`*client.ClientManager`、`*Store` 等最小集），**不得注入 `*svc.ServiceContext`**——否则形成 `svc→mqtt→svc` 导入环。`main()` 不保留广播接线样板。
- **nacos 元数据约定**：广播相关 app（ieccaller / oryxserver）nacos 注册时写入 `deployMode`、`broadcastTopic`、`broadcastAckTopic`、`broadcastInstanceId`（svc 提供同名公开访问器），消费方据此定位集群广播实例。

### 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| 无 broker / non-cluster | app 侧不创建 Broadcaster（zero MQTT 依赖） |
| BroadcastReply 超时 | `antsx.ErrReplyExpired`（TTL 默认 10s） |
| ack.Success=false | `(nil, ErrorFromKind(ack.ErrorKind, ack.Error))`；未识别 kind → `errors.New(msg)` |
| 无 reply router | `mqttx.ErrNoReplyRouter`（BroadcastReply 前不发布） |
| 未注册 method（消费侧） | 回失败 ack（unknown kind）让调用方快速失败，不等超时 |
| executor 返回 ErrSkipAck | 不回 ack，调用方侧保持超时语义 |

### 5. Good/Base/Bad Cases

- Good：ieccaller `BroadcastReply(ctx, method, protojsonPayload, 10s)` 等待设备执行结果；oryxserver `Broadcast(ctx, method, taskIDBytes)` fire-and-forget。
- Base：注册 `errKindIECRejected` 后，`CommandRejectedError` 经 `NormalizeErrorKind` 归为 `iec_rejected`，远端 `ErrorFromKind` 还原同类型错误。
- Bad：把 taskId 加进 `BroadcastBody.TaskId` 字段（业务字段污染通用层）；把 `iec_rejected` 写成 SDK 常量；调用方手工拼 `ackTopic`。

### 6. Tests Required

- topic_test（`oryx/server/broadcast`、`iec/broadcast_reply/xyz` 具体值 + Pattern 匹配）；dispatcher_test（回环、未注册、成功/失败、ErrSkipAck）；errors_test（kind 双向）；client_test（内部 Tid/AckTopic 生成、成功只回字节、失败按 kind 还原）；并发骨架 `go test -race`。

### 7. Wrong vs Correct

#### Wrong
```go
// 通用层感知业务字段 + 调用方拼协议字段
type BroadcastBody struct { ..., TaskId string `json:"taskId"` }
err := svc.Broadcast(ctx, method, &BroadcastBody{TaskId: id, AckTopic: "iec/broadcast-ack/" + svc.id})
```

#### Correct
```go
// 协议字段 SDK 内部填充；业务数据走 Body；业务 kind 由 app 注册
err := svc.Broadcast(ctx, relay.MethodStreamRelayStop, []byte(taskId))

// 装配：NewServiceContext 内闭环；业务包收窄依赖（svc→mqtt 单向，mqtt 不导入 svc）
mqtt.NewBroadcast(svcCtx.RelayRegistry).RegisterExecutors(svcCtx.Broadcaster)
svcCtx.Broadcaster.AddBroadcastHandler()
```

## WebSocket `wsx`

- 状态只能通过包内状态机变更；外部使用 `WithOnStateChange` 等回调观察，不能直接修改运行态字段。
- 认证、token refresh、heartbeat 和 reconnect 属于连接生命周期；Stop 后不得再次启动后台循环或写入旧连接。
- 重连期间使用连接 session/context 区分新旧循环，避免旧 reader/heartbeat 影响新连接。

## 反模式

- 记录认证 header、完整请求/响应或设备敏感 payload。
- 无限制 `io.ReadAll` 外部响应或下载。
- 在公共 MQTT 包硬编码 DJI、IEC 104 或其他业务 topic。
- 把 publish 成功描述为业务处理成功或 Exactly Once。

## 验证

- HTTP 覆盖 deadline、取消、header 优先级、状态码、大小上限和 streaming error。
- MQTT 覆盖快速响应、发送失败、未知 TID、超时、重复响应和关闭。
- WebSocket 覆盖连接、认证失败、断线重连、Stop 幂等和旧 session 不再回调；并发变更运行 race test。
