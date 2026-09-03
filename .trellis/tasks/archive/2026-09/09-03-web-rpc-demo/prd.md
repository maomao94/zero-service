# Web 端数据发送与 RPC 功能优化

## 目标
优化 Web 端数据发送逻辑，解耦聊天与调试功能，确保消息持久化，并完善客户端 RPC 注册演示。

## 背景
当前 `RealtimeTools` 组件耦合了数据发送和 RPC 调用，且消息上报逻辑不清晰。
根据需求，需区分两种场景：
1.  **业务聊天**：普通用户需先通过 API 上报数据库，再通过 SDK 广播；访客直接通过 SDK 广播。
2.  **调试工具**：保留 `sendMeetingData` 和 `performMeetingRpc` 作为独立的调试功能，用于定向测试。

## 需求

### 1. 聊天功能优化 (`ChatPane`)
*   **普通用户**：发送消息时，先调用 `/sendMeetingData` 接口存储数据，成功后再调用 `room.localParticipant.publishData` 广播。
*   **访客**：直接调用 `room.localParticipant.publishData` 广播（无鉴权，不上报数据库）。
*   **接收方**：参会人通过 SDK 接收消息，能识别发送者身份。

### 2. 调试工具拆分 (`RealtimeTools`)
将现有的 `RealtimeTools` 拆分为两个独立的测试功能：
*   **Send Meeting Data (API)**: 调用后端接口 `/sendMeetingData` 测试定向/广播数据发送。
*   **Perform RPC (API)**: 调用后端接口 `/performMeetingRpc` 测试服务端 RPC。
*   **Client RPC (SDK)**: 新增客户端 RPC 注册与接收演示。

## 验收标准
- [ ] 普通用户发送聊天消息时，数据库有记录（通过 API 查询验证）。
- [ ] 访客发送聊天消息时，仅通过 SDK 广播，无 API 调用。
- [ ] `RealtimeTools` 中的三个功能（Send Data, Perform RPC, Client RPC）互不干扰，逻辑独立。
- [ ] 客户端 RPC 注册后，能响应服务端的 `echo` 调用并弹窗。

## 范围外
- 后端接口修改。
