# livegtw 网关 — 执行计划

## 前置

- [ ] live-service 的 pb 代码已生成（LiveRpc client 可用）
- [ ] 参考 gtw/ 的 handler/logic/svc 组织 + xhttp.JsonBaseResponseCtx 响应封装

## 步骤

1. **骨架**：app/livegtw/ 目录 + livegtw.go main + etc/livegtw.yaml + internal/config
2. **svc**：ServiceContext 组装 LiveRpc zrpc client（`live.NewLiveRpc`）+ webhook KeyProvider
3. **api 定义**：livegtw.api 定义会议路由（goctl api 生成 handler/logic 骨架；或手写 handler——参考 gtw 混合模式）
4. **业务 logic**：10 个会议 API 转发（req→RPC req→resp）
5. **webhook**：handler + logic（验签 → 转发 WebhookNotify）
6. **测试页**：static/index.html（占位，正文由 live-test-page 任务提供）+ embed handler + 路由
7. **单测**：webhook 验签 200/401、转发参数映射
8. **验证**：`go build ./livegtw/...`、`go vet ./livegtw/...`、`go test ./livegtw/...`

## 验证命令

```bash
go build ./livegtw/...
go vet ./livegtw/...
go test ./livegtw/...
# 冒烟：
go run ./app/livegtw -f app/livegtw/etc/livegtw.yaml
curl http://127.0.0.1:11002/test/meeting
```

## 评审门

- [ ] 10 个业务 API 转发正确
- [ ] webhook 验签失败 401 且不转发（有测试）
- [ ] /test/meeting 返回 200
- [ ] 编译/vet/test 全绿

## 回滚点

- 本任务只新增 livegtw 目录，删除即可回滚
- 依赖 app/live 的 pb 代码存在（若 live-service 未完成，本任务阻塞）