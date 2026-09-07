# SIP 供应商管理与电话测试

## Goal

完善 `web/live` 的 SIP 测试工作流，让开发人员可以在当前 LiveKit 体系中管理 SIP 供应商、选择供应商发起外呼，并在工作台直接测试电话功能。

## Requirements

- 在工作台增加“供应商”页签，展示 SIP 供应商的编码、名称、地址、主叫号码、启用状态和创建时间。
- 供应商页签支持新增、编辑、启用/禁用、删除，并在操作后刷新列表和显示成功/失败反馈。
- 供应商凭据支持录入和更新，但列表展示时不得显示认证密码。
- 工作台电话测试区始终显示供应商下拉选择；无供应商、供应商加载失败、拨号中和拨号失败均有明确状态。
- 会议管理中的电话拨号必须始终使用选中的供应商，不得因为只有一个供应商而隐藏选择控件。
- 大厅增加独立的“测试电话”入口，可选择供应商并在不加入会议时触发 SIP 外呼（后端 `meetingNo` 可选）。
- 保持现有 JWT、会议、LiveKit 房间、聊天和票据流程不变；API 根地址继续支持 `VITE_API_ROOT`，用于连接 `https://api.nycatai.com/live/v1` 等环境。

## Acceptance Criteria

- [ ] 登录后可打开供应商页签并加载供应商列表。
- [ ] 新增、编辑、启用/禁用和删除操作均调用现有供应商 API，并正确刷新 UI。
- [ ] 新增/编辑表单校验必填字段，编辑时支持保留未修改的密码。
- [ ] 会议拨号区域显示供应商下拉框，未选择供应商时阻止请求。
- [ ] 大厅测试电话可以选择供应商、输入号码并展示外呼结果中的 `sipCallId`。
- [ ] `npm run build` 通过，且 `git diff --check` 通过。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
