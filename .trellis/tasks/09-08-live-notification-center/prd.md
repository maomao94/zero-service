# 前端 Socket 通知中心与邀请入会

## Goal
让在线 token 用户通过 Socket.IO 收到个人会议邀请，并在通知中心接受后进入会议。

## Scope
- 为 `web/live` 建立 socketgtw 连接，携带现有 JWT；连接成功后加入以 `user_id` 命名的个人通知房间。
- 监听后端固定邀请事件，维护未读数、列表、忽略/关闭和连接状态。
- 接受邀请调用已鉴权 join API，不信任通知 payload 中的 LiveKit token。
- 在大厅/顶栏提供通知入口，适配移动端；重复事件去重，会议结束/失效有提示。

## Dependencies
依赖 `invite-contract-push` 的事件/payload/API 契约和 `livegtw-user-auth` 的身份语义。

## Acceptance Criteria
- [ ] 登录后在线用户能连接 socketgtw 并进入 user_id 房间。
- [ ] 收到邀请后通知中心展示可读信息，接受后成功进入会议。
- [ ] 断线、鉴权失败、重复、无效或已结束邀请有稳定 UI 行为。
- [ ] 不把访问 token 渲染或写入通知 payload；`npm run build` 通过。
