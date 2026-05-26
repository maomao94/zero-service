# AI Agent Project Development

## Goal

Build a reliable AI Agent foundation for `zero-service` by validating and correcting the shared Eino abstraction in `common/einox` before expanding feature work through `aiapp/aisolo` and `aiapp/aigtw`. Phase 1 is foundation-first: make Eino execution boundaries, wrapper contracts, interrupt/resume semantics, tool policy, model factory behavior, and knowledge chunking testable and predictable.

## Background / Known Context

- User wants to continue development for `aiapp/aisolo`, `aiapp/aigtw`, and required `common/einox` work.
- Existing code may contain logic issues, especially around Eino wrapping.
- User selected **foundation-first** for Phase 1.
- Project backend convention: go-zero services keep Handler/Server thin, Logic owns business flow, common reusable AI capabilities belong in `common/einox`.
- API/RPC contracts must start from `.api` / `.proto`; generated files should be produced by `gen.sh`, not hand-written.
- `aisolo` already exposes session, stream, interrupt, mode, skills, messages, health, and knowledge-binding RPCs in `aiapp/aisolo/aisolo.proto`.
- `aigtw` exposes HTTP/SSE Solo APIs in `aiapp/aigtw/doc/aigtw.api` and forwards to `aisolo` RPC.
- `common/einox` currently contains both ADK-based Agent wrappers and a separate custom RuntimeRunner.

## Phase 1 Scope

Phase 1 prioritizes `common/einox` correctness and explicitly avoids broad product expansion. `aiapp/aisolo` and `aiapp/aigtw` are used only as consumers for compatibility checks and targeted validation.

## Product / Engineering Principles

- **Foundation before expansion**: current Agent features should stand on a deterministic, tested Eino integration layer before adding new endpoints, modes, UI, or provider capabilities.
- **ADK semantics are the source of truth for session agents**: any flow that promises checkpoint, interrupt, resume, multi-step tool execution, or ADK mode behavior should use the ADK execution model or explicitly document why it does not.
- **One public contract per concern**: model factory, tool policy, runtime execution, protocol event schema, and knowledge chunking should each have a single obvious contract. Duplicate APIs must be routed, deprecated, or removed.
- **Test deterministic boundaries, not LLM output**: tests should assert request validation, execution-path selection, tool invocation shape, emitted event order, interrupt/resume payload handling, and error propagation using fake models/tools.
- **Service code stays thin around common contracts**: `aisolo` should orchestrate sessions and streams; `aigtw` should map HTTP/SSE to RPC; reusable Eino mechanics belong in `common/einox`.
- **No invisible degradation**: disabled/misconfigured components should be observable through explicit state, health output, or errors; avoid ambiguous `(nil, nil)` patterns where callers can silently skip intended capabilities.

### Must Have

- Document and enforce the intended boundary between ADK Agent execution and the custom `runtime.Runner` path.
- Validate or fix high-risk Eino wrapper contracts:
  - agent options such as `WithModelOption`;
  - duplicate chat model factory behavior;
  - ADK protocol adapter error propagation and event semantics;
  - human interrupt/resume middleware behavior;
  - tool kit / policy / runtime registry consistency;
  - knowledge chunk overlap configuration.
- Add deterministic tests for wrapper behavior without requiring a live LLM provider.
- Confirm `aisolo` default Agent Ask/Resume behavior is consistent with the selected execution boundary.
- Preserve existing go-zero layering and generated-code boundaries.

### Must Not Have

- No new frontend/UI work.
- No broad rewrite of `aiapp/aisolo`, `aiapp/aigtw`, and `common/einox` in one phase.
- No new generic agent framework beyond what current Eino/ADK integration requires.
- No new provider abstraction unless needed to remove existing duplication safely.
- No product feature expansion such as new modes, new APIs, multi-agent workflows, or long-term memory beyond stabilizing current contracts.

## Requirements

- Treat `common/einox` as the primary delivery surface for Phase 1.
- Use executable tests to prove wrapper behavior; avoid manual-only AI behavior checks.
- Keep `aiapp/aisolo` RPC contract compatible unless a confirmed bug requires contract change.
- If `.proto` or `.api` changes become necessary, run the corresponding `gen.sh` and inspect generated diffs.
- Keep service-specific orchestration in `aiapp/aisolo/internal/turn` and `internal/logic`; keep reusable Eino contracts in `common/einox`.
- Preserve unrelated dirty worktree changes.

## Phase 1 Deliverables

### Deliverable 1: Execution Boundary Decision

- Decide whether session-oriented Agent execution is ADK-primary, runtime-primary, or dual-path with explicit capability differences.
- Recommended target: ADK-primary for session Agent flows; keep `runtime.Runner` only as a named lite runtime for non-interrupt chat/tool/RAG where checkpoint/resume is not promised.
- Document the decision in `design.md` before implementation.

