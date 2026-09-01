# app/live 会议 gRPC 服务

## Goal

构建 `app/live` gRPC 服务：会议单据与参会记录的 pgsql 持久化、LiveKit 管理操作封装（基于 common/livekitx）、join token 发放、webhook 事件处理。配置风格参考 app/oryxserver（pgsql + Redis + Nacos 可选）。

## Requirements

1. `live.proto` 定义会议业务 RPC（goctl 生成骨架后手写 logic）。
2. gormx 模型：`LiveMeeting`（会议单据）、`LiveMeetingParticipant`（参会记录）。
3. 会议能力：
   - CreateMeeting(title, creatorIdentity)：生成唯一业务会议号，落库，CreateRoom（房间名=会议号）
   - JoinMeeting(meetingId, identity, name)：校验会议存在 → JoinToken（CanPublish/CanSubscribe/CanPublishData）→ 返回 token + wsUrl
   - GetMeeting / ListMeetings（分页）
   - EndMeeting：Redis lock 防重入 → DeleteRoom → 状态 ended → 参与者批量离会
   - KickParticipant / MuteParticipant / MuteParticipantVideo / ListParticipants
   - SendMeetingData（广播/定向，destinations 为空广播）/ PerformMeetingRpc
   - WebhookNotify：event ID 幂等（Redis SETNX + 本地缓存），room_started/room_finished/participant_joined/participant_left/track_* 状态同步
4. 错误语义：业务错误可判定（会议不存在、已结束、身份冲突），SDK 错误保留原始链。
5. 配置：`etc/live.yaml` 参考 oryxserver——pgsql `postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable&TimeZone=Asia/Shanghai`、Redis 36379、LiveKit 7880 devkey/secret、ListenOn 0.0.0.0:21017。

## Acceptance Criteria

- [x] `go build ./app/live/...` + `go vet ./app/live/...` 通过
- [x] 模型迁移（AutoMigrate）在 pgsql 上可执行；表结构与 trigger plan/plan_batch 完全一致（LegacyStringBaseModel + VersionMixin + CreateUser/UpdateUser/DeptCode + sql.Null* + int status）
- [x] 单测全绿（sqlite 内存库 + miniredis + fake LiveKit，`go test -race`）：会议状态流转、webhook 事件处理与重放一致性、参数校验、锁竞争（真实 RedisLock + miniredis）、非法数据拒绝、IdUtil 会议号格式（M+18位）、CreateUser/UpdateUser/DeptCode 落库断言
- [x] 真实链路冒烟（dev server + pgsql + redis）：IdUtil 会议号、JoinToken/wsUrl、参与者落库（create_user/update_user=操作人、dept_code 落库）、webhook 原始字节链路（participant_joined + 重放不报错）、SendData、结束会议（carbon 时间格式 + update_user 更新）、幂等结束、已结束拒绝加入、NotFound
- [x] EndMeeting 用 go-zero RedisLock（`redis.NewRedisLock`，写法对齐 oryxserver relay Store.Lock）
- [x] webhook 处理：原始 proto 字节 → `livekit.WebhookEvent` SDK 对象解析；已知事件全 case（处理或 TODO 标注）；不做 TTL 幂等（处理操作本身幂等，补发/迟到事件安全重放闭环）
- [x] 创建数据时 CreateUser + UpdateUser 同时赋值（项目惯例，对齐 trigger；不依赖 gormx UserContext hook）
- [x] 用户身份统一 `authctx.GetUserId`（不用已废弃的 `tool.GetCurrentUserId`）
- [x] pgsql 落库验证 + redis key 前缀（`live:outId_M` / `live:lock:meeting:*`）

## Notes

- 服务端不 JoinRoom（不做实时媒体连接），livekitx JoinRoom 能力留给后续 Agent/机器人场景
- 会议号用 `tool.IdUtil.NextId("M", "live")`（Redis 序号，category=live 与其他业务隔离）
- JoinToken validFor 默认 2h，可配置
- Redis key 统一 `live:` 业务前缀（最前段系统 key 由 redis 配置自动附加）
- LiveKit 客户端经 `httpc.NewServiceWithClient("httpc-livekit", ...)` 注入 go-zero httpc.Service
- 错误码统一 extproto（tool.NewErrorByPbCode/Wrap）：参数 101101、记录不存在 102102、业务状态 105102、重复操作 105103、缓存 103101、第三方 106102、DB 102101
- proto 风格：snake_case 字段 + json_name 驼峰 + 中文注释 + 时间字段注释"格式：yyyy-MM-dd HH:mm:ss"；RPC 返回统一 XxxRes 包一层
- webhook 链路：livegtw 验签后把 livekit.WebhookEvent proto 字节透传给 WebhookNotify
- **遗留**：其他 app/* 服务（trigger 等 14 处）仍用废弃的 `tool.GetCurrentUserId`，属存量代码，后续单独任务统一迁移