# Configure gnetx debug hex format

## Goal

Allow `common/gnetx` debug payload logging to use selectable hex formats while keeping the current default output unchanged.

## Requirements

- `DebugSerializer` must keep its current default output format for existing callers.
- `DebugSerializer` must accept an optional format configuration for hex rendering.
- The new configuration must support at least compact lower-case hex, compact upper-case hex, and spaced upper-case hex for protocol debug logs.
- IEC104-style protocol logs must be able to opt in to spaced upper-case byte rendering.
- The change must not alter codec framing behavior or serializer semantics beyond log formatting.

## Acceptance Criteria

- [x] Existing `gnetx.DebugSerializer` callers keep the same default log output.
- [x] New tests cover at least one default-format case and one alternate-format case.
- [x] A caller can opt into spaced upper-case hex rendering for raw packet logs.
- [ ] `go test -race -count=1 ./common/gnetx/...` passes.

## Verification Notes

- Focused tests pass for `common/tool` and `common/gnetx` hex formatting.
- Full `go test -race -count=1 ./common/gnetx/... ./common/tool/...` is blocked by the pre-existing `TestClientOnConnectOnReconnect` timing failure in `common/gnetx`.

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
