# 简化 livekitx Hook 设计并补充场景字段文档

## Goal

拆除 `common/livekitx` 过度设计的 typed Hook 分发层（16 个 Hook + `eventDispatcher` + 回调桥接 + Data 包解析）与自定义连接状态管理（Store/ConnectionState），改为 **Room API + SDK 原生回调**：业务通过 livekitx 完成加入房间、创建房间、创建并加入、删除房间，所有实时事件直接用 SDK 原生 `RoomCallback` 处理，Webhook 只导出验签 KeyProvider；同时产出场景+字段文档与可作参考的示例/测试。

## Background

- 09-01 会话用户反馈：typed Hook（注册顺序快照/错误聚合/panic 恢复/关闭语义）"没必要再包一层"；回调桥 + `bridgeDataPacket` "做太深了"；Webhook 分发"没必要定义那么复杂"；"直接复用 sdk 钩子定义"并"做一个文档，写好每个场景，字段是啥意思"。
- 追加方向（用户 09-01 第二轮）：`ConnectionStatus`/`ConnectionState` 与 `store Store` **全部砍掉**——"应该从 connect 的 callback 来做，不是按照自己定义，sdk 本身定义了一套 callback"、"业务侧自己注册 callback 业务逻辑"。
- 追加方向（用户 09-01 第三轮）：`RealtimeRoom` 设计不对——"你应该设计的是一个 room 的 api：加入房间、创建房间、创建并加入房间、删除房间等"。
- 追加要求：补充 example RoomCallback 测试，全部测试保留作为未来参考。
- 已确认：文档形态为**独立文档 `docs/livekit-callbacks-guide.md`**（README 加链接）；无现有调用方（`app/meeting` 未开发），删除 API 无需迁移垫片。

## Requirements

- R1 删除 typed Hook 分发层：`eventDispatcher`、`Handler`/`Subscription`、`hookSet`、16 个 `OnXxx` 注册 API、全部 typed 事件类型（`hooks.go`/`events.go` 整文件删除）。
- R2 删除 Store 与连接状态：`store.go` 整文件、`ConnectionStatus`/`ConnectionState`、`WithStore` option、`Client` 的 `stateMu`/`store`/`realtime` 字段、`RealtimeRoom` 类型。
- R3 重写为 Room API（`realtime.go` 改造为 `room` API）：
  - `JoinRoom(ctx, token, callback, opts...) (*lksdk.Room, error)`：加入已有房间，callback 为 SDK 原生 `*lksdk.RoomCallback`（nil 允许、原样透传，无桥接无合成事件）；返回 SDK 原生 `*lksdk.Room`（业务自行 `room.Disconnect()`、`room.LocalParticipant.PerformRpc` 等）。
  - `CreateRoom(ctx, name, opts...) (*livekit.Room, error)`：经管理 API 创建房间。
  - `CreateAndJoinRoom(ctx, name, token, callback, opts...) (*lksdk.Room, error)`：创建房间后使用业务签发的 token 加入；连接失败不自动删房间（可能已有他人入会），文档说明。
  - `DeleteRoom(ctx, name) error`：经管理 API 删除房间。
  - `Connect` 名称废弃（保留吗？否——按用户要求改为 JoinRoom 语义）。
- R4 Webhook：删除 `OnWebhook`/`ReceiveWebhook`/`WebhookEvent`；导出 `NewWebhookKeyProvider(signingKey string) auth.KeyProvider`，业务自行 `webhook.ReceiveWebhookEvent(req, livekitx.NewWebhookKeyProvider(key))`。
- R5 保留基础设施：`New`/`Close`/管理 API 五服务/`JoinToken`/`SIPToken`/`ChatTopic = "chat"` 常量/`ErrClosed`/`ErrInvalidConfig`/`ErrInvalidTokenOptions`；删除 `HookPanicError`。
- R6 新增 `docs/livekit-callbacks-guide.md`：覆盖全部 `RoomCallback` 场景（以锁定 SDK v2.18.1 的 `RoomCallback`/`ParticipantCallback` 定义为基准逐一核对），每场景列回调签名、字段类型与含义、触发来源、典型用途、Go 示例；另含聊天双路径识别（`*livekit.ChatMessage` vs `UserDataPacket`+`ChatTopic`）、RPC 收发（`RegisterRpcCtxMethod`/`PerformRpc`/`UnsupportedMethod` 行为）、Webhook 接收（验签/幂等）三节。
- R7 `common/livekitx/README.md`：删除 16 Hook 章节与 Store 描述，改为 Room API + 原生回调指引并链接新文档。
- R8 测试改造：删除依赖 typed Hook/Store 的测试；新增 `example_test.go`（Room API + RoomCallback 完整示例，可作未来参考）；集成测试用原生回调断言（入会/聊天/RPC 往返）；webhook 测试改验签验真/验假；全部测试保留作为未来参考。
- R9 更新 `.trellis/spec/backend/livekit-guidelines.md`：Scenario/Hook/Store 契约改为 Room API + 原生回调版本。

## Acceptance Criteria

- [ ] AC1 包内无残留：grep 无 `eventDispatcher`/`dispatch(`/`OnChatMessage(`/`OnWebhook(`/`HookPanicError`/`RoomConnectionEvent`/`ConnectionState`/`RealtimeRoom`/`WithStore`/`store` 引用（`ChatTopic` 常量除外）
- [ ] AC2 Room API 存在：`JoinRoom`/`CreateRoom`/`CreateAndJoinRoom`/`DeleteRoom`，Join 类返回 SDK 原生 `*lksdk.Room`，callback 原样透传（单测验证自定义 callback 收到真实回调）
- [ ] AC3 Webhook 验签：正确 signing key 通过、错误 key 返回签名错误（单测）；业务侧调用方式与文档示例一致
- [ ] AC4 集成测试（`LIVEKITX_INTEGRATION=1`，本地 dev server）用原生回调断言入会/聊天/RPC 往返，全部通过
- [ ] AC5 `gofmt`/`go test ./common/livekitx/...`/`go test -race`/`go vet`/`go build ./...`/`go test ./...` 全部通过
- [ ] AC6 `docs/livekit-callbacks-guide.md` 存在且覆盖 R6 全部场景与字段（以 SDK 源码核对）；`example_test.go` 存在可作参考；README 已链接且无 16 Hook/Store 残留描述
- [ ] AC7 `livekit-guidelines.md` Scenario/Hook/Store 契约已更新为 Room API + 原生回调版本

## Out of Scope

- 管理 API（`api.go` Twirp client 与注入）、Token 实现细节重设计（保持现状）
- 新增任何新的 Hook/回调注册机制或连接状态持久化（本次只删不改）
- `app/meeting` 业务逻辑
- 房间内踢人/租约/选主等分布式协调（业务自行基于管理 API 实现）