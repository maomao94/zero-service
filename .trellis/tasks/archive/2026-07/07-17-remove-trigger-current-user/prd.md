# Remove trigger current user proto field

## Goal

Remove `currentUser` from trigger RPC request payloads and rely on propagated context for current-user data.

## Requirements

- Delete `extproto.CurrentUser currentUser = 100;` from `app/trigger/trigger.proto` request messages.
- Regenerate trigger protobuf/go-zero outputs from the service generation script when possible.
- Update trigger business logic that reads `in.CurrentUser` to read the current user from `context.Context` instead.
- Preserve existing request field numbers for all remaining fields.
- Keep the change scoped to `app/trigger` unless generated outputs or compile errors require adjacent updates.

## Acceptance Criteria

- [x] `app/trigger/trigger.proto` no longer exposes `currentUser` request fields.
- [x] Generated trigger Go/validation/descriptor outputs no longer include request `CurrentUser` accessors or validation for removed fields.
- [x] Trigger logic no longer references `in.CurrentUser`.
- [x] Targeted trigger verification passes or any blocker is documented with the failing command and cause.

## Confirmed Facts

- `app/trigger/gen.sh` is the generation entrypoint for `trigger.proto`.
- Project convention requires editing `.proto`, running the service `gen.sh`, then syncing `internal/logic`.
- Context propagation is the standard path for cross-service user data.

## Out of Scope

- Changing `extproto.CurrentUser` itself.
- Refactoring unrelated trigger request fields or endpoint behavior.
- Adding compatibility shims for the removed request-body field.

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
