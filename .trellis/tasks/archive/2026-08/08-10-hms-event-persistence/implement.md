# 实施计划

- [ ] 在事件入口将 `tid`、`bid` 写入 SDK typed context，并提供只读 helper。
- [ ] 保持 HMS hook 签名和注册装配不变，从 context 保存推送关联字段及 trace ID。
- [ ] 调用 resolver 保存实际 `message_key` 和 message。
- [ ] 删除 HMS hook 中宿主拓扑查询和 `gimbal_position` 派生。
- [ ] 更新 GORM model，移除五个平铺索引字段并修正 imminent 注释。
- [ ] 扩展 SDK 分发、hook 和 model 测试。
- [ ] 运行目标包测试、`go test ./app/djicloud/internal/hooks` 和 `git diff --check`。
