# LiveKit 视频会议对接规范

## 适用范围

修改 `common/livekitx`、`app/live`、LiveKit JWT/Webhook/Twirp、房间/参与者、Egress、Ingress、SIP 或 Agent 调度时读取。API 版本基线见 [对接指南](../../../docs/live/integration-guide.md)，本规范不把本地开发工作树当作稳定版本。

## Scenario: common/livekitx 公共 API 契约

### 1. Scope / Trigger

`common/livekitx` 是零业务依赖的机制层：统一配置与生命周期、Twirp 管理 API 访问、Token 签发、Webhook 验签 KeyProvider。**不包装任何 SDK 方法**——业务通过 `client.Room()` 拿到 `livekit.RoomService` 后直接调用 SDK 原生方法，通过 `lksdk.NewRoom()` + `JoinWithContext` 加入房间，通过 `livekitx.NewJoinToken()` 签发 token。**实时事件（断开、重连、轨道、Data、聊天、RPC）全部由业务通过 SDK 原生 `RoomCallback` 处理**，包内不存在任何事件分发机制。修改该包配置/生命周期、Token 构造或 Webhook 验签时适用。

### 2. Signatures

```go
// 业务系统在服务启动时构造一次并复用；Option 直接作用于 Client。
client, err := livekitx.New(
    livekitx.WithURL(cfg.LiveKit.ServerURL),
    livekitx.WithAPIKey(cfg.LiveKit.APIKey, cfg.LiveKit.APISecret),
    livekitx.WithHTTPService(httpc.NewService("livekit")), // 可选
)

// 管理 API：业务直接使用 SDK 原生 RoomService 方法
roomSvc := client.Room()
roomSvc.CreateRoom(ctx, &livekit.CreateRoomRequest{Name: roomName, ...})
roomSvc.DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: roomName})
roomSvc.RemoveParticipant(ctx, &livekit.RoomParticipantIdentity{Room: roomName, Identity: id})
roomSvc.SendData(ctx, &livekit.SendDataRequest{Room: roomName, Topic: &topic, Data: payload, Kind: livekit.DataPacket_RELIABLE, DestinationIdentities: dests})
roomSvc.MutePublishedTrack(ctx, &livekit.MuteRoomTrackRequest{Room: roomName, Identity: id, TrackSid: sid, Muted: muted})
roomSvc.ListParticipants(ctx, &livekit.ListParticipantsRequest{Room: roomName})

// 加入房间：SDK 原生方式
room := lksdk.NewRoom(&lksdk.RoomCallback{...})
room.JoinWithContext(ctx, cfg.LiveKit.ServerURL, lksdk.ConnectInfo{
    APIKey: key, APISecret: secret, RoomName: roomName, ParticipantIdentity: identity,
})
defer room.Disconnect()

// 入会 Token 签发：直接调用函数
token, err := livekitx.NewJoinToken(livekitx.JoinTokenOptions{
    APIKey: apiKey, APISecret: apiSecret, Room: roomName, Identity: identity,
    ValidFor: validFor, CanPublish: true, CanSubscribe: true, CanPublishData: true,
})

// Webhook：业务自行验签
event, err := webhook.ReceiveWebhookEvent(req, livekitx.NewWebhookKeyProvider(cfg.WebhookKey))
```

- `New(opts ...Option) (*Client, error)`：Option 为 `func(*Client)`，配置类（URL/凭据/HTTP）写入 `c.config`；校验 URL、成对凭据和 transport 冲突；不派生任何默认超时。
- `(*Client).Room() livekit.RoomService`：返回底层 SDK RoomService，供业务直接调用。
- `(*Client).API() *API`：返回协议生成的管理 client（Room/Egress/Ingress/SIP/AgentDispatch）。
- `(*Client).Config() Config`：返回构造时的配置副本。
- `(*Client).Close() error`：仅幂等标记关闭；不管理实时连接、不关闭业务注入的 HTTP client。

### 3. Contracts

- 请求超时只能由业务 context 控制；`livekitx` 不设置、不覆盖、不派生 deadline。禁止重新引入 Timeout option。
- 管理 API 的请求级 Token 必须带房间范围 `VideoGrant.Room`，否则房间级参与者管理/Data/静音/元数据请求返回 `unauthenticated: permissions denied`。
- 实时回调是 SDK 原生 `*lksdk.RoomCallback`，原样透传（无桥接、无 Merge、无合成事件）；`JoinWithContext` 成功返回即"已连接"（SDK 原生无 connected 回调）；断开由业务调 `room.Disconnect()`。
- 聊天/富媒体约定交业务：文本可用 `PublishChatMessage`/`*livekit.ChatMessage`；语音、图片等二进制走 `SendData`（`UserDataPacket`，接收方在 `OnDataPacket` 收到），topic 与 payload 格式由业务定义。
- Webhook 事件可能重复、迟到、乱序或丢失；业务在 handler 中按 event ID 持久化去重，公共包不建默认永久内存去重，不承诺 Exactly Once。
- `NewJoinToken` 签发的 token 只用于客户端直连；加入房间走 `ConnectInfo` 固定最小 grant（RoomJoin+Room），业务需要自定义权限时用 `NewJoinToken(opts)` 或 SDK 原生 `JoinWithContextAndToken`。

### 4. Validation & Error Matrix

- 空 URL / 缺 key 或 secret / `HTTPClient` 与 `HTTPService` 同时注入 → `ErrInvalidConfig`（构造期返回）。
- Webhook 验签失败 → `webhook.ReceiveWebhookEvent` 返回签名错误，不调用任何业务代码。
- 管理 API 错误保留 Twirp code/message，不包装成"服务不可用"。

### 5. Good/Base/Bad Cases