### Deliverable 2: Wrapper Contract Hardening

- Fix misleading or unused contracts such as `WithModelOption`.
- Consolidate or route duplicate chat model factory APIs.
- Ensure protocol adapter stream errors are observable by callers.
- Ensure knowledge chunk overlap config either works or is removed from the supported contract.
- Clarify disabled knowledge/search-tool behavior so callers cannot accidentally treat disabled as successful initialization.

### Deliverable 3: Deterministic Test Harness

- Add fake-model and fake-tool tests for wrapper behavior.
- Cover ADK adapter events, interrupt extraction, stream error propagation, and tool-call-only output.
- Cover tool policy-to-registry consistency.
- Cover chunking with overlap if the config remains.

### Deliverable 4: Consumer Compatibility Check

- Verify `aisolo` executor behavior still matches its proto comments and session status machine.
- Verify `aigtw` does not promise capabilities that Phase 1 foundation cannot provide.
- Avoid gateway API changes unless exploration proves an existing contract mismatch.

## MVP Success Scenario

The first implementation phase is successful when a developer can run deterministic tests proving that a session Agent uses the intended execution path, emits protocol events consistently, propagates stream/tool errors, supports or explicitly rejects interrupt/resume according to the chosen boundary, and keeps knowledge/tool/model contracts unambiguous.

## Non-Goals For Phase 1

- Do not optimize prompt quality or evaluate model answer quality.
- Do not add new business tools beyond test doubles or contract examples.
- Do not add new persistence backends.
- Do not make distributed deployment the default; only preserve compatibility and document risks.
- Do not redesign static Solo UI or frontend interactions.

## Acceptance Criteria

- [ ] A design decision exists for ADK Runner vs custom RuntimeRunner responsibilities.
- [ ] `common/einox` contains tests that fail before and pass after each fixed wrapper-contract bug.
- [ ] Protocol event adapter tests cover normal message stream, tool-call-only stream, interrupt event, and stream error propagation.
- [ ] Knowledge chunking tests cover configured overlap if overlap remains part of config.
- [ ] Model factory tests or compile-time checks prove there is one supported creation path or a documented compatibility wrapper.
- [ ] Tool policy/registry tests prove blocked tools cannot be invoked through the runtime path by bypassing policy.
- [ ] `aisolo` tests prove Ask/Resume execution-path semantics are consistent with the final design decision.
- [ ] Targeted tests pass for changed `common/einox` packages.
- [ ] Targeted tests pass for impacted `aiapp/aisolo` packages if the execution boundary touches turn execution.
- [ ] Targeted tests pass for impacted `aiapp/aigtw` packages if gateway contracts or types are touched.
- [ ] `go test` commands used for validation exit with code 0 or failures are documented as pre-existing/unrelated.
- [ ] Any `.api` / `.proto` contract change is regenerated through module `gen.sh` and diff-inspected.
- [ ] No secrets, local absolute credentials, or provider API keys are introduced.

## Out of Scope

- Production deployment changes, Docker/Kubernetes rollout, or Nacos/service-discovery changes.
- New UI design or Solo web interface redesign.
- Full RAG quality tuning beyond correcting currently exposed config/contract bugs.
- Adding live-provider integration tests that require real API keys.
- Committing code; humans own commits in this project workflow.

## Research References

- [`research/aisolo-architecture.md`](research/aisolo-architecture.md) — `aisolo` contract, ServiceContext, executor, modes, tests, and risks.
- [`research/einox-wrapper.md`](research/einox-wrapper.md) — `common/einox` package map, Eino coupling, likely bugs, and remediation priorities.
- [`research/aigtw-integration.md`](research/aigtw-integration.md) — `aigtw` HTTP/SSE gateway, RPC forwarding, knowledge boundary, tests, and risks.
- [`research/cross-layer-contracts.md`](research/cross-layer-contracts.md) — create session, chat, resume, knowledge, modes/skills cross-layer contracts and mismatches.
- [`research/eino-best-practices.md`](research/eino-best-practices.md) — Eino ADK Runner/checkpoint/interrupt/streaming best-practice notes from official docs/examples.

## Open Questions

- Should Phase 1 remove/demote the custom RuntimeRunner path, or keep it as an explicitly documented “lite runtime” with narrower guarantees?

## Proposed Answer To Open Question

Recommended: **keep `runtime.Runner` temporarily as a named lite runtime, but make ADK-primary the only path for session Agent flows that expose checkpoint, interrupt, resume, or ADK mode semantics**. This minimizes disruptive deletion while preventing the current ambiguous dual-path behavior from leaking into user-facing promises.
