# 联调测试与跨服务验收执行计划

## Preconditions

- [ ] `09-08-invite-contract-push` 已完成并通过其测试。
- [ ] `09-08-livegtw-user-auth` 已完成并通过其测试。
- [ ] `09-08-sip-call-conference-lobby` 已完成并通过前端构建。
- [ ] `09-08-live-notification-center` 已完成并通过前端构建。
- [ ] 确认当前工作区只包含上述任务及用户已有变更。

## Execution

1. 审查 `live.proto`、livegtw API 和前端事件常量，搜索 RPC、event、payload 字段的全部调用方。
2. 为 `app/live` 的邀请 Logic 补齐成功、参数、会议状态和 socketpush 失败测试；断言目标 room、event、payload 和无 token。
3. 为 `app/livegtw` 的身份中间件/路由补齐 user_id 拒绝、合法 claim、票据与 webhook 回归测试。
4. 运行受影响 Go 服务测试和编译；审查生成代码及全仓调用方编译错误。
5. 运行 `web/live` TypeScript/Vite 构建。
6. 在可用环境中，用两个不同 user_id 浏览器会话验证邀请送达、通知展示、忽略、接受入会、重复通知和失效会议。
7. 在可用 SIP 环境中验证独立外呼后自动进入返回会议，并记录电话参与者入会结果。
8. 执行最终差异、敏感信息和范围审查。

## Validation Commands

```bash
go test ./app/live/...
go test ./app/livegtw/...
go test ./socketapp/socketpush/...
go test ./socketapp/socketgtw/...
go build ./app/live/...
go build ./app/livegtw/...
npm run build
git diff --check
```

在 `web/live` 下运行前端命令。若 Socket.IO、LiveKit、Redis、PostgreSQL 或 SIP 环境不可用，执行可运行的本地检查，并在验收记录中列出未执行的端到端场景和原因。

## Review Gates

- Go 测试必须验证“提交推送”与“目标已处理”不是同一语义。
- HTTP 测试必须证明 user_id 校验发生在 gRPC 调用之前。
- 前端验收必须确认通知 payload 不含 LiveKit token，接受操作走标准入会 API。
- 不因测试任务修改生成文件、部署凭据或无关模块。
