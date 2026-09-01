# app/live 会议 gRPC 服务 — 技术设计

## 目录结构（参考 app/oryxserver）

```
app/live/
├── live.go                 # main：启动 gRPC server
├── live.proto              # RPC 契约
├── live/                   # goctl 生成的 pb 代码
├── gen.sh                  # goctl 生成脚本
├── etc/live.yaml           # 配置（参考 oryxserver：pgsql+redis）
├── internal/
│   ├── config/config.go    # Config：RpcServerConf + LiveKit + DB + Redis
│   ├── server/liveserver.go
│   ├── svc/servicecontext.go  # 依赖注入：livekitx.Client、gorm DB、redis
│   ├── logic/*.go          # 业务 logic（goctl 骨架 + 手写）
│   └── model/gormmodel/    # gormx 模型（参考 oryxserver 的 model 组织）
```

## 数据模型（pgsql）

```sql
live_meetings:
  id            BIGSERIAL PK
  meeting_no    VARCHAR(32) UNIQUE   -- 业务会议号 = LiveKit 房间名
  title         VARCHAR(128)
  status        SMALLINT             -- 1 created 2 active 3 ended（创建后即 active）
  creator_identity VARCHAR(64)
  started_at    TIMESTAMPTZ
  ended_at      TIMESTAMPTZ NULL
  created_at / updated_at

live_meeting_participants:
  id         BIGSERIAL PK
  meeting_no VARCHAR(32)   -- 关联会议（冗余房间名，便于 webhook 反查）
  identity   VARCHAR(64)
  name       VARCHAR(64)
  status     SMALLINT      -- 1 joined 2 left
  joined_at  TIMESTAMPTZ
  left_at    TIMESTAMPTZ NULL
  UNIQUE(meeting_no, identity)
```

模型实现：参考 app/oryxserver/model 的 gorm 模型写法 + common/gormx 基建（TenantMixin 不需要，会议业务无租户；用 gorm.Model 风格自定时间戳）。

## 服务状态机

```
created →(CreateRoom 成功)→ active →(EndMeeting/DeleteRoom 或 webhook room_finished)→ ended
```
- 创建即 active（LiveKit 房间无"未开始"概念，房间存在即可用）
- 结束幂等：redis lock + 状态判断（仅 active→ended）

## RPC 契约（live.proto）

```proto
service LiveRpc {
  CreateMeeting(req) → MeetingResp
  JoinMeeting(req) → JoinMeetingResp {token, wsUrl}
  GetMeeting(req) → MeetingResp
  ListMeetings(req) → ListMeetingsResp   // 分页
  EndMeeting(req) → Empty
  KickParticipant(req) → Empty
  MuteParticipant(req) → Empty           // audio 或 video 由 type 区分
  ListParticipants(req) → ListParticipantsResp
  SendMeetingData(req) → Empty           // topic+payload+destinations(空=广播)
  PerformMeetingRpc(req) → RpcResp
  WebhookNotify(req) → Empty             // 由 livegtw 验签后转发
}
```

webhook 事件在服务内用 `livekit.WebhookEvent` 的 Event/Id/Room/Participant 字段解析（proto 直接引用 github.com/livekit/protocol/livekit 类型会侵入服务契约，故 WebhookNotify 用自有的扁平结构：eventId/eventType/roomName/identity/participantName/participantMetadata/track 信息）。

## Webhook 幂等

- 入口：Redis `SETNX idem:live:webhook:<eventId> 1 EX 86400`；已存在则直接返回成功（幂等）
- 失败场景：Redis 不可用时降级为内存 map（进程内去重，MVP 可接受）
- 处理逻辑：
  - `room_finished` → 状态 ended（若 DeleteRoom 已处理则跳过）
  - `participant_joined` → 参会记录 upsert（status joined）
  - `participant_left` → 参会记录更新（status left + left_at）
  - 其他事件（track_*、room_started、egress 等）→ 记录日志忽略

## 依赖注入（svc）

```go
type ServiceContext struct {
  Config  config.Config
  LiveKit *livekitx.Client   // New(WithURL/WithAPIKey/WithHTTPClient)
  DB      *gorm.DB           // gormx 打开 pgsql
  Redis   redis.UniversalClient
  MeetingStore MeetingStore  // 封装 LiveMeeting CRUD
  ...
}
```

## 错误约定

- 业务错误返回 `status.Error(codes.NotFound/AlreadyExists/FailedPrecondition/Internal, msg)`，文案中文
- livekitx 错误保留原始链（errors.As 可判定 SDK ServerError）
- 参数校验：meetingNo/identity 空 → InvalidArgument

## 配置（etc/live.yaml，参考 oryxserver）

```yaml
Name: live.rpc
ListenOn: 0.0.0.0:21017
Mode: dev
Log: ...
LiveKit:
  Url: http://127.0.0.1:7880
  ApiKey: devkey
  ApiSecret: secret
  WebhookKey: secret
  TokenValidFor: 2h
DB:
  DataSource: postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable&TimeZone=Asia/Shanghai
Redis: 127.0.0.1:36379 / G62m50oigInC30sf
```

## 测试策略

- 单测：logic 用 gormx 测试基建（sqlite 或 postgres 测试库）+ redis mock；webhook 幂等用内存版 store 接口
- 集成：`LIVEKITX_INTEGRATION=1` 已有机制层覆盖；本服务集成放 e2e 任务
- 模型迁移：启动时 AutoMigrate（与 oryxserver 一致）