- Good：`client.Room().CreateRoom(ctx, req)` 直接使用 SDK；`lksdk.NewRoom(cb)` + `JoinWithContext` 加入房间；`livekitx.NewJoinToken(opts)` 签发 token。
- Base：同一 `Client` 多次调用 `Room()` 获取 RoomService（每次返回同一实例）。
- Bad：试图调用 `client.CreateRoom()`、`client.JoinRoom()` 等已删除的 wrapper 方法。

### 6. Tests Required

- 单元（`api_test.go`）：Twirp mock 断言管理 API 请求路由与认证。
- `token_test.go`：`NewJoinToken` 用 `auth.ParseAPIToken` + `verifier.Verify(secret)` 断言 grant。
- `webhook_test.go`：`NewWebhookKeyProvider` 验真/验假。
- 集成（`integration_test.go`，`LIVEKITX_INTEGRATION=1` 开启）：用 `joinRoomHelper` 直接调用 SDK 加入房间，断言原生回调、聊天、RPC、SendData、踢人。
- 断言点：回调调用次数与字段、错误类型可 `errors.Is`、管理请求体字段。

### 7. Wrong vs Correct

#### Wrong

```go
// 1. 调用已删除的 wrapper 方法
client.CreateRoom(ctx, &livekit.CreateRoomRequest{Name: "room"})
client.JoinRoom(ctx, "room", "user")
client.DeleteRoom(ctx, "room")
client.SendData(ctx, "room", "topic", payload)

// 2. 调用已删除的 JoinToken 方法
token, _ := client.JoinToken("room", "user", time.Hour)

// 3. 使用已删除的 WithCallback/WithConnectOption
room, _ := client.JoinRoom(ctx, "room", "user", livekitx.WithCallback(cb))
```

#### Correct

```go
// 1. 直接使用 SDK
roomSvc := client.Room()
roomSvc.CreateRoom(ctx, &livekit.CreateRoomRequest{Name: "room"})
roomSvc.DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: "room"})
roomSvc.SendData(ctx, &livekit.SendDataRequest{Room: "room", Topic: &topic, Data: payload})

// 2. 用函数签发 token
token, _ := livekitx.NewJoinToken(livekitx.JoinTokenOptions{...})

// 3. 用 SDK 原生方式加入房间
room := lksdk.NewRoom(&lksdk.RoomCallback{...})
room.JoinWithContext(ctx, url, lksdk.ConnectInfo{...})
```

## 会议业务服务（app/live）规范

适用：`app/live`（Live 会议 gRPC 业务服务）及其网关（livegtw）。机制层约定见上文。

### 1. Webhook 链路（验签 → 原始字节 → SDK 对象处理）

- 链路：LiveKit 推送 → livegtw 用 `webhook.ReceiveWebhookEvent` 验签（失败 401，不调用业务）→ 把 `*livekit.WebhookEvent` **proto 序列化后的原始字节**经 `WebhookNotify(WebhookNotifyReq{Data: bytes})` 透传给 live 服务 → live 服务 `proto.Unmarshal` 解析为 SDK 对象后处理业务。禁止在 proto 中做字段扁平化（eventId/eventType/roomName/...），会丢失 SDK 结构。
- **不做 TTL 幂等**：LiveKit 事件可能重复、迟到、补发（对账闭环需要重放），而处理操作本身幂等（参会记录 `Where+Assign+FirstOrCreate` upsert、会议状态仅 active→ended 流转、未知会议安全忽略），重复/补发重放结果一致。禁止再加 event ID 去重键阻碍补发。
- **事件全 case**：`room_finished`/`participant_joined`/`participant_left` 处理；`room_started` 无需处理（创建即 active）；`participant_connection_aborted`/`track_published`/`track_unpublished`/`egress_started`/`egress_updated`/`egress_ended`/`ingress_started`/`ingress_ended` 建 case 标注 TODO 并记日志；未知事件安全忽略。禁止 default 静默吞掉已知事件。

### 2. 会议业务约定

- 会议号：`tool.IdUtil.NextId("M", "live")`（Redis 序号 + 日期；`category` 参数按业务域隔离，防止与其他服务撞号）。
- Redis key：统一以业务域前缀开头（如 `live:lock:meeting:*`、`live:outId_M`），最前段系统 key 由 redis 配置自动附加。
- 入会 token：`CanPublish`/`CanSubscribe`/`CanPublishData` 全开（否则参与者无法发布媒体/发 Data）；业务直接调用 `livekitx.NewJoinToken(opts)` 签发。
- 时间输出：RPC 出参时间统一 `carbonx.FormatDateTimeOrEmpty`/`FormatNullDateTime`（`yyyy-MM-dd HH:mm:ss` 字符串），不使用时间戳。
- 创建人/更新人/机构：从 gRPC metadata 取（`grpcx.LoggerInterceptor` 注入 `x-user-id` 等 → `authctx.GetUserId`），proto 入参不传；模型保留 `create_user`/`update_user`/`dept_code`。
- 并发控制：`redis.NewRedisLock(r, key)` 直接构造（go-zero RedisLock，Lua 原子 + `SetExpire` TTL 自动释放），`AcquireCtx` 返回 `(bool, error)` 区分"未获得锁"与"Redis 错误"。
- **会议锁规范**：同一会议的所有操作（结束、加入、票据加入）必须使用同一把分布式锁，锁 key 统一为 `live:lock:meeting:{meetingNo}`，TTL 10 秒。禁止为不同操作使用不同锁 key（如 `:end`、`:join` 后缀），否则无法防止会议结束与加入的并发冲突。锁前缀定义在 `helper.go` 的 `redisMeetingLockPrefix` 常量中。

  ```go
  // ✓ 正确：使用统一的会议锁 key
  lock := redis.NewRedisLock(l.svcCtx.Redis, redisMeetingLockPrefix+meetingNo)
  lock.SetExpire(meetingLockTTL)
  ok, err := lock.AcquireCtx(l.ctx)
  if err != nil {
      return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_03_CACHE, err, "获取会议锁失败")
  }
  if !ok {
      return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_REPEAT, "会议正在被操作，请稍后重试")
  }
  defer lock.Release()

  // ✗ 错误：使用带后缀的锁 key（无法防止并发）
  lock := redis.NewRedisLock(l.svcCtx.Redis, redisMeetingLockPrefix+meetingNo+":end")
  ```

