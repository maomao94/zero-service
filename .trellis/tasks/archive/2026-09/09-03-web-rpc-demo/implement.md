# 实施计划：数据发送与 RPC 解耦

## 执行步骤

### 1. 优化聊天逻辑 (`ChatPane`)
*   **位置**: `web/live/src/App.tsx` -> `ChatPane` -> `send` 函数。
*   **修改**:
    *   如果 `!guest`，先调用 `api.sendMeetingData(meetingNo, 'lk.chat', payload)`。
    *   无论是否访客，最后都调用 `room.localParticipant.publishData`。

### 2. 拆分调试工具 (`RealtimeTools`)
*   **位置**: `web/live/src/App.tsx` -> `RealtimeTools`。
*   **修改**:
    *   **Send Data**: 调用 `api.sendMeetingData`，UI 文本改为 "Send Data (API)"。
    *   **Perform RPC**: 调用 `api.performRpc`，UI 文本改为 "Perform RPC (API)"。
    *   **Client RPC**: 新增按钮 "Register Echo (SDK)" 和对应的模态框逻辑。

### 3. 验证
*   **Chat**: 登录用户发送消息 -> 查数据库 -> 其他用户收到消息。
*   **Chat Guest**: 访客发送消息 -> 无数据库记录 -> 其他用户收到消息。
*   **Debug Tools**: 分别测试三个按钮，确保功能独立且正确。

## 验证命令
- 前端: `npm run dev`
- 后端: 检查数据库记录 (chat), API 日志 (rpc/data).
