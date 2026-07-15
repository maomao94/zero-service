# Diagnose cron next_run timezone offset

## Goal

Diagnose and fix the cron task scheduling bug where an hourly task executed at `2026-07-15 13:44:22` is scheduled for `2026-07-15 22:44:21` instead of the expected next local run at `2026-07-15 14:44:21`.

## Requirements

- Identify why `next_run` is offset by 8 hours for an hourly RRULE containing `DTSTART;TZID=Asia/Shanghai`.
- Preserve correct local-time scheduling semantics for Asia/Shanghai cron task configs.
- Add focused regression coverage at the smallest correct seam if one exists.
- Keep the change scoped to cron task next-run calculation and related parsing behavior.

## Acceptance Criteria

- [ ] Given an hourly cron config with `DTSTART;TZID=Asia/Shanghai:20260715T134421`, `last_run=2026-07-15 13:44:22`, and hourly interval `1`, the next run is calculated as `2026-07-15 14:44:21` local time, not `2026-07-15 22:44:21`.
- [ ] Existing cron scheduling behavior remains covered by tests or targeted verification.
- [ ] Root cause is documented in the final summary.

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
