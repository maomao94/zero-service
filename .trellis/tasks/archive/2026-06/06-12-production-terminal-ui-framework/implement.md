# Production terminal UI framework Implementation Plan

## Execution Order

1. Complete `06-12-uix-production-framework-foundation` first so shared layout/help/status behavior is stable.
2. Complete `06-12-dtui-production-host-navigation` next so all modules are reachable and command visibility is fixed.
3. Complete `06-12-dtui-docker-resource-modules` after host wiring, because resource modules depend on visible navigation/help surfaces.
4. Complete `06-12-dtui-config-compose-deploy-workflows` after config/host shape is stable.
5. Complete `06-12-dtui-docs-packaging-quality` last, then run the parent integration review.

## Concurrency Guardrails

- Run child tasks sequentially. Do not start multiple child implementations or checks in parallel.
- Avoid background subagents unless a child task has a concrete, narrow reason; if used, dispatch one at a time and include `Active task: <task path>` in the prompt.
- Prefer direct code inspection and focused local tests over broad agent fan-out.
- If external web research is needed, fetch one source at a time and do not block implementation on network failures when local specs answer the question.
- Before starting implementation, explicitly target/start the first child task (`06-12-uix-production-framework-foundation`); the current breadcrumb may point to the last-created docs child.

## Parent Review Checklist

- Verify every child task has reviewed `prd.md`, `design.md`, and `implement.md` before `task.py start`.
- Verify child tasks update `implement.jsonl` and `check.jsonl` with relevant specs/code before implementation.
- Verify each child `implement.jsonl` includes the parent task artifacts for cross-child requirements.
- Confirm no child requires Docker daemon at `dtui` startup.
- Confirm no child enables `!` shell execution.
- Confirm destructive/overwrite operations are default-visible but require second confirmation consistently across modules and docs.

## Final Validation

```bash
go test ./cli/uix/... ./cli/dtui/...
go build ./cli/dtui
go vet ./cli/uix/... ./cli/dtui/...
git diff --check
```

## Risk Points

- Terminal layout regressions from status/help changes.
- Duplicate config modules (`plugins/config` and `plugins/settings`) causing inconsistent behavior if both remain.
- Docker actions that block without timeout or hide command output.
- Deploy copy without backup/rollback semantics.
- README drifting from actual `main.go` wiring.

## Rollback Points

- After each child task, run focused validation before starting the next child.
- If host wiring breaks no-Docker startup, revert only `cli/dtui/main.go` and related module registration changes.
- If framework layout changes break modules, revert `cli/uix` changes before touching Docker workflows.
