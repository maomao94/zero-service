# livekitx 开放 UpdateSubscriptions/ListParticipants 透传

## Goal

在 `common/livekitx` 中以透传方式开放 LiveKit 管理 API 的 `UpdateSubscriptions` 与 `ListParticipants`，由业务侧自行判断调用时机（如演讲者强制全员订阅），不引入业务语义封装。

## Requirements

- `Client.UpdateSubscriptions(ctx, roomName, identity string, trackSids []string, subscribe bool) error`：
  仿 `MuteParticipant` 模式做参数校验后原样透传 `c.api.Room().UpdateSubscriptions`。
- `Client.ListParticipants(ctx, roomName string) ([]*livekit.ParticipantInfo, error)`：
  校验后透传 `c.api.Room().ListParticipants`，供业务获取演讲者轨道 SID。
- 内部 `muteParticipantTracks` 复用新的 `ListParticipants`，消除重复调用。
- 不新增业务语义方法；不改动 `docs/livekit-integration-guide.md`。

## Acceptance Criteria

- [ ] `UpdateSubscriptions` 校验：nil/closed client 返回 `ErrClosed`；nil ctx / 空房间 / 空身份 / 空 trackSids 返回 `ErrInvalidTokenOptions`。
- [ ] `ListParticipants` 校验：nil/closed client 返回 `ErrClosed`；nil ctx / 空房间返回 `ErrInvalidTokenOptions`。
- [ ] mock 服务端测试断言 `UpdateSubscriptions` 请求字段（room/identity/trackSids/subscribe）与 `ListParticipants` 返回。
- [ ] `go test ./common/livekitx/...` 与 `go vet ./common/livekitx/...` 通过。

## Notes

- `requestRoom`（api.go:114）已包含 `UpdateSubscriptionsRequest`，token grant 自动限定房间，无需改动认证。