# LiveKit 视频会议对接规范

## 适用范围

修改 `common/livekitx`、`app/meeting`、LiveKit JWT/Webhook/Twirp、房间/参与者、Egress、Ingress、SIP 或 Agent 调度时读取。API 版本基线见 [对接指南](../../../docs/livekit-integration-guide.md)，本规范不把本地开发工作树当作稳定版本。

## Scenario: common/livekitx 公共 API 契约

### 1. Scope / Trigger

`common/livekitx` 是零业务依赖的机制层，业务服务通过它完成管理 API、Token、实时连接、Data/RPC、Webhook 和 Hook 分发。修改该包导出 API、事件类型、Hook 签名、Store 或客户端装配时适用。

### 2. Signatures

```go
// 业务系统在服务启动时构造一次并复用。
client, err := livekitx.New(
    livekitx.WithURL(cfg.LiveKit.ServerURL),
    livekitx.WithAPIKey(cfg.LiveKit.APIKey, cfg.LiveKit.APISecret),
    // 可选：注入 go-zero httpc.Service 或标准 *http.Client；两者不能同时注入。
    livekitx.WithHTTPService(httpc.NewService("livekit")),
    // 可选：替换默认内存 Store，用于分布式连接状态（只存可序列化 ConnectionState）。
    livekitx.WithStore(redisStore),
)
```

- `New(opts ...Option) (*Client, error)`：校验 URL、成对凭据和 transport 冲突；不派生任何默认超时。
- `(*Client).API()`：返回协议生成的管理 client（Room/Egress/Ingress/SIP/AgentDispatch），请求 context 由调用方控制。
- `(*Client).Connect(ctx, token, callback, opts...)`：内部构造回调桥，叠加业务自定义 `*lksdk.RoomCallback` 后接入 Hook dispatcher；返回 `*RealtimeRoom`。
- `(*Client).OnChatMessage(h) Subscription` 等 16 个实例级 Hook：注册返回可注销句柄。
- `(*Client).ReceiveWebhook(ctx, req, signingKey, handler)`：验签成功后分发；同时提供 `OnWebhook` 注册分发。
- `(*Client).Close() error`：幂等；关闭 dispatcher 并清理本进程连接，不关闭业务注入的 HTTP client。

### 3. Contracts

- 请求超时只能由业务 context 控制；`livekitx` 不设置、不覆盖、不派生 deadline。禁止重新引入 Timeout option。
- 管理 API 的请求级 Token 必须带房间范围 `VideoGrant.Room`，否则房间级参与者管理/Data/静音/元数据请求返回 `unauthenticated: permissions denied`。
- Store 只保存可序列化的 `ConnectionState`（room/identity/session/node/status/updated_at）；`*RealtimeRoom` 是本进程连接资源，只能内存注册表管理并本地 `Close`，禁止写入 Redis/DB。
- Webhook 事件可能重复、迟到、乱序或丢失；业务在 handler 中按 event ID 持久化去重，公共包不建默认永久内存去重，不承诺 Exactly Once。
- `Connect` 传入的 `ctx` 同时作为连接级 Hook 的 context 来源；连接断开时取消。

### 4. Validation & Error Matrix

- 空 URL / 缺 key 或 secret / `HTTPClient` 与 `HTTPService` 同时注入 / nil context → `ErrInvalidConfig`（构造期返回）。
- 空 token 或 nil context 调 `Connect` → `ErrInvalidTokenOptions`。
- 关闭后注册或分发 Hook → `ErrClosed`。
- handler panic → recover 为 `*HookPanicError`（带事件类型），不终止读循环。
- Webhook 验签失败 → 直接返回签名错误，不调用任何 handler。
- 已关闭 client 调 `ReceiveWebhook` → `ErrClosed`。

### 5. Good/Base/Bad Cases

- Good：`client.OnChatMessage` 注册后，SDK Data 回调收到 `*livekit.ChatMessage` 或 topic=`ChatTopic` 的 `UserDataPacket` 时真实触发；业务在 handler 落库、审核、算未读数。
- Base：业务叠加自定义 `RoomCallback`，`Connect` 内部按 SDK Merge 语义先执行 livekitx 桥接再执行自定义回调。
- Bad：用 `map[*RealtimeRoom]struct{}`/`sync.Map` 冒充分布式连接状态，或把 `*RealtimeRoom` 存进 Redis 由另一节点调用 `Disconnect`。

### 6. Tests Required

- 单元：dispatcher 注册顺序、注销、并发、panic、关闭；Data→ChatMessage 双路径；Webhook 验签失败不调 handler、未知事件安全处理。
- Twirp mock：认证 grant、5 个服务路由/方法/请求体、房间级 grant 限定、Twirp code/message 保留、context 取消不发请求。
- 集成（本地 `livekit-server --dev`，`LIVEKITX_INTEGRATION=1` 开启，默认跳过）：房间生命周期、Token 入会、真实 Hook 桥接、`PublishChatMessage`→`OnChatMessage`、RPC 往返+元数据、断开原因。
- 断言点：Hook 调用顺序与次数、事件字段、错误类型可 `errors.Is/As`、Store 状态与连接关闭同步、无 goroutine 泄漏。

