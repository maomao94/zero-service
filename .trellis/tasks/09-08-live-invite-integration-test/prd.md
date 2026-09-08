# 联调测试与跨服务验收

## Goal
验证通知入会和电话会议体验从 API 到浏览器的完整数据流。

## Scope
- 后端单测/集成测试覆盖协议、身份校验、socketpush 调用、错误和重复请求。
- 前端构建与关键交互验证：登录连接、通知中心、接受入会、独立 SIP 拨号进入会议、电话用户后加入。
- 在具备环境时验证 socketgtw、Redis、LiveKit、SIP；缺失外部环境时明确记录未完成项。

## Dependencies
依赖前四个子任务完成。

## Acceptance Criteria
- [ ] `go test` 覆盖受影响 Go 服务，`npm run build` 通过，`git diff --check` 通过。
- [ ] 在线目标用户收到邀请并可入会；离线/失败路径结果可解释。
- [ ] 缺失 user_id 无法访问 live_gtw。
- [ ] 独立 SIP 外呼创建会议、前端进入会议、电话用户加入链路有记录。