- **meeting_code 生成**：9位数字（100000000-999999999），用户输入的会议号。生成方式：`tool.RandomDigits(9)`（内部用 lancet `random.RandNumberOfLength`，math/rand 非 crypto/rand）。唯一性保证：分布式锁 `live:lock:meeting_code_gen`（TTL 5s）+ DB 唯一索引 + 代码层3次重试。锁前缀定义在 `helper.go` 的 `redisMeetingCodeLockPrefix` 常量中。存储：`live_meetings.meeting_code` 字段，普通索引（非唯一索引，唯一性由代码层保证）。创建会议时生成，插入失败则重试，3次都失败则报错。

- **会议号二选一查询模式**：业务接口（JoinMeeting、GenerateMeetingTicket 等）支持 `meeting_no` / `meeting_code` 二选一。前端传 `meeting_no` → 直接查询（`meeting_no` = LiveKit 房间名）；传 `meeting_code` → 先查 `meeting_no`，再查房间；两者都传 → 优先使用 `meeting_no`；两者都不传 → 返回错误。repo 层提供 `GetMeetingByCode(ctx, code)` 方法，返回完整 meeting 对象。

  ```protobuf
  // proto 定义示例
  message JoinMeetingReq {
      // 会议号（与 meeting_code 二选一）
      string meeting_no = 1;
      // 用户会议号（9位数字，与 meeting_no 二选一）
      string meeting_code = 2;
  }
  ```

- **LiveKit 房间 Sid 保存**：`CreateRoom` 返回的 `room.Sid` 保存到数据库，便于 Egress/Webhook 等场景使用。

  ```go
  room, err := l.svcCtx.LiveKit.Room().CreateRoom(l.ctx, &livekit.CreateRoomRequest{...})
  // room.Sid 保存到 live_meetings.room_sid 字段
  meeting := &gormmodel.LiveMeeting{
      RoomSid: room.Sid,
      // ...
  }
  ```

- **MeetingInfo proto 字段编号**：

  | 字段 | 编号 | 说明 |
  |------|------|------|
  | meeting_no | 1 | 业务会议号 |
  | meeting_code | 2 | 用户会议号 |
  | title | 3 | 标题 |
  | status | 4 | 状态 |
  | create_user | 5 | 创建人 |
  | update_user | 6 | 更新人 |
  | dept_code | 7 | 机构 |
  | start_time | 8 | 开始时间 |
  | end_time | 9 | 结束时间 |
  | create_time | 10 | 创建时间 |
  | empty_timeout | 11 | 无人房间保留秒数 |
  | departure_timeout | 12 | 所有人离开后保留秒数 |
  | max_participants | 13 | 最大参会人数 |
  | room_sid | 14 | LiveKit 房间 Sid |

  新增字段从 15 开始编号。

### 3. 模型风格

新业务表（会议等）按 `app/trigger/model/gormmodel` 的 plan 系列风格：`gormx.LegacyStringBaseModel`（string 主键 + create_time/update_time + is_deleted 软删）+ `gormx.VersionMixin` + `CreateUser`/`UpdateUser`/`DeptCode`（sql.NullString）+ 可空字段用 `sql.NullString`/`sql.NullTime` + `int` 状态 + 索引名 `idx_<表名>_<字段>`。

### 4. 业务错误码

业务错误统一 extproto + `tool.NewErrorByPbCode`/`NewErrorByPbCodeWrap`（reason=六位错误码，HTTP 自动映射），禁止裸 `status.Error`/`status.Errorf`。常用映射：参数 `101101`、记录不存在 `102102`、记录已存在 `102103`、缓存/Redis `103101`、未认证 `104101`、业务状态不允许 `105102`、重复操作 `105103`、DB `102101`、第三方（LiveKit）`106102`。

## 版本与依赖

- 审计基线为 LiveKit Server `v1.13.6`、`github.com/livekit/server-sdk-go/v2 v2.18.1`。
- Protocol 版本由选定 SDK 的 `go.mod` 解析；不得手工猜测或截断 pseudo-version。
- 升级后重新执行核心示例的 `go mod tidy`、`go test ./...`，并检查生成字段、枚举、oneof、配置和官方链接。

## Token 与授权

- 业务服务使用 `auth.NewAccessToken` 签发短期、指定房间、最小权限的 join token；管理 token 不返回客户端。`livekitx` 提供 `NewJoinToken(opts JoinTokenOptions)`（自定义 grant 场景）。
- `VideoGrant` 的房间、创建、管理、录制、Ingress、Agent、发布/订阅和数据字段必须以锁定 Protocol 生成类型核对；`CanPublish`/`CanSubscribe` 是 `*bool`，用 `grant.SetCanPublish(bool)` 设置，不能直接结构体字面量赋值。
- 聊天/Data 链路 Token 必须设置 `CanPublishData`（`grant.SetCanPublishData(true)`），否则 Data 发布被服务端拒绝。
- SIP 使用 `SIPGrant`，不得写成 `VideoGrant` 字段。
- API key/secret、Webhook signing key、SIP 密码只能来自安全配置。
- `auth.NewAccessToken(key, secret).ToJWT()` 在 key 为空时报错；Token 测试断言用 `auth.ParseAPIToken(token)` + `verifier.Verify(secret)`，不要在测试中输出 token/secret。

## 状态与 Webhook

LiveKit 拥有房间、参与者、轨道和录制运行态；zero-service 拥有用户、会议单据、角色授权、审计和业务状态。Webhook 只触发同步或对账，不授予权限，也不替代管理 API。

## 实时连接与回调

