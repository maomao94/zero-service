# AI Agent Project Implementation Plan

## Implementation Checklist

- [ ] Merge remaining exploration results into PRD/design.
- [ ] Confirm final Phase 1 design decision for ADK-primary vs lite-runtime coexistence.
- [ ] Inspect dirty worktree paths before any edit and preserve unrelated changes.
- [ ] Characterization: add tests around current high-risk `common/einox` contracts before behavior edits.
- [ ] Characterization: cover protocol adapter normal stream, tool-call-only stream, interrupt event, and receive-error path.
- [ ] Characterization: cover runtime runner event/tool behavior with fake model and fake tools.
- [ ] Characterization: cover knowledge chunking with current max-size and desired overlap behavior.
- [ ] Contract cleanup: fix or remove unused `WithModelOption` behavior.
- [ ] Contract cleanup: consolidate or clearly route duplicate chat model factories.
- [ ] Contract cleanup: make disabled knowledge/search-tool state explicit enough that callers cannot confuse disabled with successful initialization.
- [ ] Execution boundary: update or document `aisolo` default Agent Ask path so it does not conflict with Resume semantics.
- [ ] Reliability: fix protocol adapter stream error propagation and add tests.
- [ ] Reliability: ensure tool-call-only assistant streams produce observable protocol events.
- [ ] Reliability: clarify tool kit / policy / runtime registry flow with tests.
- [ ] Knowledge: implement or remove knowledge chunk overlap behavior and add tests.
- [ ] Consumer check: update `aisolo` executor tests if the default Agent execution boundary changes.
- [ ] Consumer check: update `aigtw` tests only if gateway mapping/contracts are touched.
- [ ] Consumer check: verify `aigtw` knowledge config parity assumptions with `aisolo` and document health/meta expectations.
- [ ] Consumer check: verify SSE stream error behavior is represented by protocol events or documented as post-header logging only.
- [ ] Run targeted validation commands and record results.

## Suggested Work Breakdown

### Work Item A: Test Harness And Characterization

Primary paths:

- `common/einox/protocol/adapter.go`
- `common/einox/runtime/runner.go`
- `common/einox/knowledge/chunk.go`
- `common/einox/agent/agent_option.go`

Deliverable: failing or characterization tests that describe the intended contract before implementation edits.

### Work Item B: Wrapper Contract Fixes

Primary paths:

- `common/einox/agent/*`
- `common/einox/model/*`
- `common/einox/knowledge/*`

Deliverable: cleaned public wrapper APIs with tests proving behavior.

### Work Item C: Execution Boundary Alignment

Primary paths:

- `aiapp/aisolo/internal/turn/executor.go`
- `aiapp/aisolo/internal/turn/executor_state_test.go`
- `common/einox/runtime/runner.go` if lite runtime is retained/documented

Deliverable: default Agent Ask/Resume semantics aligned with ADK-primary design or explicit lite-runtime limitation.

### Work Item D: Gateway Compatibility Review

Primary paths:

- `aiapp/aigtw/doc/aigtw.api`
- `aiapp/aigtw/internal/logic/solo/*`
- `aiapp/aigtw/internal/svc/servicecontext.go`

Deliverable: no contract mismatch between HTTP/SSE promises and stabilized `aisolo`/`einox` capabilities.

## Validation

Initial command set, to be refined after exact files change:

- `go test ./common/einox/...`
- `go test ./common/einox/protocol ./common/einox/runtime ./common/einox/knowledge ./common/einox/agent ./common/einox/model ./common/einox/tool/...`
- `go test ./aiapp/aisolo/internal/turn ./aiapp/aisolo/internal/svc ./aiapp/aisolo/internal/logic`
- `go test ./aiapp/aigtw/internal/logic/solo ./aiapp/aigtw/internal/svc` if gateway contracts are touched
- `go build ./aiapp/aisolo/... ./aiapp/aigtw/...` if implementation changes compile surfaces

Validation evidence to record in the final implementation pass:

- command;
- exit code;
- relevant failing test names if any;
- whether failure is caused by this task or pre-existing environment/dependency state.

## Review Gates

- Gate 1: Remaining exploration results merged and Phase 1 execution-boundary decision confirmed.
- Gate 2: Characterization tests pass before risky refactor.
- Gate 3: Targeted tests/build pass after implementation.
- Gate 4: Trellis check before marking task ready for implementation completion.

## Definition Of Done For Phase 1

- `common/einox` high-risk wrapper contracts are covered by deterministic tests.
- ADK-primary vs lite-runtime responsibilities are documented in `design.md` and reflected in code/tests.
- `aisolo` Ask/Resume behavior no longer depends on an implicit, incompatible dual path.
- `aigtw` gateway promises are compatible with stabilized backend capabilities.
- All changed Go packages pass targeted tests.
- Any skipped validation is documented with the exact blocker.
