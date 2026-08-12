# Add shared RDATE contract and compiler

## Goal

建立 Plan 与 CronJob 共用的指定时间/精确排除时间请求契约和 RRULE Set 编译能力，为后续两条业务链路提供统一、已生成且有测试的基础。

## Requirements

- 在相关 Trigger 请求和 CronJob 管理视图中增加 `repeated string specified_times` 与 `repeated string excluded_times`，各最多 1000 项，格式 `yyyy-MM-dd HH:mm:ss`。
- 运行 `app/trigger/gen.sh` 更新所有生成物，不手工修改生成代码。
- 扩展共享 Schedule 编译函数，按 `Asia/Shanghai` 解析并分别写入 RDATE/EXDATE。
- 每个精确时间必须位于规范化后的闭区间 `[start_time, end_time]`。
- RRULE 与指定时间同秒去重；精确排除只排除同一秒；`exclude_dates` 排除当天所有候选。
- 本子任务只建立公共契约和编译能力，不接入 Plan/CronJob Logic 或数据库模型。

## Acceptance Criteria

- [ ] Proto 字段及生成代码一致，空列表合法，第 1001 项校验失败。
- [ ] 编译结果包含可解析的 RDATE/EXDATE，时区和秒精度正确。
- [ ] 开始/结束边界可命中，区间外和格式错误被拒绝。
- [ ] RRULE/RDATE 重复时间只产生一个候选，排除日期移除当天 RDATE。
- [ ] 现有不传 RDATE 的规则行为和测试保持不变。

## Dependency

- 无；这是后续 Plan 与 CronJob 集成的前置子任务。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