- 业务直接使用 SDK 原生 `*lksdk.RoomCallback`，通过 `lksdk.NewRoom(callback)` 构造，`room.JoinWithContext` 加入。
- `JoinWithContext` 成功返回即"已连接"（SDK 原生无 connected 回调）；断开由业务调 `room.Disconnect()`。
- 聊天识别是业务职责：`*livekit.ChatMessage`（文本）与 `UserDataPacket`（业务自定义 topic）都在 `OnDataPacket` 到达。
- 服务端 RPC（`client.Room().PerformRpc()`）是服务端→单个客户端的定向请求-响应。
- SendData 与 PerformRpc 选型：只通知不关心结果→ `SendData`；要客户端执行并返回结果→ `PerformRpc`。
- 服务端 RPC 错误诊断矩阵（`rpc_self_test.go` 实测验证）：

| 错误 | 原因 | 排查方向 |
|------|------|---------|
| `RpcError 1400: Method not supported at destination` | 目标客户端**在线但未注册**该 method | 检查目标端 `registerRpcMethod` 是否执行（刷新页面后注册会丢失） |
| `no response from servers` | 目标参与者**不存在或已离线** | 用 ListParticipants 确认目标在线 |
| `ResponseTimeout` 类错误 | 目标在线、已注册，但 handler 未在时限内返回 | handler 阻塞（如等用户输入）或网络问题 |

- 目标参与者可以是调用方自身（自发自收），LiveKit 服务端按 identity 路由，不做 caller≠destination 校验；排查 RPC 失败时不要怀疑"自己发给自己"。
- 管理 API 方法名以锁定 protocol 源码核对：静音是 `MutePublishedTrack`；结束会议统一用 `DeleteRoom`。

## 错误、生命周期与反模式

- 每个外部调用设置 context 超时，保留 Twirp code/message。
- ServiceContext 复用 client，关闭服务时释放。
- 不在请求中重复创建 API client，不硬编码 secret，不在公共包固化业务 grant。

## 部署与验证

生产使用可信 TLS 的 HTTPS/WSS；WebRTC UDP/TCP、ICE、NAT、公网地址、TURN、WebSocket、Redis、多节点负载均衡和独立 Egress/Ingress/SIP/Agent 进程按官方当前配置逐项验收。开放 API 端口不等于媒体链路可用。

验证至少包括：临时 module 锁定稳定 SDK 后 `go mod tidy` 与 `go test ./...`；Token/Webhook/API 的成功、权限失败、超时、重复和边界测试；`git diff --check` 以及文档链接、版本、secret 和个人路径扫描。没有 Server、凭据、浏览器媒体、TURN、Redis 集群或外部运营商时，必须报告未完成，不得声称端到端通过。

`common/livekitx` 本地集成测试以 `LIVEKITX_INTEGRATION=1` 显式开启（默认跳过，不影响普通单测），连接 `http://127.0.0.1:7880`、`devkey`/`secret`；覆盖房间生命周期、SDK 原生入会、原生回调（入会/聊天双路径/RPC 往返/断开原因）、`SendData` 富媒体投递。Egress/Ingress/SIP/Agent 与 Webhook 服务端推送不做真实端到端断言，只能标注环境前置条件。

## 网关（livegtw）标准开发规范

适用：HTTP 网关 `app/livegtw`，转发请求到 `app/live` gRPC 服务。

### 1. 开发流程

```
1. 编写 livegtw.api 定义接口（类型定义 + 路由 + 鉴权组）
2. 执行 gen.sh 生成 handler / logic / types / routes
3. 在 logic 文件中实现业务逻辑（调用 gRPC client）
4. 特性钩子类（webhook、测试页）不走 api 定义，直接在 livegtw.go 配置路由
```

### 2. 命名规范（api 层 vs gRPC 层）

| 层 | 请求 | 响应 | 说明 |
|----|------|------|------|
| `livegtw.api` | `XxxRequest` | `XxxReply` | HTTP 网关类型，goctl 生成到 `types` 包 |
| `live.proto` | `XxxReq` | `XxxRes` | gRPC 类型，protoc 生成到 `live` 包 |

- 两层类型名**不同**（`Request/Reply` vs `Req/Res`），避免同包同名冲突
- logic 中做映射：`types.XxxRequest` → `live.XxxReq`，`live.XxxRes` → `types.XxxReply`

### 3. 文件结构与职责

```
app/livegtw/
├── livegtw.api              # API 定义文件
├── gen.sh                   # 代码生成脚本
├── livegtw.go               # 主入口
├── internal/
│   ├── config/config.go     # 配置结构
│   ├── svc/servicecontext.go # ServiceContext（持有 gRPC client、中间件）
│   ├── handler/
│   │   ├── routes.go        # [生成] 路由注册
│   │   ├── meeting/         # [生成] meeting 组 handler（httpx 默认格式）
│   │   ├── ticket/          # [生成] 免鉴权组 handler
│   │   ├── webhook/         # [手写] webhook handler（不走 api 定义）
│   │   └── testpage/        # [手写] 测试页 handler
│   ├── logic/
│   │   ├── meeting/         # [生成+手写] meeting 组 logic（含 meeting_helper.go）
│   │   ├── ticket/          # [生成+手写] 免鉴权组 logic
│   │   └── webhook/         # [手写] webhook logic
│   ├── middleware/           # [生成+手写] 中间件
│   │   └── meetingauthmiddleware.go
│   └── types/               # [生成] 请求/响应类型（XxxRequest/XxxReply）
```

### 4. API 定义规范（livegtw.api）

```go
// 类型定义：请求用 XxxRequest，响应用 XxxReply
type CreateMeetingRequest {
    Title string `json:"title"`
}

type CreateMeetingReply {
    Meeting MeetingInfo `json:"meeting"`
}

// 业务接口组：需要 JWT + 中间件
@server (
    prefix:     live/v1
    group:      live
    jwt:        JwtAuth
    middleware: MeetingAuth
)
service livegtw {
    @doc "创建会议"
    @handler createMeeting
    post /createMeeting (CreateMeetingRequest) returns (CreateMeetingReply)
}

// 免鉴权组：不声明 jwt/middleware（如票据加入会议）
@server (
    prefix: live/v1
    group:  ticket
)
service livegtw {
    @doc "根据票据加入会议（无需JWT）"
    @handler joinMeetingByTicket
    get /joinMeetingByTicket (JoinMeetingByTicketRequest) returns (JoinMeetingByTicketReply)
}
```