### 7. Wrong vs Correct

#### Wrong

```go
// SDK v2.18.1 的 NewLiveKitAPI 没有 HTTP client option；WithHTTPClient 是 AgentClientOption，
// 传进去无法编译；不要声称统一入口支持注入。
api, err := lksdk.NewLiveKitAPI(
    lksdk.WithURL(url), lksdk.WithAPIKey(key, secret),
    lksdk.WithHTTPClient(customClient), // 编译失败
)
```

```go
// 空 API key 的 KeyProvider 永远验签失败；不要用 NewSimpleKeyProvider("", key)。
event, err := webhook.ReceiveWebhookEvent(req, auth.NewSimpleKeyProvider("", key))
```

#### Correct

```go
// 需要自定义 transport 时使用协议生成 client + Twirp；httpc.Service 按
// common/alarmx.NewAlarmxHttpClient 的适配模式注入。
transport := httpx.NewTwirpTransport(client, httpc.NewService("livekit"))
```

```go
// 验签必须使用允许任意 key claim 的 KeyProvider，实际按 signing key 校验。
event, err := webhook.ReceiveWebhookEvent(req, webhookKeyProvider(signingKey))
```

## HTTP client 注入约定

- `server-sdk-go/v2 v2.18.1` 的 `NewLiveKitAPI` 只接受 `WithURL`/`WithAPIKey`/`WithToken`，**没有 HTTP client option**（`WithHTTPClient` 属于 AgentClient）。
- 需要注入 transport 时使用 Protocol 生成的 Twirp client（`/twirp/livekit.RoomService/` 等路径）构造，参考 `common/alarmx.NewAlarmxHttpClient` 的结构嵌入模式适配 go-zero `httpc.Service`。
- `HTTPClient`（标准 `*http.Client`）与 `HTTPService`（go-zero）互斥，构造期校验冲突；注入的 client 生命周期归业务所有，`Close` 不关闭它。
- 不得伪造“统一入口已注入 HTTP client”的文档或测试。

## 版本与依赖

## 版本与依赖

- 审计基线为 LiveKit Server `v1.13.6`、`github.com/livekit/server-sdk-go/v2 v2.18.1`。
- Protocol 版本由选定 SDK 的 `go.mod` 解析；不得手工猜测或截断 pseudo-version。
- 本地源码提交只能作为编译/源码证据，必须与稳定 tag 分栏记录。
- 升级后重新执行核心示例的 `go mod tidy`、`go test ./...`，并检查生成字段、枚举、oneof、配置和官方链接。

## 所有权与客户端

`LiveKitAPI` 是管理 API 统一入口。服务启动时通过 `NewLiveKitAPI`、`WithURL`、`WithAPIKey` 构造并复用；通过 `.Room()`、`.Egress()`、`.Ingress()`、`.SIP()`、`.AgentDispatch()` 访问子客户端。SDK 管理调用自动处理认证，但不表示外部执行服务已部署。`ConnectToRoom`/`ConnectToRoomWithToken` 是实时参与者路径，不能与管理 API 权限混淆。

```go
// 服务启动时构造一次；密钥从安全配置注入，不写入日志或响应。
api, err := lksdk.NewLiveKitAPI(
    lksdk.WithURL(cfg.LiveKit.ServerURL),
    lksdk.WithAPIKey(cfg.LiveKit.APIKey, cfg.LiveKit.APISecret),
)
if err != nil {
    return nil, err
}
```

直接使用 Protocol 仅限 Webhook、token 或确有需要的生成 client；不要依据旧版本文档使用未经验证的 `New*JSONClient` 或字段。

## Token 与授权

- 业务服务使用 `auth.NewAccessToken` 签发短期、指定房间、最小权限的 join token；管理 token 不返回客户端。
- `VideoGrant` 的房间、创建、管理、录制、Ingress、Agent、发布/订阅和数据字段必须以锁定 Protocol 生成类型核对；`CanPublish`/`CanSubscribe` 是 `*bool`，用 `grant.SetCanPublish(bool)` 设置，不能直接结构体字面量赋值。
- 聊天/Data 链路 Token 必须设置 `CanPublishData`（`grant.SetCanPublishData(true)`），否则 Data 发布被服务端拒绝。
- SIP 使用 `SIPGrant`，不得写成 `VideoGrant` 字段。
- worker 注册、房间 dispatch 和 Cloud Agents 管理是不同授权边界，不能互相推导。
- API key/secret、Webhook signing key、SIP 密码只能来自安全配置。
- `auth.NewAccessToken(key, secret).ToJWT()` 在 key 为空时报错；Token 测试断言用 `auth.ParseAPIToken(token)` + `verifier.Verify(secret)`，不要在测试中输出 token/secret。

