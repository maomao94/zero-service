# 执行计划

- [x] 统一 `TaskConfig`、`TaskStore.UpdateNextRun` 和 MemoryStore 的零值时间语义。
- [x] 调整调度计算和执行路径：RRULE 耗尽写空值，空值手动触发不 panic。
- [x] 将 ISP GORM 模型改为 `sql.NullTime`，更新 DBStore 与转换层。
- [x] 调整 ISP 任务规则构造、cron handler、任务控制和列表输出。
- [x] 更新并补充通用调度器、MemoryStore、DBStore、转换和 ISP 规则测试。
- [x] 运行 `gofmt`、目标包测试、受影响包 `go vet` 和 `git diff --check`。
- [x] 检查 diff 只包含本任务文件，不覆盖现有 gnetx、ISP client 和配置改动。

## 验证命令

```bash
go test ./common/crontask ./app/ispagent/internal/crontask ./app/ispagent/internal/handler ./app/ispagent/internal/svc ./app/ispagent/internal/logic
go vet ./common/crontask ./app/ispagent/internal/crontask ./app/ispagent/internal/handler ./app/ispagent/internal/svc ./app/ispagent/internal/logic
git diff --check
```
