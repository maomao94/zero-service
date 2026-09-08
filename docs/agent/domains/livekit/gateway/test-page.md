# LiveKit 会议测试页

### 1. Scope / Trigger

适用：维护 `/test/meeting` 的浏览器端 LiveKit 验证页面。该页面必须同时验证 HTTP 网关契约和 LiveKit client 2.x 的实时媒体、Data、RPC 能力。

### 2. Contracts

- 已登录入会调用 `POST /live/v1/live/joinMeeting`，请求体只传 `meetingNo`；身份和名称由服务端鉴权上下文决定。
- 票据入会调用免鉴权 `GET /live/v1/ticket/joinMeetingByTicket?ticket=...`，票据已绑定身份，不再从页面提交 identity/name。
- 聊天使用 LiveKit Data topic `lk.chat` 实时传输，payload 至少包含 `messageId`、`content`、`messageType`；同时调用已鉴权的 `POST /live/v1/live/reportMeetingMessage` 持久化，并从 `GET /live/v1/live/listMeetingMessages` 加载历史。
- 网关响应按 `{code, msg, data}` 解析；票据请求不发送 JWT，业务请求发送当前 JWT。

### 3. Good / Bad Cases

- Good：连接后遍历本地和远端 `trackPublications`，使用 `publication.track` 或 `TrackSubscribed` 的 track 渲染；媒体发布/取消发布事件同步 tile 和按钮状态。
- Good：聊天历史、本地回显和 Data 重复消息按 `messageId` 去重；无效 Data payload 按普通文本处理。
- Bad：只监听未来的 `TrackSubscribed`，或读取不存在的 `publication.videoTrack`，会漏掉已发布轨道和本地预览。
- Bad：复用带 JWT 的 API helper 请求 ticket join，或只依赖 Data 而不调用 report/history 接口，会分别导致票据入会失败和聊天记录缺失。

### 4. Tests Required

- JavaScript syntax check：提取内联 script 后运行 `node --check`。
- 网关验证：运行 `go test ./...`、`go vet ./...` 和 `git diff --check`。
- 浏览器集成验证：在真实 LiveKit、HTTPS/localhost 媒体权限和 Redis 环境中检查摄像头、麦克风、屏幕共享、重连、Data 聊天、HTTP 历史及票据入会；缺少环境时不得声称端到端通过。
