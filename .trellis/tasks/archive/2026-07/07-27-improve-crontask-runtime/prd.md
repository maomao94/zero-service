# 完善 CronTask 调度基础能力

## Goal

完善 `common/crontask` 的通用调度契约与可观测性：准确区分计划执行时间和实际成功完成时间，记录 handler 回调全生命周期审计日志，并将持久化的 RFC 5545 RRULE/RRULE Set 生成为适合业务展示的简体中文描述。

## Confirmed Facts

- `TaskConfig.NextRun` 在 Store 中会在 claim 后临时变为 lease 截止时间；原计划时间需要独立字段表达。
- 当前 `LastRun` 在 handler 成功返回后写入，实际语义是成功完成时间，不是计划执行时间。
- 当前 `TaskStore.Complete` 只接收 `nextRun` 和 `lastRun`，无法明确表达本次原计划时间。
- 当前调度器只有失败日志，成功回调、下次时间计算和完成提交没有审计日志。
- 项目已依赖 `github.com/teambition/rrule-go v1.8.2`，它支持解析 RRULE 及包含 `DTSTART`、`RRULE`、`RDATE`、`EXDATE` 的 RRULE Set。
- 需求 API 尚未上线，无需为本次公共契约保留旧签名或兼容层。

## Requirements

- 生产代码主要位于 `common/crontask/`；允许修改 Trigger 的 `CalcPlanTaskDate` proto、生成产物、Logic 和测试以接入规则描述，不修改其他业务 API、GORM 模型或回调业务实现。
- `TaskConfig` 明确区分：
  - `ScheduledTime`：当前在途 claim/retry 对应的原计划时间；未在途时为零值。
  - `LastRun`：最近一次 handler 成功完成的实际时间。
  - `LastScheduledRun`：最近一次成功周期执行对应的原计划时间；手动 `RunNow` 不更新。
- `TaskStore.Complete` 使用结构化完成结果，同时传递本次原计划时间、实际成功完成时间和下次计划时间；零值语义明确。
- 周期执行成功时原子提交 `LastRun` 与 `LastScheduledRun`；失败、panic、过期 lease 和 MaxDelay 跳过不得伪造成功时间。
- `RunNow` 成功只更新 `LastRun`，不更新 `LastScheduledRun`，也不改变周期 `NextRun`。
- 调度器在 claim、handler 开始、handler 成功/失败、删除、跳过、下次时间计算和完成提交边界输出结构化审计日志；日志包含任务标识和关键时间，不记录 payload/extra。
- 所有非空 `RRuleStr` 必须是包含 DTSTART 和 RRULE 的完整 RRULE Set；空字符串仍表示一次性任务，不再接受或生成裸 RRULE。
- 新增 `DescribeRRule(value string) (string, error)`：
  - 空字符串返回空描述。
  - 只解析完整 RRULE Set string。
  - 以 `DTSTART` 时区展示 `DTSTART`、`UNTIL`、`RDATE` 和 `EXDATE`。
  - 输出稳定、准确的简体中文业务描述，不通过翻译英文 `toText` 实现。
  - 支持 `FREQ`、`INTERVAL`、`BYMONTH`、`BYMONTHDAY`、`BYDAY`、`BYHOUR`、`BYMINUTE`、`BYSECOND`、`COUNT`、`UNTIL`、`DTSTART`、`RDATE`、`EXDATE`。
  - 同维度值按并集描述，不同过滤维度按交集描述；时间使用小时、分钟、秒的笛卡尔积。
  - 无法准确描述的高级组合返回可识别错误，不生成误导性兜底文案。
- 不引入新的 RRULE humanizer 第三方依赖。
- `CalcPlanTaskDateRes` 新增 `scheduleDescription`，由当前实际展开日期所使用的同一个 RRULE Set 生成；描述失败返回参数错误，不返回与 `planDates` 不一致的空描述。
- `CalcPlanTaskDateRes` 返回最终 `rruleStr` 原文用于排障。
- Trigger CronJob 与 ispagent 配置查询动态返回 `scheduleDescription`，描述以持久化 RRULE Set 为唯一真值。
- Trigger Plan 在创建并展开全部日期时保存当次 RRULE Set 快照；该字段仅用于审计、排障和描述，不改变 Plan/Batch/ExecItem 状态机。

## Acceptance Criteria

- [ ] 给定 `DTSTART;TZID=Asia/Shanghai:20260727T000000\nRRULE:FREQ=DAILY;UNTIL=20261231T155959Z;BYHOUR=9;BYMINUTE=30;BYSECOND=0`，描述包含“每天 09:30 执行”及上海时区的有效期 `2026-07-27 00:00:00` 至 `2026-12-31 23:59:59`。
- [ ] 月、周、日、时、分频率和 `INTERVAL` 均有表驱动测试，覆盖负数月日、序号星期、多时间笛卡尔积、COUNT/UNTIL、EXDATE/RDATE、无效和不支持规则。
- [ ] 周期 handler 延迟成功后，`LastRun` 等于实际完成时间，`LastScheduledRun` 等于 claim 前的原计划时间。
- [ ] handler 失败、panic、MaxDelay 跳过和 lease CAS 失败不会错误更新两个成功时间。
- [ ] `RunNow` 成功只更新 `LastRun`，保留 `LastScheduledRun` 和周期 `NextRun`。
- [ ] 成功与失败回调链路均有可关联日志，且不包含 payload/extra。
- [ ] `CalcPlanTaskDate` 对每天 09:30 的规则同时返回计划日期和包含“每天 09:30 执行”的 `scheduleDescription`，描述包含规范化后的有效期和排除日期。
- [ ] Trigger proto 通过 `app/trigger/gen.sh` 重新生成，Logic 与生成契约保持一致。
- [ ] CronJob 与 ispagent 配置列表/详情的 `scheduleDescription` 与各自持久化 `rrule_str` 一致。
- [ ] Plan 详情/列表返回创建时用于展开日期的 `rruleStr` 和对应 `scheduleDescription`。
- [ ] `go test ./common/crontask`、`go test -race ./common/crontask`、`go vet ./common/crontask` 和 `git diff --check` 通过。

## Out Of Scope

- Trigger/ispagent 的 GORM 空轮询 SQL 静默；该能力需要具体 Store 使用 `gormx.WithoutSQLTrace`。
- Trigger/StreamEvent/ispagent 的业务回调分发和 gRPC 请求响应日志。
- Plan/PlanBatch/PlanExecItem 的调度和状态机改造；它们仍是独立的预展开调度体系，不属于 crontask。
- 数据库字段、迁移、其他 proto/API 和前端展示接入。
- 多语言框架和英文描述。

## Notes

- 后续存储适配任务需将 `LastScheduledRun` 映射为可空字段，并在同一 lease CAS 中提交。