- 业务接口必须声明 `jwt: JwtAuth` 和 `middleware: MeetingAuth`
- 免鉴权接口单列一个 `@server` 块，不写 `jwt`/`middleware`
- 查询类用 `get`，写入类用 `post`；get 的请求参数 tag 用 `form`, post 用 `json`
- webhook、测试页等特性钩子不走 api 定义，直接在 `livegtw.go` 配置路由
- **路由命名与 gRPC 接口保持一致**：handler 名和路径都使用小驼峰，与 gRPC 方法名对应（如 `CreateMeeting` → `createMeeting` → `/createMeeting`）
- **例外：组合业务路由**：如果网关接口是多个 gRPC 调用组合的业务逻辑（非直接转发），则按前端业务语义命名，不必与 gRPC 一致
- **例外：同一 gRPC 多个前端路由**：如"全部会议列表"和"我的会议列表"都调用 `ListMeetings`，但前端路由应分别命名为 `/listMeetings` 和 `/myMeetings`（后者在 logic 中自动注入 identity 参数）
- **网关自有接口**：如 `/getCurrentUser`（从 authctx 获取当前用户信息），不调用 gRPC，直接在网关 logic 实现

### 5. 中间件规范

```go
// internal/middleware/meetingauthmiddleware.go
type MeetingAuthMiddleware struct {
    claimMapping map[string]string
}

func NewMeetingAuthMiddleware(claimMapping map[string]string) *MeetingAuthMiddleware {
    return &MeetingAuthMiddleware{claimMapping: claimMapping}
}

func (m *MeetingAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        if auth := r.Header.Get("Authorization"); auth != "" {
            ctx = authctx.WithAuthType(ctx, "user")
            ctx = authctx.WithAuthorization(ctx, auth)
        }
        ctx = authctx.BridgeJWTClaims(ctx, m.claimMapping)
        next(w, r.WithContext(ctx))
    }
}
```

- 中间件在 `ServiceContext` 中初始化：`m := middleware.NewMeetingAuthMiddleware(c.JwtAuth.ClaimMapping)`
- `routes.go` 通过 `serverCtx.MeetingAuth` 引用
- **用户身份**通过 `authctx.GetUserId(ctx)` / `authctx.GetUserName(ctx)` 获取，写入 gRPC metadata 透传

### 6. Logic 实现规范

```go
// internal/logic/meeting/createmeetinglogic.go
type CreateMeetingLogic struct {
    logx.Logger
    ctx    context.Context
    svcCtx *svc.ServiceContext
}

func NewCreateMeetingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateMeetingLogic {
    return &CreateMeetingLogic{
        Logger: logx.WithContext(ctx),
        ctx:    ctx,
        svcCtx: svcCtx,
    }
}

func (l *CreateMeetingLogic) CreateMeeting(req *types.CreateMeetingRequest) (resp *types.CreateMeetingReply, err error) {
    r, err := l.svcCtx.LiveRpcCli.CreateMeeting(l.ctx, &live.CreateMeetingReq{Title: req.Title})
    if err != nil {
        return nil, err
    }
    return &types.CreateMeetingReply{Meeting: toMeetingInfo(r.GetMeeting())}, nil
}
```

- 每个接口一个单独的 logic 文件（goctl 标准拆分模式）
- 公共转换函数（`toMeetingInfo`/`toParticipantInfo`）放在 `meeting_helper.go`，可以被同组 logic 复用
- **logic 只做请求转发和类型转换**，不写业务编排/单测
- 依赖登录用户的接口（join、listMyMeetings）用 `authctx.GetUserId(l.ctx)` 取身份，不使用请求体传

#### Logic 函数签名分类

根据 `.api` 定义是否有返回类型，Logic 函数签名分为两类：

| 类型 | `.api` 定义 | Logic 签名 | Handler 使用 |
|------|------------|-----------|-------------|
| 有返回 | `post /xxx (Req) returns (Reply)` | `func (l *XxxLogic) Xxx(req *types.XxxRequest) (resp *types.XxxReply, err error)` | `resp, err := l.Xxx(&req)` |
| 无返回 | `post /xxx (Req)` | `func (l *XxxLogic) Xxx(req *types.XxxRequest) error` | `err := l.Xxx(&req)` |

```go
// 有返回的 Logic
func (l *EndMeetingLogic) EndMeeting(req *types.EndMeetingRequest) (resp *types.EndMeetingReply, err error) {
    r, err := l.svcCtx.LiveRpcCli.EndMeeting(l.ctx, &live.EndMeetingReq{MeetingNo: req.MeetingNo})
    if err != nil {
        return nil, err
    }
    return &types.EndMeetingReply{}, nil
}

// 无返回的 Logic（void 操作）
func (l *EndMeetingLogic) EndMeeting(req *types.EndMeetingRequest) error {
    _, err := l.svcCtx.LiveRpcCli.EndMeeting(l.ctx, &live.EndMeetingReq{MeetingNo: req.MeetingNo})
    return err
}
```

#### Proto 与 HTTP 类型转换

proto 和 HTTP 类型可能不一致，需要手动转换：

| Proto 类型 | HTTP 类型 | 转换方式 |
|-----------|----------|---------|
| `uint32` | `int32` | `uint32(req.ExpireSeconds)` |
| `[]byte` | `string` | `[]byte(req.Payload)` |
| `int64` | `int64` | 直接赋值 |
| `string` | `string` | 直接赋值 |

