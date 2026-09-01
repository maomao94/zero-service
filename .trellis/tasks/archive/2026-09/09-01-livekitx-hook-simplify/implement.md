# 简化 livekitx Hook 设计并补充场景字段文档 — Implement（修订版）

## 顺序清单

1. 删除：`hooks.go`、`events.go`、`data.go`、`store.go`；`errors.go` 删除 `HookPanicError`。
2. `config.go`：删除 `WithStore`、`Config.store`、`Client.stateMu/store/realtime` 字段；`Close` 仅幂等标记关闭。
3. `realtime.go` 改造为 `room.go`：`JoinRoom`/`CreateRoom`/`CreateAndJoinRoom`/`DeleteRoom`（签名见 design.md），`ChatTopic` 常量移入；删除 `RealtimeRoom`/`Connect`。
4. `webhook.go`：导出 `NewWebhookKeyProvider(signingKey string) auth.KeyProvider`；删除 `OnWebhook`/`ReceiveWebhook`/`WebhookEvent`。
5. 测试改造：删 `hooks_test.go`/`events_test.go`/`store_test.go`（或桥接/Store 依赖部分）；`example_test.go` 重写为 Room API + RoomCallback 完整示例；新增 `room_test.go`（校验分支 + CreateRoom/DeleteRoom Twirp mock + callback 透传）；`integration_test.go` 改原生回调断言（入会/聊天/RPC/断开原因）；`webhook_test.go` 改验真/验假；`config_test.go` 删 WithStore 用例。
6. `common/livekitx/README.md`：删 16 Hook 章节与 Store 描述，改 Room API + 原生回调指引，链接新文档。
7. 新建 `docs/livekit-callbacks-guide.md`（场景清单见 design.md，字段以 /Users/hehanpeng/GolandProjects/server-sdk-go 锁定源码核对）。
8. 不更新 `.trellis/spec/backend/livekit-guidelines.md`（update-spec 阶段由主会话处理）。

## 验证命令

```bash
gofmt -l common/livekitx docs 2>/dev/null; git diff --check
go test ./common/livekitx/...
go test -race ./common/livekitx/...
go vet ./common/livekitx/...
go build ./...
LIVEKITX_INTEGRATION=1 go test ./common/livekitx -run TestLiveKitDevServer -v
go test ./...
```

## 风险点

- `room.go` 是核心：`JoinRoom` 必须原样透传 callback（无 Merge/无桥接）；`CreateAndJoinRoom` 加入失败不自动删房间。
- 集成测试改原生回调后，聊天双路径识别（`*livekit.ChatMessage` vs `UserDataPacket`+`ChatTopic`）由测试内业务代码判断，文档必须写清两条路径字段差异。
- 文档字段说明必须以锁定 SDK v2.18.1 源码核对（`RoomCallback` 定义在 room.go，`DataReceiveParams` 在 data.go），禁止猜测。
- `Client.Close` 简化后不得残留对已删除字段的引用（编译期可发现）。

## 收尾检查

- 包内无残留：`eventDispatcher`/`dispatch(`/`OnChatMessage(`/`OnWebhook(`/`HookPanicError`/`RoomConnectionEvent`/`ConnectionState`/`RealtimeRoom`/`WithStore`（`ChatTopic` 除外）。
- README 无 16 Hook/Store 残留描述。
- PRD 验收标准 AC1-AC7 逐条过一遍。