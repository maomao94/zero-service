# 通知参会人入会与会议电话体验

## Goal

会议主持人可以按用户 ID 邀请一个已登录的 token 用户；目标用户在浏览器收到通知后进入指定会议。电话外呼可以在拨号后直接进入对应会议大厅，等待电话用户加入，且测试电话工具不再占据过长的横向空间。

## Background And Confirmed Facts

- `app/live/live.proto` 已有会议、票据和 `DialSip` RPC，但没有“通知用户入会”RPC。
- `socketapp/socketpush/socketpush.proto` 已有 `BroadcastRoom`，`socketapp/socketgtw` 会将 Socket.IO 连接 token claims 中配置的元数据复制到 session；当前 `SocketMetaData` 已包含 `user_id`。
- 当前 socket 推送链路是 `app -> socketpush -> socketgtw -> browser`，适合用 `user_id` 作为个人通知房间名。
- `livegtw` 的 `MeetingAuthMiddleware` 当前只桥接 JWT claims，不校验 `user_id` 是否存在；缺少身份的请求会继续进入 handler。
- `web/live` 当前已支持会议、票据、SIP 供应商和拨号；测试电话在大厅独占一整行，且独立拨号不会自动进入返回的会议。
- 前端未安装 `socket.io-client`，需要在通知子任务中确定依赖或使用项目已有的 Socket.IO 客户端能力。

## Requirements

### R1. 通知入会接口
- 新增 Live gRPC 接口，输入至少包含目标 `user_id`、会议标识（沿用 `meeting_no` / `meeting_code` 二选一）及邀请展示所需信息。
- 服务端校验调用者身份、会议存在且可操作、目标用户 ID 非空；按 `user_id` 作为通知事件/个人房间目标。
- 使用 `socketapp/socketpush/socketpush.proto` 的现有推送能力发送固定事件和结构化 JSON payload；发送结果明确为“已提交推送”，不虚报目标浏览器已收到。
- live 与 socketgtw 使用同一套鉴权 token secret/claim 语义；不得把目标用户的 token 返回给调用者或记录到日志。

### R2. live_gtw 身份校验
- 所有需要 `MeetingAuth` 的业务路由，在进入 handler 前必须确认上下文存在非空 `user_id`。
- 缺少或无法桥接 `user_id` 时拒绝访问，不调用下游 gRPC；保持现有统一错误响应格式，并覆盖 HTTP/gateway 可观察行为。
- webhook、票据免鉴权路由不受该校验误伤。

### R3. 电话进入会议大厅
- 独立 SIP 外呼仍由服务端创建 S 前缀会议，但前端收到拨号结果后应能直接进入该会议大厅/会议页面，等待电话用户加入。
- 已传入会议号的外呼继续使用现有会议，不创建重复会议。
- 测试电话入口改为紧凑 tag/横向短控件，不再默认铺满一整行；桌面和移动端均可操作。

### R4. 前端通知中心与按用户 ID 邀请
- 登录用户建立 Socket.IO 长连接，携带与业务一致的鉴权 token，并以 `user_id` 加入个人通知房间。
- 收到邀请事件后进入通知中心，显示会议标题、会议号、邀请人/时间和状态；用户可接受进入会议或忽略/关闭。
- 通知 payload 不包含敏感 token；接受后通过已鉴权的会议入会接口获取 LiveKit token。
- 连接断开、认证失败、重复通知、过期邀请和会议已结束必须有明确 UI 状态，不阻塞普通会议使用。

## Out Of Scope

- 不新增 socketpush/socketgtw 的重复按用户元数据推送 RPC；除非实现验证后发现现有 `BroadcastRoom` 无法满足个人房间语义。
- 不把通知持久化为服务端离线消息；本期仅覆盖在线浏览器实时通知。
- 不改变 LiveKit 会议内成员权限模型、票据访客模型或 SIP 媒体链路。
- 不在本期实现电话接通/挂断状态的完整业务持久化与通知。

## Acceptance Criteria
- [ ] 合法主持人调用通知接口后，目标用户在线浏览器收到固定事件，payload 可直接展示并接受入会。
- [ ] 目标用户不在线、Socket 服务不可用或推送调用失败时，接口返回可区分的提交/失败结果，且不泄露 token。
- [ ] 缺少 `user_id` 的 JWT 请求被 `livegtw` 拒绝；合法 JWT、票据加入和 LiveKit webhook 行为保持正确。
- [ ] 独立拨号成功后前端进入服务返回的 S 会议；电话用户稍后加入时能在成员/媒体区域看到。
- [ ] 测试电话在桌面和移动端以紧凑控件呈现，会议创建/加入和供应商管理布局不回归。
- [ ] 通知中心可接收、展示、忽略和接受邀请；接受后成功进入会议或展示可理解的失败原因。
- [ ] 协议源、生成代码、后端测试、前端类型检查和构建通过，`git diff --check` 通过。

## Execution Order

1. `09-08-invite-contract-push`：Live 通知 RPC、socketpush client 装配与推送语义。
2. `09-08-livegtw-user-auth`：live_gtw `user_id` 强制鉴权及回归测试；依赖 1 的请求身份语义确认。
3. `09-08-sip-call-conference-lobby`：拨号返回会议后的前端进入会议与紧凑测试电话布局。
4. `09-08-live-notification-center`：Socket.IO 连接、个人房间、通知中心和邀请入会 UI；依赖 1、2 的事件/API 契约。
5. `09-08-live-invite-integration-test`：跨服务和浏览器联调验收。

## Blocking Questions

- 无。通知采用在线实时投递；离线用户不接收历史通知，后续如需离线可靠性另建任务。