```go
// 示例：uint32 vs int32
r, err := l.svcCtx.LiveRpcCli.GenerateMeetingTicket(l.ctx, &live.GenerateMeetingTicketReq{
    ExpireSeconds: uint32(req.ExpireSeconds), // proto 用 uint32，HTTP 用 int32
})

// 示例：[]byte vs string
r, err := l.svcCtx.LiveRpcCli.SendMeetingData(l.ctx, &live.SendMeetingDataReq{
    Payload: []byte(req.Payload), // proto 用 []byte，HTTP 用 string
})
```

#### 列表响应转换

proto 返回 `[]*live.XxxInfo`，HTTP 返回 `[]types.XxxInfo`，需要逐个转换：

```go
items := make([]types.XxxInfo, 0, len(r.GetItems()))
for _, item := range r.GetItems() {
    items = append(items, toXxxInfo(item)) // 使用 helper 函数
}
return &types.XxxReply{Items: items, Total: r.GetTotal()}, nil
```

### 7. Handler 规范

handler 使用 `xhttp.JsonBaseResponseCtx`（统一响应格式 `{code, msg, data}`，gRPC 错误自动转换）：

```go
import xhttp "github.com/zeromicro/x/http"

var req types.CreateMeetingRequest
if err := httpx.Parse(r, &req); err != nil {
    xhttp.JsonBaseResponseCtx(r.Context(), w, err)
    return
}
l := meeting.NewCreateMeetingLogic(r.Context(), svcCtx)
resp, err := l.CreateMeeting(&req)
if err != nil {
    xhttp.JsonBaseResponseCtx(r.Context(), w, err)
} else {
    xhttp.JsonBaseResponseCtx(r.Context(), w, resp)
}
```

- `xhttp.JsonBaseResponseCtx` 内部调用 `wrapBaseResponse`，自动处理 gRPC status error（提取 code + message）和普通 error（code=-1）
- 成功响应：`code=0, msg="ok", data=<resp>`
- 错误响应：HTTP 200 + `{code: <grpc-code或-1>, msg: "<错误信息>"}`
- **标准网关写法**：统一使用 `xhttp.JsonBaseResponseCtx`，除非用户特殊要求返回其他格式

### 8. ServiceContext 规范

```go
type ServiceContext struct {
    Config     config.Config
    LiveRpcCli live.LiveRpcClient    // gRPC 客户端
    MeetingAuth rest.Middleware       // 中间件
}

func NewServiceContext(c config.Config) *ServiceContext {
    logx.Must(logx.SetUp(c.Log))
    m := middleware.NewMeetingAuthMiddleware(c.JwtAuth.ClaimMapping)
    return &ServiceContext{
        Config: c,
        LiveRpcCli: live.NewLiveRpcClient(zrpc.MustNewClient(c.LiveRpcConf,
            zrpc.WithUnaryClientInterceptor(grpcx.UnaryMetadataInterceptor)).Conn()),
        MeetingAuth: m.Handle,
    }
}
```

### 9. 主入口（livegtw.go）规范

```go
func main() {
    // ... 配置加载 ...
    server := rest.MustNewServer(c.RestConf, gtwx.CorsOption())

    // 请求日志中间件（method/path/start time 写入 context）
    server.Use(gtwx.RequestLogMiddleware)

    // 响应日志（仅记录业务错误，成功由 go-zero 标准日志覆盖）
    gtwx.SetLogOkHandler()

    ctx := svc.NewServiceContext(c)

    // 业务 API 路由（通过 routes.go 注册）
    handler.RegisterHandlers(server, ctx)

    // 特性钩子路由（直接配置，不走 api 定义）
    server.AddRoute(rest.Route{
        Method:  http.MethodPost,
        Path:    "/webhook/livekit",
        Handler: webhook.LiveKitWebhookHandler(ctx),
    })

    // 测试页（可选）
    if c.EnableTestPage {
        server.AddRoute(rest.Route{
            Method:  http.MethodGet,
            Path:    "/test/meeting",
            Handler: testpage.MeetingTestPageHandler(),
        })
    }
}
```

#### 网关日志模式

- **不要调用 `gtwx.SetGrpcErrorHandler()`**（deprecated）：handlers 统一走 `xhttp.JsonBaseResponseCtx`，gRPC 错误已由 `wrapBaseResponse` 内置转换为 `{code, msg}` 响应体。
- **`RequestLogMiddleware`**：把 `method`、`path`、`start time` 写入 context，供 ok handler 读取。
- **`SetLogOkHandler`**：只在业务错误（`code != 0`）时打印 error 日志（含 method、path、duration、code、msg），成功请求由 go-zero 标准 `LogHandler` 覆盖，不重复打 info。
- 日志效果：
  ```
  [HTTP] POST /live/v1/live/createMeeting  duration=12ms  code=102102  msg="meeting not found"
  ```

### 10. gRPC 层约定（app/live）

- **按表字段简单检索**，RPC 层不做复杂业务编排。例如会议列表查询就是按 `status`/`create_user`/`title`/`identity` 等字段条件过滤，逻辑放在 repo 的 where 子句。
- 新增的检索条件（如"查某用户相关的会议"）通过给 `ListMeetingsReq` 加一个 `identity` 字段实现，**不要单开一个 `ListMyMeetings` RPC**。
- 网关 /myList 就是调用同一个 `ListMeetings`，传当前用户 identity。**避免为同一查询开多个 RPC。**

### 11. 代码生成注意事项

- `gen.sh` 会用 goctl 生成 scaffold（skeleton），需要手动填 logic 业务逻辑
- **从 `.api` 生成时只保留最新结构**：如果之前手工架过 logic，重新生成前先 `rm -rf internal/handler internal/logic internal/types`，避免新旧命名文件（驼峰 vs 下划线）共存导致重复定义
- **不要**把多个 logic 合并进一个 `meetinglogic.go`（合并文件模式和 goctl 的拆分模式冲突，会让后续 `gen.sh` 生成重复定义）。旧合并文件应删掉，改成拆分文件。
- 生成文件不要手工改结构（struct 定义/函数签名），只填 logic 方法体

