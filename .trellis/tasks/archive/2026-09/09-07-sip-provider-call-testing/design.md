# 技术设计

## 边界

本任务只修改 `web/live` 前端及其任务文档。使用已有的 `/sip-providers`、`/sip-providers/update`、`/sip-providers/delete` 和 `/sip-calls/dial` HTTP 契约，不修改后端协议或生成代码。

## 页面结构

- `LobbyView` 增加工作台页签：会议、供应商。
- 会议页保留现有创建/加入/历史记录，并新增内嵌 `DialPad`，允许 `meetingNo` 为空。
- 供应商页新增 `SipProviderPanel`，负责列表、表单和 CRUD 状态。
- `ManagePane` 复用 `DialPad`，传入当前会议号；会议中和大厅测试使用同一个拨号组件，避免供应商选择逻辑分叉。

## 数据与安全

- `SipProviderInfo` 继续作为列表模型；认证字段仅作为表单状态，不渲染到列表。
- 创建时发送完整表单；更新时发送 id 和非空变更字段。密码为空时不发送，表示保留原密码。
- 删除前使用 `window.confirm`。所有异步操作捕获异常并通过现有 toast 反馈。
- 供应商状态使用现有约定 `1=启用、2=禁用`。

## 配置与抓包

`api.ts` 保留 `VITE_API_ROOT || '/live/v1'`。生产/联调可设置 `VITE_API_ROOT=https://api.nycatai.com/live/v1`；开发代理仍服务于本地网关。请求路径和请求体均通过浏览器 Network 面板可见，便于抓包确认鉴权、供应商编码和外呼参数。

## 兼容性

不改变登录状态、LiveKit 连接状态或会议入会参数。供应商为空时页面可正常工作，只禁用/阻止外呼并给出提示。
