# 统一 live_gtw 用户鉴权与 user_id 校验

## Goal
阻止缺少业务 `user_id` 的 JWT 访问 live_gtw 业务 API。

## Scope
- 在 `MeetingAuthMiddleware` 的 JWT 校验之后读取 `authctx.GetUserId`，为空时拒绝并确保不调用下游 gRPC。
- 覆盖 claim mapping、外部 `user_id` 到标准 `user-id` 的桥接，以及统一网关错误响应。
- 明确 webhook 和票据免鉴权路由不经过该校验。

## Dependencies
依赖 `invite-contract-push` 确认通知调用者身份语义；可独立先实现中间件测试。

## Acceptance Criteria
- [ ] 缺少/空 user_id 的 token 被拒绝，合法 token 正常通过。
- [ ] claim mapping 使用下划线外部 claim 和标准 authctx key 均可工作。
- [ ] middleware 单测覆盖拒绝、通过和不影响免鉴权路由；`go test ./app/livegtw/...` 通过。