### 12. 票据系统设计

#### 票据类型

| 类型 | 值 | 说明 |
|------|---|------|
| 一次性票据 | 1（默认） | 消费后删除，只能使用一次 |
| 有效期票据 | 2 | 消费后保留至过期，可多次使用（挤掉旧设备） |

#### Redis 存储

- Key：`live:ticket:{ticket}`（单个票据 key）
- Value：JSON 字符串，包含会议号、身份、名称、过期时间、权限、票据类型
- TTL：由 `expire_seconds` 参数决定（秒）

#### 票据消费逻辑

```go
// 根据票据类型处理：一次性票据删除，有效期票据保留
if data.TicketType == 1 {
    // 一次性票据：删除 individual key
    l.svcCtx.Redis.DelCtx(l.ctx, ticketKey)
}
```

#### 过期时间校验

即使 Redis 有 TTL，也需要在代码中校验 `expireTime` 字段：

```go
// 校验票据是否过期（expireTime 格式：2006-01-02 15:04:05）
if data.ExpireTime != "" {
    expireT, err := time.ParseInLocation("2006-01-02 15:04:05", data.ExpireTime, time.Local)
    if err == nil && time.Now().After(expireT) {
        return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST, "票据已过期")
    }
}
```

### 13. 测试注意

- `fakeLiveRpcCli`（webhook 测试用）必须实现 `live.LiveRpcClient` 的**全部**方法。gRPC 接口一旦新增/删除方法，`helpers_test.go` 里的 fake 需同步补齐/删除对应方法，否则 `go test ./...` 构建失败。
- 未鉴权路由（ticket 组）的 handler 在 `internal/handler/ticket/`，logic 在 `internal/logic/ticket/`，与 meeting 组分开。

### Common Mistakes

#### Common Mistake: Proto 与 HTTP 类型不匹配

**Symptom**: 编译错误 `cannot use int32 as uint32 value` 或 `cannot use string as []byte value`

**Cause**: proto 定义使用 `uint32`/`[]byte`，但 HTTP API 定义使用 `int32`/`string`，直接赋值导致类型不匹配

**Fix**: 在 logic 中手动转换类型

```go
// Wrong
r, err := l.svcCtx.LiveRpcCli.GenerateMeetingTicket(l.ctx, &live.GenerateMeetingTicketReq{
    ExpireSeconds: req.ExpireSeconds, // int32 → uint32 编译错误
})

// Correct
r, err := l.svcCtx.LiveRpcCli.GenerateMeetingTicket(l.ctx, &live.GenerateMeetingTicketReq{
    ExpireSeconds: uint32(req.ExpireSeconds), // 手动转换
})
```

**Prevention**: 实现 logic 前先检查 proto 定义中的字段类型，注意 `uint32`/`int32`、`[]byte`/`string` 的差异

#### Common Mistake: 有返回 vs 无返回签名搞混

**Symptom**: 编译错误或 handler 调用失败

**Cause**: `.api` 定义有 `returns (Reply)` 时 Logic 应返回 `(resp, error)`，无 `returns` 时应只返回 `error`

**Fix**: 检查 `.api` 定义确认返回类型

```go
// Wrong - api 定义无 returns，但 logic 返回 (resp, error)
func (l *EndMeetingLogic) EndMeeting(req *types.EndMeetingRequest) (resp *types.EndMeetingReply, error) {
    // ...
}

// Correct - api 定义无 returns，logic 只返回 error
func (l *EndMeetingLogic) EndMeeting(req *types.EndMeetingRequest) error {
    // ...
}
```

**Prevention**: 实现 logic 前检查 `.api` 定义是否有 `returns` 关键字

### 14. Wrong vs Correct

#### Wrong

```go
// 1. 在 api 文件中定义 webhook 接口
@server (prefix: /webhook)
service livegtw {
    @handler livekitWebhook
    post /livekit (WebhookReq)
}

// 2. 在 logic 中写复杂业务逻辑（应该在 app/live 实现）
func (l *CreateMeetingLogic) CreateMeeting(req *types.CreateMeetingReq) (*types.CreateMeetingRes, error) {
    // 复杂的数据库操作、LiveKit API 调用...
}

// 3. 使用合并的 meetinglogic.go 文件（导致 gen.sh 重复定义）
// 4. api 与 gRPC 用同名类型（Req/Res 混用，导致包冲突）
// 5. 为"查我的会议"单独开 ListMyMeetings RPC
// 6. 使用 httpx.OkJsonCtx、httpx.ErrorCtx、httpx.Ok（非标准网关写法）
// 7. 类型转换错误：proto 用 uint32，HTTP 用 int32，直接赋值导致编译失败
// 8. 类型转换错误：proto 用 []byte，HTTP 用 string，直接赋值导致编译失败
```

#### Correct

```go
// 1. webhook 直接配置路由
server.AddRoute(rest.Route{
    Method:  http.MethodPost,
    Path:    "/webhook/livekit",
    Handler: webhook.LiveKitWebhookHandler(ctx),
})

// 2. logic 只做请求转发
func (l *CreateMeetingLogic) CreateMeeting(req *types.CreateMeetingRequest) (*types.CreateMeetingReply, error) {
    r, err := l.svcCtx.LiveRpcCli.CreateMeeting(l.ctx, &live.CreateMeetingReq{Title: req.Title})
    if err != nil {
        return nil, err
    }
    return &types.CreateMeetingReply{Meeting: toMeetingInfo(r.GetMeeting())}, nil
}

// 3. 使用拆分的 logic 文件，公共函数放 meeting_helper.go
// 4. api 用 Request/Reply，gRPC 用 Req/Res
// 5. 复用 ListMeetings + identity 字段，不单独开 RPC
// 6. 使用 xhttp.JsonBaseResponseCtx 统一响应格式（标准网关写法）
// 7. 正确的类型转换：uint32 vs int32
ExpireSeconds: uint32(req.ExpireSeconds)
// 8. 正确的类型转换：[]byte vs string
Payload: []byte(req.Payload)
```

