# LiveKit 会议业务服务

适用：`app/live` gRPC 会议服务。机制层约定见[公共库契约](./common.md)，HTTP 转发见[网关契约](./gateway/service.md)。

### 1. Webhook 链路（验签 → 原始字节 → SDK 对象处理）

- 链路：LiveKit 推送 → livegtw 用 `webhook.ReceiveWebhookEvent` 验签（失败 401，不调用业务）→ 把 `*livekit.WebhookEvent` **proto 序列化后的原始字节**经 `WebhookNotify(WebhookNotifyReq{Data: bytes})` 透传给 live 服务 → live 服务 `proto.Unmarshal` 解析为 SDK 对象后处理业务。禁止在 proto 中做字段扁平化（eventId/eventType/roomName/...），会丢失 SDK 结构。
- **不做 TTL 幂等**：LiveKit 事件可能重复、迟到、补发（对账闭环需要重放），而处理操作本身幂等（参会记录 `Where+Assign+FirstOrCreate` upsert、会议状态仅 active→ended 流转、未知会议安全忽略），重复/补发重放结果一致。禁止再加 event ID 去重键阻碍补发。
- **事件全 case**：`room_finished`/`participant_joined`/`participant_left` 处理；`room_started` 按[SIP 补插会议契约](./sip/service.md)处理外呼或 API 自动创建的房间；`participant_connection_aborted`/`track_published`/`track_unpublished`/`egress_started`/`egress_updated`/`egress_ended`/`ingress_started`/`ingress_ended` 建 case 标注 TODO 并记日志；未知事件安全忽略。禁止 default 静默吞掉已知事件。

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

新业务表（会议等）按 `app/trigger/model/gormmodel` 的 plan 系列风格：`gormx.LegacyStringBaseModel`（string 主键 + create_time/update_time + is_deleted 软删）+ `CreateUser`/`UpdateUser`/`DeptCode`（sql.NullString）+ 可空字段用 `sql.NullString`/`sql.NullTime` + `int` 状态 + 索引名 `idx_<表名>_<字段>`。

VersionMixin 使用规则见[数据访问规范](../../standards/data.md)。

### 4. 业务错误码

业务错误统一 extproto + `tool.NewErrorByPbCode`/`NewErrorByPbCodeWrap`（reason=六位错误码，HTTP 自动映射），禁止裸 `status.Error`/`status.Errorf`。常用映射：参数 `101101`、记录不存在 `102102`、记录已存在 `102103`、缓存/Redis `103101`、未认证 `104101`、业务状态不允许 `105102`、重复操作 `105103`、DB `102101`、第三方（LiveKit）`106102`。
