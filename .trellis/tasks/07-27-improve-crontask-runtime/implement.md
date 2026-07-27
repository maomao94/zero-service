# Implementation Plan

1. 调整公共执行时间契约。
   - 新增 `TaskConfig.LastScheduledRun`。
   - 新增 `Completion` 并调整 `TaskStore.Complete`。
   - 更新 Scheduler 和 MemoryStore。
   - 补充成功、失败、跳过、RunNow 和历史字段保留测试。
2. 完善 Scheduler 审计日志。
   - 为周期执行和 RunNow 增加稳定 context fields。
   - 覆盖 claim、handler、next-run 和 completion 边界。
   - 不记录 payload/extra。
3. 实现 RRULE 中文描述器。
   - 解析 RRULE/RRULE Set。
   - 规范化时区、默认值、列表和时间组合。
   - 生成中文周期、日期、时间、边界、RDATE/EXDATE 文案。
   - 对不支持组合返回明确错误。
4. 验证并审查。
   - 修改 `CalcPlanTaskDateRes` 并执行 `app/trigger/gen.sh`。
   - 在 Calc Logic 中复用同一 RRULE Set 生成 `scheduleDescription`，补 Logic 测试。
   - `gofmt` 修改过的 Go 文件。
   - `go test ./common/crontask`
   - `go test -race ./common/crontask`
   - `go vet ./common/crontask`
   - 运行 Trigger 相关目标测试或编译；若被既有 Store 接口迁移阻塞，明确记录阻塞调用方。
   - `git diff --check`
   - 检查 diff 只修改 `common/crontask/` 和当前 Trellis 任务产物。

## Review Gates

- 时间字段写入满足 handler 成功和 lease CAS 约束。
- 中文文案与 rrule-go 的实际候选时间语义一致。
- 日志有定位价值且不泄露业务负载。
- 不新增第三方依赖或兼容层。

## Rollback Points

- 时间契约与描述器分提交块实现，任一失败可独立回滚当前未提交 diff。
