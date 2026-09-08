# Live 通知入会设计

## Boundary

`app/live` 负责会议和邀请业务校验，并调用已有 `socketpush.SocketPushClient`。`socketpush` 负责向所有 socketgtw 节点提交广播；socketgtw 负责把消息发送到以目标 `user_id` 命名的房间。通知接口只表示推送提交结果，不保证浏览器在线或已处理。

## Contract

- 在 `app/live/live.proto` 新增一个 `NotifyMeetingParticipant` 成对 Req/Res RPC。
- 请求包含目标 `user_id`、`meeting_no` / `meeting_code`、可选目标展示名和通知上下文；会议标识遵循现有二选一规则。
- 响应只返回请求关联 ID或空响应，不返回 LiveKit token。
- 固定事件名和 payload 字段在 `app/live` 中集中定义；payload 使用结构化 JSON，至少包含会议号、会议码、标题、邀请人和通知时间。
- 目标房间直接使用经过 trim 且非空的 `user_id`。

## Authorization And Failure

- 从 gRPC metadata 使用 `authctx.GetUserId` 获取邀请人；没有调用者身份时返回现有未认证业务错误。
- 会议不存在或已结束时不调用 socketpush。
- socketpush transport/business error 原样保留 cause，并映射为现有第三方/外部服务错误码；不能吞错后返回成功。
- 发送成功只记录非敏感的会议号、目标 user_id 和关联 ID，不记录 token 或完整 payload。

## Assembly And Compatibility

- `app/live/internal/svc.ServiceContext` 增加可选 `socketpush.SocketPushClient`，由 `SocketPushConf` 配置装配；未配置时服务仍可启动，但通知请求明确失败。
- 使用 `grpcx.UnaryMetadataInterceptor` 透传调用身份，与现有跨服务 client 装配一致。
- 修改 proto 后仅通过 `app/live/gen.sh` 生成代码；同步检查 `LiveRpcClient` fake 和所有实现。
- 不新增 socketpush/socketgtw RPC，不改变已有 Socket.IO 事件协议。
