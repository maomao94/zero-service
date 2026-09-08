# Live 通知入会实施计划

## Checklist

- [ ] 更新 `app/live/live.proto`，新增 Req/Res、字段注释和 RPC 注释。
- [ ] 检查 `app/live` 配置是否已有 `SocketPushConf`；如无则按现有 zrpc 配置方式新增，并在 `ServiceContext` 条件装配 client。
- [ ] 执行 `app/live/gen.sh`，审查生成 diff，不手工编辑生成文件。
- [ ] 新增通知 Logic 和 server 路由适配，复用会议查询、错误映射和 authctx 约定。
- [ ] 使用现有 `socketpush.SocketPushClient.BroadcastRoom`，集中定义 event 与 JSON payload。
- [ ] 为成功、空 user_id、无调用者身份、会议不存在、会议结束、未配置 client、推送失败补充测试。
- [ ] 更新测试 fake/ServiceContext 初始化，使既有测试继续编译。

## Validation

```bash
cd app/live && ./gen.sh
go test ./app/live/...
go build ./app/live/...
git diff --check
```

再搜索新 RPC、event 和字段的全仓消费者，确认没有遗漏生成 client、网关或文档调用方。

## Risk And Rollback

- 主要风险是新增 `LiveRpcClient` 方法导致 webhook fake 等实现编译失败；生成后立即修复全部 fake。
- socketpush 配置缺失时不能让服务启动失败影响既有会议功能；仅通知接口返回明确错误。
- 如契约字段或事件语义需要改变，停止实现并回到 PRD/设计评审，不通过兼容性别名掩盖不一致。