## 测试页（livegtw/internal/handler/testpage）约定

### 1. Scope / Trigger

适用：维护 `/test/meeting` 的浏览器端 LiveKit 验证页面。该页面必须同时验证 HTTP 网关契约和 LiveKit client 2.x 的实时媒体、Data、RPC 能力。

### 2. Contracts

- 已登录入会调用 `POST /live/v1/live/joinMeeting`，请求体只传 `meetingNo`；身份和名称由服务端鉴权上下文决定。
- 票据入会调用免鉴权 `GET /live/v1/ticket/joinMeetingByTicket?ticket=...`，票据已绑定身份，不再从页面提交 identity/name。
- 聊天使用 LiveKit Data topic `lk.chat` 实时传输，payload 至少包含 `messageId`、`content`、`messageType`；同时调用已鉴权的 `POST /live/v1/live/reportMeetingMessage` 持久化，并从 `GET /live/v1/live/listMeetingMessages` 加载历史。
- 网关响应按 `{code, msg, data}` 解析；票据请求不发送 JWT，业务请求发送当前 JWT。

### 3. Good / Bad Cases

- Good：连接后遍历本地和远端 `trackPublications`，使用 `publication.track` 或 `TrackSubscribed` 的 track 渲染；媒体发布/取消发布事件同步 tile 和按钮状态。
- Good：聊天历史、本地回显和 Data 重复消息按 `messageId` 去重；无效 Data payload 按普通文本处理。
- Bad：只监听未来的 `TrackSubscribed`，或读取不存在的 `publication.videoTrack`，会漏掉已发布轨道和本地预览。
- Bad：复用带 JWT 的 API helper 请求 ticket join，或只依赖 Data 而不调用 report/history 接口，会分别导致票据入会失败和聊天记录缺失。

### 4. Tests Required

- JavaScript syntax check：提取内联 script 后运行 `node --check`。
- 网关验证：运行 `go test ./...`、`go vet ./...` 和 `git diff --check`。
- 浏览器集成验证：在真实 LiveKit、HTTPS/localhost 媒体权限和 Redis 环境中检查摄像头、麦克风、屏幕共享、重连、Data 聊天、HTTP 历史及票据入会；缺少环境时不得声称端到端通过。

## Web 前端（web/live）约定

### 1. Scope / Trigger

适用：React 应用 `web/live`（`@livekit/components-react` + `livekit-client` v2.x）。修改聊天、Data/RPC 调试工具或参会人身份逻辑时适用。

### 2. 三条数据链路严格解耦（核心契约）

| 链路 | 路径 | 访客行为 |
|------|------|---------|
| 群聊（业务） | `POST /reportMeetingMessage` 持久化 → `room.localParticipant.publishData(topic='lk.chat')` SDK 广播 | 跳过持久化，仅 SDK 广播 |
| Data 调试 | 仅 `POST /sendMeetingData`（服务端广播/定向），**不发** `publishData` | 管理面板不渲染，不可达 |
| RPC 调试 | 仅 `POST /performMeetingRpc`（服务端→目标参会人） | 同上 |

禁止在群聊里调用 `sendMeetingData`（它是服务端 Data 调试接口，与聊天持久化是两回事）；禁止在调试工具里混用 `publishData`。

### 3. 群聊消息 ID 契约

- 服务端 `ReportMeetingMessage` 生成并返回 `messageId`（`ReportMeetingMessageRes{MessageId}`）；持久化消息与 SDK 广播消息**必须使用同一个服务端 messageId**，历史消息加载与实时去重才有效。
- 访客无鉴权不上报，用本地 `crypto.randomUUID()` 作为 messageId。
- Wrong（ID 断裂，去重失效）：本地生成 UUID 同时用于持久化请求与 SDK 广播 → 数据库里的 ID 与广播的 ID 不同。
- Correct：`const reply = await api.reportMessage(...); messageId = reply.messageId`，再用该 messageId 组装 payload 广播。

### 4. 客户端 Echo 注册（RPC 测试前提）

- **所有参会人（含访客）入会时自动注册 `echo`**：在 `MeetingRoom` 的 `useEffect` 注册、卸载时 `unregisterRpcMethod`。注册按钮在管理面板仅房主可见，若只靠手动注册，访客目标必然报 1400。
- 注册状态（`echoRegistered`）放在 `MeetingRoom` 层级管理，不能放在 tab 挂载的子组件（如 `RealtimeTools`）——切 tab 重挂载会重置 state，UI 与实际注册状态脱节。
- Handler **必须立即返回**，返回 JSON 字符串含 `identity`/`name`/`payload`/`callerIdentity`；绝不能用未 resolve 的 Promise 等待用户输入——后端 `responseTimeoutMs` 到时直接报错，前端"卡死"观感。
- SDK 重复注册同名方法会 throw（`RPC handler already registered`），注册函数用 try/catch 包裹。

### 5. 前端健壮性检查清单（每次改动过一遍）

- [ ] 所有异步点击 handler 有 catch + notify（未处理的 rejection 静默失败）
- [ ] 提交类按钮有 loading 状态防双击（双击创建两个会议是最常见事故）
- [ ] `navigator.clipboard` 调用补 `.catch`（非 HTTPS 上下文会 reject）
- [ ] 危险操作（结束会议、移出成员、离开会议）用 `window.confirm` 确认
- [ ] 权限不足时按钮 disabled + title 提示，而不是点击后 toast 警告
- [ ] 收到的 Data payload 做字段类型校验（畸形数据不能以 undefined 作 React key）
- [ ] 启用 `noUnusedLocals`/`noUnusedParameters`，死代码在编译期报错
