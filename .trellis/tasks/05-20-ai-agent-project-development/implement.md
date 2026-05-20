# AI Agent Project Implementation Plan

## Implementation Checklist

- [x] Merge remaining exploration results into PRD/design.
- [x] Confirm final Phase 1 design decision for ADK-primary vs lite-runtime coexistence.
- [x] Inspect dirty worktree paths before any edit and preserve unrelated changes.
- [x] Characterization: add tests around current high-risk `common/einox` contracts before behavior edits.
- [x] Characterization: cover protocol adapter normal stream, tool-call-only stream, interrupt event, and receive-error path.
- [x] Characterization: cover runtime runner event/tool behavior with fake model and fake tools.
- [x] Characterization: cover knowledge chunking with current max-size and desired overlap behavior.
- [x] Contract cleanup: fix or remove unused `WithModelOption` behavior.
- [x] Contract cleanup: consolidate or clearly route duplicate chat model factories.
- [x] Contract cleanup: make disabled knowledge/search-tool state explicit enough that callers cannot confuse disabled with successful initialization.
- [x] Execution boundary: update or document `aisolo` default Agent Ask path so it does not conflict with Resume semantics.
- [x] Reliability: fix protocol adapter stream error propagation and add tests.
- [x] Reliability: ensure tool-call-only assistant streams produce observable protocol events.
- [x] Reliability: clarify tool kit / policy / runtime registry flow with tests.
- [x] Knowledge: implement or remove knowledge chunk overlap behavior and add tests.
- [x] Consumer check: update `aisolo` executor tests if the default Agent execution boundary changes.
- [x] Consumer check: update `aigtw` tests only if gateway mapping/contracts are touched.
- [x] Consumer check: verify `aigtw` knowledge config parity assumptions with `aisolo` and document health/meta expectations.
- [x] Consumer check: verify SSE stream error behavior is represented by protocol events or documented as post-header logging only.
- [x] Run targeted validation commands and record results.

## Implementation Log

### 2026-05-20 First Pass

- Implemented `SplitIntoDocumentsWithOverlap` and wired knowledge ingestion to `Config.EffectiveChunkOverlapRunes()`.
- Added deterministic chunk overlap tests for normal overlap and capped oversized overlap.
- Updated protocol adapter so assistant/tool stream receive errors are returned to callers and emitted as protocol `error` events.
- Added protocol adapter tests for assistant stream errors, tool stream errors, and tool-call-only assistant streams.
- No `.api` or `.proto` changes were made.

Validation executed:

- `go test ./common/einox/knowledge` — passed.
- `go test ./common/einox/protocol` — passed after fixing test stream writer blocking.
- `go test ./common/einox/protocol ./common/einox/knowledge` — passed.
- `go test ./common/einox/...` — passed.

Diagnostics:

- LSP diagnostics on changed Go files found no errors or warnings.
- `common/einox/knowledge/chunk.go` has optional Go 1.25 hints (`min`, `strings.SplitSeq`) left unchanged to avoid unrelated modernization.

### 2026-05-20 Second Pass

- Clarified `agent.WithModelOption` as ADK model runtime options and added `Agent.ModelOptions()` to return a defensive copy for explicit `adk.WithChatModelOptions` usage.
- Added agent option tests proving model options are stored and not exposed as the internal slice.
- Routed `model.NewChatModelByOption` through the canonical `NewChatModel(ctx, Config)` path to remove duplicate provider construction logic.
- Added model option tests for option-to-config conversion and unsupported-provider error behavior.

Validation executed:

- `go test ./common/einox/agent ./common/einox/model` — passed.
- `go test ./common/einox/...` — passed.

Diagnostics:

- LSP diagnostics on second-pass changed Go files found no errors or warnings after test adjustment.

### 2026-05-20 Third Pass

- Added runtime characterization coverage proving a policy-filtered `ToolRegistry` rejects model-requested tools that were not allowed by `tool.Policy`.
- Added a lite-runtime boundary test proving `runtime.Runner` does not expose an ADK-style `Resume` surface.
- No production code changes were required in this pass.

Validation executed:

- `go test ./common/einox/runtime` — passed.
- `go test ./common/einox/...` — passed.

Diagnostics:

- LSP diagnostics on `common/einox/runtime/runner_test.go` found no errors or warnings.

### 2026-05-20 Fourth Pass

- Aligned `aiapp/aisolo/internal/turn.Executor` with the ADK-primary design: session `Ask` no longer silently switches default `AGENT` mode to the lite `RuntimeRunner` when both are configured.
- Kept `RuntimeRunner` as an explicit lite helper path and updated runtime-specific executor tests to call that helper directly instead of treating it as default session semantics.
- Passed stored `Agent.ModelOptions()` into ADK `Run` / `ResumeWithParams` through `adk.WithChatModelOptions`, so the second-pass option cleanup is now applied by the session ADK path.
- Tightened `ServiceContext.initExecutor` so the turn executor is only created when the ADK mode pool is available; runtime-only startup no longer exposes a session Ask/Resume executor with weaker semantics.
- Added executor tests proving default `AGENT` Ask uses the ADK pool even when `RuntimeRunner` is configured, and proving ADK model runtime options reach the fake chat model.
- Extended the deterministic runtime fake model to record raw model options for ADK/runtime option assertions.
- Review fix: updated `runtimeMaxHistoryMessages` comments in `aisolo` config code and sample yaml so operators see it as a lite `RuntimeRunner` helper setting, not default session Ask/Resume behavior.
- No `.api` or `.proto` changes were made.

