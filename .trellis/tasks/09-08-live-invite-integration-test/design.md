# 联调测试与跨服务验收设计

## Purpose

本任务不新增业务功能。它在前四个子任务完成后验证邀请通知与 SIP 入会体验的跨层契约、失败语义和可部署性。

## System Boundaries

```
livegtw HTTP -> live gRPC -> socketpush gRPC -> socketgtw Socket.IO -> web/live
                                      |
                                      +-> user_id room

web/live -> livegtw HTTP -> live gRPC -> LiveKit / SIP
```

- `app/live` 是邀请业务、会议状态和 socketpush 调用的写入/编排所有者。
- `socketpush` 只确认向所有 socketgtw 实例提交广播，不确认目标用户在线或浏览器已处理。
- `socketgtw` 从 JWT claims 提取 `user_id` 元数据，并把浏览器会话加入个人房间。
- `web/live` 维护临时通知 UI；接受邀请时必须重新通过 `livegtw` 获取 LiveKit join token。
- `livegtw` 负责验证用户 JWT 和 `user_id`，票据与 webhook 路由保持在该边界之外。

## Verification Layers

| Layer | Evidence | Required assertions |
| --- | --- | --- |
| Contract | `live.proto`、生成代码、调用方搜索 | RPC、字段 JSON 名、固定 event/payload 与所有消费者一致 |
| Live unit | `app/live` Logic tests with fake socketpush client | 成功调用、输入/会议状态拒绝、socketpush error、payload 不含 token |
| Gateway unit | `app/livegtw` middleware/handler tests | 缺少 user_id 在调用 gRPC 前拒绝；合法 claim 通过；票据/webhook 不受影响 |
| Socket integration | socketgtw + authenticated Socket.IO client | `user_id` 元数据、个人 room、固定事件和 payload 可接收 |
| Frontend | TypeScript build plus browser smoke test | 连接、通知展示、忽略、接受、重复/失效和断线状态 |
| SIP smoke | configured LiveKit/SIP environment | 独立拨号返回 S 会议、浏览器自动进入、电话参与者随后出现 |

## Test Matrix

| Scenario | Expected result | Environment |
| --- | --- | --- |
| Host invites online target | API success means push submitted; target receives one notification and can join | live, socketpush, socketgtw, browser, Redis, DB |
| Target offline | API remains submission-success if socketpush accepts; no false delivery claim | live, socketpush |
| socketpush unavailable/error | invite API exposes a mapped failure; browser has no new notification | live + fake/disabled socketpush |
| Empty/missing target user ID | Live rejects request before push | live unit |
| Meeting missing or ended | Live rejects request before push | live unit |
| JWT lacks user_id | livegtw rejects before gRPC invocation | livegtw unit/HTTP |
| Ticket and webhook routes | preserve existing access behavior | livegtw HTTP |
| Duplicate notification | frontend renders one actionable notification | browser |
| Accepted ended invitation | frontend shows join failure and keeps UI usable | browser + livegtw |
| Independent SIP dial | returned S meeting is joined by browser; phone participant can later appear | LiveKit + SIP provider |

## Environment And Rollback

- Unit and compile checks are mandatory and run without external LiveKit, socketgtw or SIP availability.
- Socket/browser and SIP checks require configured secrets, Redis, PostgreSQL, LiveKit and a reachable SIP provider; missing prerequisites are recorded as unexecuted, never reported as passed.
- No schema or data migration is expected. Rollback is limited to reverting the four prerequisite feature tasks; this test task should not introduce production behavior.
