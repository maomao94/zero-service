# livegtw 用户身份校验实施计划

## Checklist

- [ ] 在 `MeetingAuthMiddleware` 完成 JWT claims bridge 后校验 `authctx.GetUserId`。
- [ ] 缺失或空白 user ID 时使用项目统一错误响应并提前返回。
- [ ] 保持 Authorization、auth type、claim mapping 和下游 context 传播。
- [ ] 增加中间件成功/拒绝测试；验证标准 `user-id` 和映射 `user_id` 两种 claim。
- [ ] 确认 routes 中业务组继续使用该 middleware，票据路由不使用。

## Validation

```bash
go test ./app/livegtw/...
go build ./app/livegtw/...
git diff --check
```

## Risk And Rollback

- 错误响应必须在 handler 前结束，否则可能产生下游调用或重复响应。
- 不修改生成 routes 结构；仅修改手写 middleware 和测试。
- 若现有网关错误格式不适合 middleware，使用项目统一 `xhttp.JsonBaseResponseCtx`，不直接写自定义 body。
