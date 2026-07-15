# 整理 common tool 工具函数并统一秒级时间

## Goal

Organize the mixed helpers in `common/tool` into responsibility-focused files while preserving the existing `tool` package API, and provide a shared second-precision time helper for code paths that must persist timestamps without sub-second precision.

## Requirements

- Keep `common/tool` as one Go package and preserve existing exported function names and behavior.
- Split the current catch-all `util.go` helpers into smaller files by responsibility.
- Add a shared helper for the project-standard current time rounded to the start of the current second.
- Use the shared helper in ISP task start handling so `plan_start_time` and `start_time` are persisted without milliseconds.
- Keep the change minimal; avoid broad rewrites or unrelated API changes.

## Acceptance Criteria

- [ ] Existing call sites continue to compile without import or function-name changes.
- [ ] `common/tool` helpers are grouped into focused files instead of one large mixed `util.go`.
- [ ] ISP task start flow uses the shared second-precision time helper.
- [ ] Focused tests for `common/tool` and ISP task handler pass.

## Notes

- Lightweight task; PRD-only planning is sufficient.
