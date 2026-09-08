# LiveKit Web 前端

### 1. Scope / Trigger

适用：React 应用 `web/live`（`@livekit/components-react` + `livekit-client` v2.x）。修改聊天、Data/RPC 调试工具或参会人身份逻辑时适用。

### 2. 三条数据链路严格解耦（核心契约）

| 链路 | 路径 | 访客行为 |
|------|------|---------|
| 群聊（业务） | `POST /reportMeetingMessage` 持久化 → `room.localParticipant.publishData(topic='lk.chat')` SDK 广播 | 跳过持久化，仅 SDK 广播 |
| Data 调试 | 仅 `POST /sendMeetingData`（服务端广播/定向），**不发** `publishData` | 管理面板不渲染，不可达 |
| RPC 调试 | 仅 `POST /performMeetingRpc`（服务端→目标参会人） | 同上 |

禁止在群聊里调用 `sendMeetingData`（它是服务端 Data 调试接口，与聊天持久化是两回事）；禁止在调试工具里混用 `publishData`。

### 3. 群聊消息 ID 契约

- 服务端 `ReportMeetingMessage` 生成并返回 `messageId`（`ReportMeetingMessageRes{MessageId}`）；持久化消息与 SDK 广播消息**必须使用同一个服务端 messageId**，历史消息加载与实时去重才有效。
- 访客无鉴权不上报，用本地 `crypto.randomUUID()` 作为 messageId。
- Wrong（ID 断裂，去重失效）：本地生成 UUID 同时用于持久化请求与 SDK 广播 → 数据库里的 ID 与广播的 ID 不同。
- Correct：`const reply = await api.reportMessage(...); messageId = reply.messageId`，再用该 messageId 组装 payload 广播。

### 4. 客户端 Echo 注册（RPC 测试前提）

- **所有参会人（含访客）入会时自动注册 `echo`**：在 `MeetingRoom` 的 `useEffect` 注册、卸载时 `unregisterRpcMethod`。注册按钮在管理面板仅房主可见，若只靠手动注册，访客目标必然报 1400。
- 注册状态（`echoRegistered`）放在 `MeetingRoom` 层级管理，不能放在 tab 挂载的子组件（如 `RealtimeTools`）——切 tab 重挂载会重置 state，UI 与实际注册状态脱节。
- Handler **必须立即返回**，返回 JSON 字符串含 `identity`/`name`/`payload`/`callerIdentity`；绝不能用未 resolve 的 Promise 等待用户输入——后端 `responseTimeoutMs` 到时直接报错，前端"卡死"观感。
- SDK 重复注册同名方法会 throw（`RPC handler already registered`），注册函数用 try/catch 包裹。

### 5. 前端健壮性检查清单（每次改动过一遍）

- [ ] 所有异步点击 handler 有 catch + notify（未处理的 rejection 静默失败）
- [ ] 提交类按钮有 loading 状态防双击（双击创建两个会议是最常见事故）
- [ ] `navigator.clipboard` 调用补 `.catch`（非 HTTPS 上下文会 reject）
- [ ] 危险操作（结束会议、移出成员、离开会议）用 `window.confirm` 确认
- [ ] 权限不足时按钮 disabled + title 提示，而不是点击后 toast 警告
- [ ] 收到的 Data payload 做字段类型校验（畸形数据不能以 undefined 作 React key）
- [ ] 启用 `noUnusedLocals`/`noUnusedParameters`，死代码在编译期报错

### 6. 浏览器 LiveKit 地址

- 浏览器连接 LiveKit 使用独立的媒体地址，禁止把 `VITE_API_ROOT`（`livegtw` HTTP API）当成 LiveKit WebSocket 地址。
- 本地 Vite 默认使用同源 `/livekit`，开发代理会去掉此前缀并转发到宿主机 `127.0.0.1:7880`。
- 生产和测试环境通过 `VITE_LIVEKIT_URL` 配置 Nginx 暴露给浏览器的 `wss://` 地址；不得把 Docker 内部地址（例如 `ws://livekit-server:7880`）直接下发给浏览器。同源 `/livekit` 代理必须去掉此前缀并透传 WebSocket Upgrade 请求头。
