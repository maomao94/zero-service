# Integrate RDATE into CronJob scheduling

## Goal

把已完成的共享指定时间/精确排除契约接入 CronJob 创建、更新、持久化、调度和管理视图。

## Requirements

- 依赖 `08-12-rdate-contract-compiler` 完成并通过检查。
- Create/Submit 新建、Update/Submit 更新传递两个精确时间列表；更新可替换和清空列表。
- CronJob 模型新增两份可空 JSON 列，转换层平铺/重建并回显。
- 最终 `rrule_str` 是调度、推进和预览的权威来源，保持 100 年限制和在途更新拒绝。

## Acceptance Criteria

- [ ] Create/Submit/Update/Get/List 的两个列表持久化与回显一致。
- [ ] 清空列表后旧 Set 不再包含对应 RDATE/EXDATE；在途更新不部分覆盖。
- [ ] 首次 next_run、完成推进和预览正确应用指定时间与精确排除。

## Dependency

- 必须在 `08-12-rdate-contract-compiler` 完成后开始；可在 Plan 子任务完成后顺序执行。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
