# HTML 测试页

## Goal

提供单文件 HTML 测试页，覆盖会议全功能，挂载于 livegtw `/test/meeting`。前端使用 livekit-client v2.22.1（jsDelivr UMD，`LivekitClient` 全局命名空间），免构建。

## Requirements

1. 单文件 `index.html`（内嵌 CSS/JS），由 livegtw `//go:embed` 提供。
2. 功能面板（六项 + 事件日志）：
   - **连接**：LiveKit URL（默认 ws://127.0.0.1:7880）、会议号、身份、名称输入；创建会议 / 加入会议（调 livegtw API 拿 token）；`room.connect(wsUrl, token)`
   - **音视频**：发布摄像头/麦克风、本地预览、远端视频网格（`RoomEvent.TrackSubscribed` → attach）、参与者列表（v2 `room.remoteParticipants`）、离开/断开
   - **管理操作**：结束会议、踢人、音频/视频静音他人（调 livegtw API）
   - **聊天**：SDK 原生（`RoomEvent.ChatMessage`）+ 自定义 topic UserData（`RoomEvent.DataReceived` 按 topic 识别，topic=chat）
   - **Data**：`publishData(payload, {reliable:true, topic, destinationIdentities})` 广播/定向；服务端广播按钮（调 livegtw sendData）
   - **RPC**：`registerRpcMethod('echo')` + `performRpc({destinationIdentity, method, payload})` 双向
   - **屏幕共享**：`setScreenShareEnabled(true/false)`
   - **事件日志**：所有关键 RoomEvent 输出到日志面板（时间戳 + 事件名 + 摘要）
3. UI 组织参考 `livekit/components-js`（@livekit/components-react v2.9.24）的 PreJoin/VideoConference/chat 布局思路；components 为 React 库需构建链，测试页不引入，仅作参考。
4. API 调用约定：livegtw 前缀 `/live/v1/meeting/*`，响应 `{code, msg, data}`。

## Acceptance Criteria

- [x] 浏览器打开 `/test/meeting` 页面完整渲染（单文件 index.html，livekit-client v2.22.1 UMD，JS 语法校验通过、HTTP 200；媒体渲染/编解码需真实浏览器+设备，e2e 验证）
- [x] 双标签页（同一会议号，两个身份）互相看到音视频（代码路径已实现 setCameraEnabled/setMicrophoneEnabled + TrackSubscribed attach；真实媒体验证归 e2e）
- [x] 聊天双向（原生 ChatMessage + topic 双路径）、Data 广播/定向（publishData options）、RPC echo 往返（registerRpcMethod/performRpc）、屏幕共享（setScreenShareEnabled）均已实现
- [x] 管理操作（踢人/静音/结束会议）通过页面按钮生效（调 livegtw API，已接 JWT 鉴权）
- [x] 事件日志面板持续输出（RoomEvent 全量打点）
- [x] 网关 JWT 鉴权适配：请求带 `Authorization: Bearer <token>`（无 token 回退 X-User-Id 免认证）

## Notes

- 决策：components-js 是 React 组件库，纯 HTML 测试页无法直接复用（需 React 构建链）；测试页用 livekit-client UMD 手写，UI 布局参考其 PreJoin/VideoConference 样式，正式前端后续可基于 components-react
- 所有 fetch 失败/异常在页面提示，便于联调排障
- 不引入任何 npm/构建工具，保持单文件可移植
- API 核对（v2.22.1 源码确认）：`setScreenShareEnabled/setCameraEnabled/setMicrophoneEnabled` 返回 Promise、`isCameraEnabled/isMicrophoneEnabled/isScreenShareEnabled` getter、`publishData(payload, {reliable, topic, destinationIdentities})`、`registerRpcMethod`/`performRpc({destinationIdentity, method, payload})`、`ChatMessage{id,message,timestamp}`、`room.remoteParticipants`（Map by identity）
- 网关 JWT：livegtw 启用 `JwtAuth.AccessSecret`（与 gtw 一致），业务 API 需 Bearer token；webhook/测试页路由不挂 JWT 中间件