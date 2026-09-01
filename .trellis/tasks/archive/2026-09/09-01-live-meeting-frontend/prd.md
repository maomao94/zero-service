# 面向用户的 Live 会议前端

## Goal

将现有单文件页面改造成真实用户可用的群视频会议入口：先用 JWT 登录，再创建或加入会议，会议中完成视频、群聊和主持人管理。

## Requirements

- 首屏仅显示 JWT 登录，支持显示/隐藏、保存、退出和清晰错误提示。
- 登录后显示会议大厅；身份和昵称从 JWT claim 推导或生成默认值。
- 会议中以视频网格为主，底部媒体控制，侧栏提供群聊、成员、管理。
- 保留现有 livegtw API 和 LiveKit v2.22.1 客户端能力，不改后端。
- 所有网络、鉴权、API、LiveKit、媒体权限和管理错误必须同时 toast/banner + 日志提示。
- 支持响应式布局、会议号复制、空状态、请求防重复、危险操作确认、断开清理。
- 用户输入不得造成 DOM 注入；JWT 不写入日志。

## Acceptance Criteria

- [x] 未登录时不显示会议控制界面（authView/appView 状态切换）。
- [x] 登录/退出和刷新恢复流程可用（localStorage + JWT claim 默认身份），401 回登录页并明确提示。
- [x] 创建、加入、离开和断开恢复流程可用（enterMeeting/closeRoom 状态机）。
- [x] 双标签场景具备群视频、群聊（topic lk.chat）、成员管理、Data/RPC/屏幕共享入口（27 项功能点全 ✓）。
- [x] 桌面/移动布局无严重遮挡（@media max-width:800px），空状态和连接状态清楚（toast + status bar）。
- [x] inline JS 语法（node new Function）、go build ./app/livegtw/...、git diff --check 全通过。

## 实现摘要

单文件 index.html 重写为产品化会议页面：
- **登录**：JWT Token 输入（显示/隐藏）、localStorage 保存、自动恢复、401 回登录页、退出登录
- **大厅**：创建会议（标题）、会议号加入（自动身份/昵称）、会议号复制
- **会议**：视频网格（本地+远端 tile + 首字母 avatar）、底部摄像头/麦克风/屏幕共享/离开
- **侧栏**：群聊（lk.chat topic）、成员（选择目标）、管理（踢人/静音音频视频/刷新/结束会议/Data/RPC）
- **反馈**：全局 toast + 事件日志（可折叠诊断区）+ 按钮忙碌状态 + 危险操作 confirm
- **安全**：XSS 防护（textContent/escapeValue）、JWT 不写日志、用户输入不注入 innerHTML
- **响应式**：桌面视频+侧栏并排、移动端堆叠

## Notes

- 登录校验策略：调用 GET /live/v1/meeting/participants?meetingNo=token-validation，401 回登录页，其他错误进入大厅（token 已保存，首次业务调用会再次校验）。
- 原生 ChatMessage 发送 API 未验证，群聊使用已验证的 publishData topic=lk.chat。
- 真实音视频/屏幕共享需浏览器人工验证，CLI 无法覆盖。
