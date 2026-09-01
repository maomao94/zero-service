# 简化 livekitx Hook 设计并补充场景字段文档 — Design（修订版）

## 架构边界

改造后 `common/livekitx` 分两层：

1. **基础设施**：配置（`New`/`Options`/`Close`）、管理 API（Twirp client + 注入）、Token helper、Webhook 验签 KeyProvider。
2. **Room API + 原生透传**：加入/创建/创建并加入/删除房间；实时连接直接返回 SDK 原生 `*lksdk.Room`，所有实时事件（断开、重连、轨道、Data、聊天、RPC）由业务在原生 `RoomCallback` 中处理。

包内**不再存在**：事件分发机制、typed 事件类型、连接状态存储（Store/ConnectionState）、RealtimeRoom 句柄。

## 保留面（不动或仅微调）

| 文件 | 保留内容 |
|---|---|
| `config.go` | `New`/`Options`/`WithURL`/`WithAPIKey`/`WithHTTPService`/`WithHTTPClient`/`Close`、校验逻辑；**删除 `WithStore` 与 `store` 字段，`Client` 删除 `stateMu`/`realtime` 字段** |
| `api.go` | `API` 五服务（Room/Egress/Ingress/SIP/AgentDispatch）、`authInterceptor`、`requestRoom`、`serviceHTTPClient` |
| `token.go` | `JoinToken`/`SIPToken`（含 `SetCanPublish`/`SetCanSubscribe`/`CanPublishData`） |
| `errors.go` | `ErrClosed`/`ErrInvalidConfig`/`ErrInvalidTokenOptions`；**删除 `HookPanicError`** |

## 删除面

- `hooks.go`、`events.go`、`data.go`、`store.go` 整文件删除。
- `RealtimeRoom`、`Connect`、`ConnectionStatus`/`ConnectionState`、`WithStore`、`ChatMessageEvent` 等 typed 事件、`NewWebhookKeyProvider` 之外的 Webhook 相关 API（`OnWebhook`/`ReceiveWebhook`/`WebhookEvent`）。
- 依赖它们的测试：`bridge_test.go`（桥接/Store 相关）、`hooks_test.go`、`store_test.go`、`example_test.go` 重写。

## 重写面

### room.go（由 realtime.go 改造，文件名改为 room.go，第三轮修订）

```go
// WithRoomCallback 注入默认 RoomCallback；JoinRoom/CreateAndJoinRoom 未显式
// 指定回调时使用该回调。callback 为 SDK 原生类型，原样透传。
func WithRoomCallback(callback *lksdk.RoomCallback) Option

// JoinRoom 使用 SDK ConnectInfo 加入已有房间：APIKey/APISecret 取自 client
// 配置（JoinWithContext 内部生成 join token），业务只传房间名与身份。
// JoinWithContext 成功返回即已连接（SDK 原生无 connected 回调）。
func (c *Client) JoinRoom(ctx context.Context, roomName, identity string, opts ...lksdk.ConnectOption) (*lksdk.Room, error)

// CreateRoom 经管理 API 创建房间；返回协议房间信息。
func (c *Client) CreateRoom(ctx context.Context, name string) (*livekit.Room, error)

// CreateAndJoinRoom 先创建房间，再用 ConnectInfo 加入；
// 加入失败不自动删除房间（可能已有其他参与者），由业务决定清理。
func (c *Client) CreateAndJoinRoom(ctx context.Context, roomName, identity string, opts ...lksdk.ConnectOption) (*lksdk.Room, error)

// DeleteRoom 经管理 API 删除房间。
func (c *Client) DeleteRoom(ctx context.Context, name string) error

// RemoveParticipant 经管理 API 将指定参与者移出房间（踢人）。
func (c *Client) RemoveParticipant(ctx context.Context, roomName, identity string) error

// InviteParticipant 经管理 API 向房间内指定参与者发送一条邀请消息
// （可靠 Data 通道，topic 与 payload 由业务定义）。
func (c *Client) InviteParticipant(ctx context.Context, roomName, identity, topic string, payload []byte) error

// EndRoom 经管理 API 结束会议（EmptyTimeout=0 立即结束）。
func (c *Client) EndRoom(ctx context.Context, name string) error
```