```go
// 只允许指定身份加入指定房间；具体有效期由业务契约确定。
token, err := auth.NewAccessToken(apiKey, apiSecret).
    SetIdentity(identity).
    SetName(name).
    SetValidFor(time.Hour).
    SetVideoGrant(&auth.VideoGrant{RoomJoin: true, Room: room}).
    ToJWT()
```

## 状态与 Webhook

LiveKit 拥有房间、参与者、轨道和录制运行态；zero-service 拥有用户、会议单据、角色授权、审计和业务状态。Webhook 只触发同步或对账，不授予权限，也不替代管理 API。

使用 `webhook.ReceiveWebhookEvent` 处理原始 body 和 `Authorization`；前置中间件不能消费或改写 body。验签的 KeyProvider 必须支持任意 key claim（否则 `NewSimpleKeyProvider("", key)` 永远失败），按 signing key 校验。事件可能重复、迟到、乱序或丢失，消费者必须以 event ID 去重、持久化后异步处理，并对关键状态主动对账。未知事件安全记录并忽略。HTTP 响应、重试和失败传播必须与幂等实现一致，不能承诺 Exactly Once。

## 功能边界

- 房间/参与者管理使用 SDK 对应方法和最小 grant。
- Egress 的管理调用不等于录制成功；需要独立 Egress 服务、存储和网络。
- Ingress、SIP、Agent dispatch 同样需要对应服务、凭据、worker 或运营商连接。
- Cloud failover、Cloud Agents、托管 TURN/运营商能力不能推导为自托管能力。
- `Connector`、`AgentSimulation`、`pkg/cloudagents` 不属于 `common/livekitx` MVP；README 只允许“能力边界”排除说明，不得出现能力入口。
- 请求字段、枚举、输出格式和配置默认值必须以选定版本源码和官方配置样例核对，不能复制历史固定数值。

## Hook 与实时接线

- `Client.Connect` 内部构造回调桥（SDK Merge 语义：先 livekitx 桥接、后业务自定义 `RoomCallback`），把 Room/Participant/Track/Connection/Data/RPC 回调接入 typed Hook；业务不能通过透传 `RoomCallback` 绕过分发。
- 聊天 Hook 由 SDK Data 回调真实触发：`*livekit.ChatMessage` 或 topic=`ChatTopic` 的 `UserDataPacket`；不是只在单元测试里空转。
- `RealtimeRoom.RegisterRPC` 先分发 `OnRPCRequest`（含 `RPCMetadataFromContext` 元数据）再调业务 handler；未注册方法由 SDK 返回 UnsupportedMethod，不触发 Hook。
- `*RealtimeRoom` 是进程内连接资源：本地内存注册表管理、`Close` 本进程断开；分布式状态只通过 Store 保存可序列化 `ConnectionState`，跨节点踢人/租约/选主由业务自行实现，不硬编码进包。
- SDK v2.18.1 的 `PerformRpc` 不接收 context，用 `PerformRpcParams.ResponseTimeout` 控制时限；`OnDisconnected` 回调触发前 disconnect reason 已写入，桥接可安全读取。

## 错误、生命周期与反模式

- 每个外部调用设置 context 超时，保留 Twirp code/message；不要把全部错误改成“服务不可用”。
- ServiceContext 复用 client，关闭服务时释放由本任务创建的资源；异步 Webhook/对账任务必须可退出、重试和去重。
- 不在请求中重复创建 API client，不硬编码 secret，不把 Webhook 当唯一状态源，不在公共包固化业务 grant。
- 不手工拼接 Twirp 请求或绕过已验证 SDK；确需 Protocol client 时先编译验证构造函数和生成类型。

## 部署与验证

生产使用可信 TLS 的 HTTPS/WSS；WebRTC UDP/TCP、ICE、NAT、公网地址、TURN、WebSocket、Redis、多节点负载均衡和独立 Egress/Ingress/SIP/Agent 进程按官方当前配置逐项验收。开放 API 端口不等于媒体链路可用。

验证至少包括：临时 module 锁定稳定 SDK 后 `go mod tidy` 与 `go test ./...`；Token/Webhook/API 的成功、权限失败、超时、重复和边界测试；`git diff --check` 以及文档链接、版本、旧 API、secret 和个人路径扫描。没有 Server、凭据、浏览器媒体、TURN、Redis 集群或外部运营商时，必须报告未完成，不得声称端到端通过。

`common/livekitx` 本地集成测试以 `LIVEKITX_INTEGRATION=1` 显式开启（默认跳过，不影响普通单测），连接 `http://127.0.0.1:7880`、`devkey`/`secret`；覆盖房间生命周期、Token 入会、Hook 桥接、聊天 Data 和 RPC 往返。Egress/Ingress/SIP/Agent 与 Webhook 服务端推送不做真实端到端断言，只能标注环境前置条件。
