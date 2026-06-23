# GIS proto 支持围栏洞

## Goal

Allow GIS fence protocol messages to express polygon holes before the service reaches production.

## Requirements

- Replace single-ring fence geometry in proto contracts with an explicit polygon model.
- Preserve support for the existing simple outer-ring use case through the new polygon shape.
- Represent holes as inner rings that are excluded from point-in-fence and cell-coverage semantics.
- Keep the change focused on protocol definitions and comments unless generated-code impact requires follow-up.

## Acceptance Criteria

- [ ] `gis.proto` exposes a reusable polygon/ring structure for fences.
- [ ] Fence creation, update, detail, active point checks, and cell generation requests can carry holes.
- [ ] Proto comments clearly describe outer ring and hole semantics.
- [ ] References to old `points` geometry in the proto are removed or clarified.

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