- `Client` 增加 `callback *lksdk.RoomCallback` 字段（WithRoomCallback 注入，nil 允许）。
- `JoinRoom`：校验（ErrClosed/ErrInvalidTokenOptions 语义改为校验空房间名/身份）→ `lksdk.NewRoom(c.callback)` → `room.JoinWithContext(ctx, c.config.URL, lksdk.ConnectInfo{APIKey: c.config.APIKey, APISecret: c.config.APISecret, RoomName: roomName, ParticipantIdentity: identity}, opts...)` → 返回 room。签名以 SDK v2.18.1 源码核对（JoinWithContext 存在且签名如上）。
- `CreateAndJoinRoom`：先 `CreateRoom` 再 `JoinRoom`；失败原样返回，不自动删房间。
- `ChatTopic` 常量**删除**（用户判定 topic 约定交业务）。
- 全部导出 API 简洁中文注释。

### webhook.go

```go
// NewWebhookKeyProvider 返回校验 LiveKit Webhook 签名所需的 KeyProvider；
// 按 signing key 校验任意 key claim（不能使用 NewSimpleKeyProvider("", key)，
// 空 API key 永远验签失败）。
func NewWebhookKeyProvider(signingKey string) auth.KeyProvider
```

业务自行：

```go
event, err := webhook.ReceiveWebhookEvent(req, livekitx.NewWebhookKeyProvider(cfg.WebhookKey))
```

## 测试改造

- 删除：`hooks_test.go`、`events_test.go`、`store_test.go`、`bridge_test.go`（或其中桥接/Store 依赖部分）。
- `example_test.go` 重写为 **Room API + RoomCallback 完整示例**（含 JoinRoom/CreateRoom/CreateAndJoinRoom/DeleteRoom、聊天/Data/RPC 原生用法、Webhook 接收），编译验证，作为未来参考。
- 新增 `room_test.go`：Room API 单元测试（校验分支：ErrClosed/ErrInvalidTokenOptions；CreateRoom/DeleteRoom 走 Twirp mock 断言请求与方法；callback 透传——传入自定义 callback 可收到入会回调，可拆入集成测试）。
- `integration_test.go`：改用原生 `RoomCallback` 断言——`OnParticipantConnected` 断言入会、`OnDataPacket` 内断言 `*livekit.ChatMessage` 与 RPC 往返（`RegisterRpcCtxMethod`/`PerformRpc`）、`OnDisconnectedWithReason` 断言断开原因；`LIVEKITX_INTEGRATION=1` 开关保留。
- `webhook_test.go`：`NewWebhookKeyProvider` 验真/验假（正确 signing key 通过、错误 key 拒绝）。
- 保留：`config_test.go`（删除 WithStore 相关用例）、`api_test.go`、`token_test.go`。
- 全部测试保留，作为未来参考。

## 文档

- 新建 `docs/livekit-callbacks-guide.md`：以 SDK v2.18.1 `RoomCallback` + `ParticipantCallback` 定义为基准，逐一覆盖全部回调场景——断开（`OnDisconnected`/`OnDisconnectedWithReason`）、参与者入会/离会、活跃说话人、房间 metadata/录制/迁移（`OnRoomMoved`/`OnRoomMovedWithSID`）、重连（`OnReconnecting`/`OnReconnected`）、本地轨道（发布/退订/被订阅）、远端轨道（发布/退订/订阅/退订/失败/静音/取消静音）、参与者 metadata/属性/说话/连接质量、Data（`OnDataPacket` + `DataReceiveParams` 字段）。每场景列：回调签名、字段类型与含义、触发来源、典型用途、Go 示例。另加三节：聊天消息识别、RPC 收发、Webhook 接收。
- `common/livekitx/README.md`：删 16 Hook 章节与 Store 描述，改为 Room API + 原生回调指引并链接新文档。

## 兼容性

- 无现有调用方（`app/meeting` 未开发），删除/重命名 API 不需要迁移垫片。
- 集成测试依赖本地 `livekit-server --dev`（127.0.0.1:7880, devkey/secret），验证环境与之前一致。