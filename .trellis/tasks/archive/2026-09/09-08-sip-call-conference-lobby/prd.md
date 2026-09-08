# 电话拨号直接进入会议大厅体验

## Goal
拨号后立即进入对应会议，主持人留在会议中等待电话用户加入；测试电话控件紧凑呈现。

## Scope
- 调整 `web/live` 的 DialPad 调用链，使独立拨号使用 `DialSipReply.meeting.meetingNo` 进入会议。
- 当前会议拨号不改变已有会议上下文。
- 将大厅测试电话从整行展开面板改为紧凑 tag/横向控件，响应式适配移动端。

## Dependencies
依赖后端现有 `DialSip` 返回会议；不依赖通知 RPC。

## Acceptance Criteria
- [ ] 独立拨号成功后进入返回的 S 会议并显示等待状态。
- [ ] 已有会议拨号不跳转到错误会议。
- [ ] 桌面不占整行，移动端不溢出；拨号失败仍显示错误。
- [ ] `npm run build` 通过。
