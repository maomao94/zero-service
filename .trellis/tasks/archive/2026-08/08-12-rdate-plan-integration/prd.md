# Integrate RDATE into Plan scheduling

## Goal

把已完成的共享指定时间/精确排除契约接入 Plan 日期计算和创建展开链路。

## Requirements

- 依赖 `08-12-rdate-contract-compiler` 完成并通过检查。
- `CalcPlanTaskDate` 将两个精确时间列表传给共享编译器，返回应用 RDATE/EXDATE 后的日期、描述和 Set。
- `CreatePlanTask` 传递两个列表，按最终 Set 展开 Batch/ExecItem，并保留 3 年和 5000 执行项限制。
- 不改变 Plan 状态机和运行时调度来源。

## Acceptance Criteria

- [ ] 日期预览包含范围内未排除的指定时间，并正确应用精确/整日排除与去重。
- [ ] CreatePlanTask 为 RDATE 创建对应 Batch/ExecItem。
- [ ] 3 年、过去时间过滤和 5000 项限制保持有效。

## Dependency

- 必须在 `08-12-rdate-contract-compiler` 完成后开始。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
