# 前端优化：9位会议号 + 加入参数配置 + 票据类型

## Goal

优化前端页面，支持9位会议号加入会议、加入会议时的参数配置、以及票据类型选择（一次性/有效期）。

## Background

后端已支持：
- `JoinMeetingRequest` 支持 `meetingNo` / `meetingCode` 二选一
- `GenerateMeetingTicketRequest` 支持 `ticketType`（1=一次性，2=有效期）
- `JoinMeetingRequest` 支持权限配置（canPublish, canSubscribe, canPublishData, canPublishSources）

前端需要更新以使用这些功能。

## Requirements

### 1. 9位会议号加入会议
- 大厅页面"加入会议"输入框支持输入9位会议号（meetingCode）
- 输入框 placeholder 改为 "会议号或9位会议码"
- 调用 `api.joinMeeting` 时，判断输入内容：
  - 纯9位数字 → 传 `meetingCode`
  - 其他 → 传 `meetingNo`

### 2. 加入会议参数配置
- 大厅页面"加入会议"区域增加"高级选项"折叠面板
- 配置项：
  - 是否发布音视频（canPublish）
  - 是否订阅音视频（canSubscribe）
  - 是否发送消息/数据（canPublishData）
- 默认值：全部开启

### 3. 票据类型选择
- 生成票据弹窗增加"票据类型"选择
- 选项：
  - 一次性票据（默认）：消费后删除，只能使用一次
  - 有效期票据：消费后保留至过期，可多次使用
- 显示当前选择的票据类型说明

## Acceptance Criteria

- [ ] 大厅页面输入9位数字可以加入会议
- [ ] 大厅页面输入会议号（M开头）可以加入会议
- [ ] 加入会议时可以配置权限参数
- [ ] 生成票据时可以选择票据类型
- [ ] 票据类型选择有明确的说明文字
- [ ] 所有功能正常工作，无 TypeScript 错误

## Files to Modify

- `web/live/src/types.ts` - 更新类型定义
- `web/live/src/lib/api.ts` - 更新 API 调用
- `web/live/src/App.tsx` - 更新 UI 组件

## Notes

- 后端 API 已经支持，只需前端适配
- 保持现有 UI 风格，使用 Tailwind CSS
- 参考现有票据生成弹窗的实现