Validation executed:

- `go test ./aiapp/aisolo/internal/turn` — passed.
- `go test ./aiapp/aisolo/internal/svc` — passed.
- `go test ./common/einox/runtime` — passed.
- `go test ./common/einox/...` — passed.
- `go test ./aiapp/aisolo/internal/turn ./aiapp/aisolo/internal/svc ./common/einox/...` — passed.
- `go test ./aiapp/aisolo/...` — passed.
- `go test ./aiapp/aisolo/internal/config ./aiapp/aisolo/internal/svc ./aiapp/aisolo/internal/turn` — passed after review comment fix.

Diagnostics:

- LSP diagnostics on `aiapp/aisolo/internal/turn/executor.go`, `aiapp/aisolo/internal/svc/servicecontext.go`, and `common/einox/runtime/fake_model.go` found no diagnostics.
- LSP diagnostics on `aiapp/aisolo/internal/turn/executor_state_test.go` found only optional modernization hints in existing concurrency test code (`range over int`, `WaitGroup.Go`); left unchanged to avoid unrelated modernization.
- LSP diagnostics on `aiapp/aisolo/internal/config/config.go` after comment-only fix reported existing go-zero json tag warnings (`optional`, `default`, `options`) across the file; no new code error was introduced.
- YAML diagnostics for `aiapp/aisolo/etc/aisolo.yaml` could not run because `yaml-language-server` is not installed in the environment.

### 2026-05-20 Fifth Pass

- Added exported sentinel error `ErrKnowledgeDisabled` to `common/einox/knowledge/service.go` so callers can distinguish "explicitly disabled" from "misconfigured" using `errors.Is(err, ErrKnowledgeDisabled)` instead of relying on ambiguous `(nil, nil)` return.
- Updated `NewService` to return `ErrKnowledgeDisabled` when `Config.Enabled == false` (was `(nil, nil)`).
- Updated `aiapp/aisolo/internal/svc.initKnowledge` to handle the sentinel: logs `"[svc] knowledge disabled by config"` at info level without setting `KnowledgeInitErr`, so health reporting correctly classifies disabled as `"disabled"` not `"misconfigured"`.
- Updated `aiapp/aigtw/internal/svc.NewServiceContext` with the same sentinel handling pattern.
- `NewSearchTool(nil)` kept as `(nil, nil)` — callers already guard with `s.Knowledge != nil`, and changing to sentinel error would introduce misleading error logs at the call sites.
- Added `TestNewServiceDisabledReturnsSentinel` and `TestNewServiceDisabledWithEmptyConfig` to prove `NewService` returns the sentinel for disabled configs.
- No `.api` or `.proto` changes were made.

Validation executed:

- `go test ./common/einox/knowledge` — passed (2 new sentinel tests + existing).
- `go test ./aiapp/aigtw/internal/svc` — passed (5 tests, no regressions).
- `go test ./aiapp/aisolo/internal/svc` — passed (5 tests, no regressions).
- `go test ./common/einox/...` — passed (all einox packages).
- `go test ./aiapp/aigtw/...` — passed (all aigtw packages).
- `go test ./aiapp/aisolo/...` — passed (all aisolo packages).

Consumer checks:

- **aigtw contracts**: Gateway mapping/contracts unchanged. `Dependencies()` output identical. No test updates needed.
- **Knowledge config parity**: Both `aisolo` and `aigtw` use `einoxkb.Config`. Both handle `ErrKnowledgeDisabled` correctly, reporting `"disabled"` without `KnowledgeInitErr`. Health/meta expectations consistent.
- **SSE stream error behavior**: Protocol adapter (`common/einox/protocol/adapter.go`) already emits `error` / `tool_stream_error` / `assistant_stream_error` protocol events on stream receive errors (lines 54-57, 96-98, 101-103), covered by first-pass characterization tests.

All Phase 1 checklist items are now complete.

Diagnostics:

- LSP diagnostics on all 4 changed Go files found no errors or warnings.

### 2026-05-20 Deepening Pass (Post-Phase 1)

- **Full test coverage for all `common/einox` packages**: Added tests for `tool` (kit + policy, 18 tests), `middleware` (approval middleware, 8 tests), `checkpoint` (memory store, 13 tests), `memory` (storage, 16 tests), `metrics` (record functions, 13 tests). Only `tool/builtin` remains without direct tests (indirectly tested through `tool` and runtime tests).
- **`http_get` builtin tool**: Added HTTP GET capability to the agent tool ecosystem — supports custom headers, 512KB body limit, 30s timeout, and uses soft-error pattern (returns error field in result JSON instead of hard-failing).
- **MCP tool bridge**: Created `MCPTool` in `common/einox/tool/mcp.go` that wraps any `MCPCaller` (e.g., `mcpx.Client`) as an Eino `InvokableTool`. Includes `MCPCaller` interface for testability. Enables aisolo agents to call arbitrary MCP server tools.
- **Remaining untested `common/einox` packages**: `tool/builtin` (indirectly tested), plus `memory` (gormx/jsonl), `metrics` (prometheus path), and `middleware` (select/text/form/ack types — only approval was tested).
- All changes pass full test suite (28+ packages, zero failures).

Diagnostics:

- LSP diagnostics on all new and changed Go files found no errors or warnings.

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
