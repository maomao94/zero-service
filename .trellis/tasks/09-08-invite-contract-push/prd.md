# 新增 Live 通知入会 gRPC 契约与推送链路

## Goal
为主持人提供按 `user_id` 通知已登录参会人入会的 Live gRPC 能力。

## Scope
- 修改 `app/live/live.proto`，新增成对 Req/Res RPC，会议号沿用 `meeting_no` / `meeting_code` 二选一。
- `app/live` 校验调用者身份、会议状态和目标 ID，并通过已有 `socketpush.SocketPushClient.BroadcastRoom` 推送固定事件。
- 在 `app/live` ServiceContext 装配 socketpush client，补充生成代码和 fake client 影响的测试。
- 推送房间名为目标 `user_id`；payload 使用 protojson/结构化 JSON，包含会议标识、展示信息和邀请上下文，不包含任何访问 token。

## Dependencies
无。完成后为 `livegtw-user-auth` 和前端通知中心提供稳定契约。

## Acceptance Criteria
- [ ] 合法请求向目标 user room 提交固定事件，推送失败被正确返回/包装。
- [ ] 空目标 ID、会议不存在、会议结束、调用者无身份均被拒绝。
- [ ] 生成代码、`go test ./app/live/...` 和 `git diff --check` 通过。
