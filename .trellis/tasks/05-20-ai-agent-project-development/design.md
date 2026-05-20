# AI Agent Project Design

## Technical Design

## Architecture Decision Record: Execution Boundary

### Context

`common/einox` currently exposes two execution styles:

- ADK-based execution through `common/einox/agent/*` and `common/einox/protocol/adapter.go`.
- Custom lite execution through `common/einox/runtime/runner.go`.

`aiapp/aisolo/internal/turn/executor.go` currently prefers the custom runtime for default Agent Ask when available, while Resume uses the ADK pool. This is risky because Resume depends on ADK checkpoint/interrupt semantics that the custom runtime does not clearly own.

### Decision

Use **ADK-primary execution** for session-oriented Agent flows. A flow is session-oriented when it exposes any of these guarantees:

- checkpoint and resume;
- human interrupt handling;
- ADK mode behavior;
- multi-agent/prebuilt workflows;
- stable Solo protocol events for frontend streaming;
- session-scoped tools or knowledge binding.

Retain the custom runtime only as a **lite runtime** if needed for simple non-resumable chat/tool/RAG flows. Lite runtime must not claim checkpoint/resume parity with ADK unless it gains equivalent tests and contracts.

### Consequences

- `aisolo` default Agent Ask should not silently switch to a different semantic engine from Resume.
- If `RuntimeRunner` remains wired, call sites must make capability differences explicit.
- Tests should prefer asserting execution-path selection and event/error behavior over live model responses.
- Future feature expansion should add capabilities to the ADK-primary path first.

### Module Boundaries

- `common/einox`: shared Eino integration layer. Owns ADK agent wrappers, model factories, runtime/lite runner, protocol events, middleware, tool registry/policy, knowledge, memory, checkpoint, filesystem restrictions, and metrics.
- `aiapp/aisolo`: gRPC service and application orchestration. Owns session state, turn execution, mode selection, streaming RPCs, interrupt persistence, and service runtime composition.
- `aiapp/aigtw`: HTTP/SSE gateway. Owns external REST/SSE contracts, JWT boundary, request/response mapping, and forwarding to `aisolo` / `aichat`.

### Current Key Risk

The system has two execution architectures:

1. ADK path: `common/einox/agent/*` plus `common/einox/protocol/adapter.go`.
2. Custom runtime path: `common/einox/runtime/runner.go`.

`aiapp/aisolo/internal/turn/executor.go` currently uses the custom `RuntimeRunner` for default `AGENT` Ask when available, but Resume always uses the ADK pool. This may create inconsistent semantics for streaming events, tools, RAG, checkpoint, and human interrupts.

### Phase 1 Design Direction

Recommended decision: make ADK the primary execution path for session-oriented Agent modes, because current Solo semantics rely on checkpoint and interrupt/resume. Keep the custom RuntimeRunner only if it is explicitly named and tested as a “lite runtime” for non-interrupt chat/tool/RAG flows.

This decision is supported by official Eino ADK examples: interrupt/resume flows are built around one `adk.Runner`, `CheckPointStore`, and stable checkpoint id, while streaming examples iterate ADK events and map them to SSE payloads.

## Best-Practice Design Rules

### Eino / ADK Integration

- Treat ADK Agent, Runner, checkpoint, middleware, and interrupt types as a coherent unit. Avoid mixing ADK resume with a non-ADK Ask path for the same session mode.
- Register checkpoint-serialized interrupt/result payload types before runtime use and keep tests around that registration.
- Keep Eino-specific structs at the `common/einox` boundary; expose stable project protocol events to service and frontend layers.
- Prefer fake `model.BaseChatModel` implementations for unit tests. Live provider tests should be optional integration tests.

### Streaming Protocol

- Emit a stable event lifecycle: turn start, message/tool/interrupt/error events as applicable, then turn end/final frame.
- Propagate stream receive errors to the executor so the session state can be restored or marked failed intentionally.
- Tool-call-only assistant chunks must still produce observable protocol events, even when assistant text content is empty.
- Event JSON must remain transport-neutral so both RPC streams and HTTP SSE can reuse it.
- Once HTTP SSE headers are opened in `aigtw`, failures cannot become normal HTTP errors; upstream execution should emit observable protocol error events or return errors before stream start whenever possible.

### Tools

- Tool exposure should flow through one policy decision before registration/invocation.
- Runtime invocation must not accept tools that policy denied.
- Human-interaction tools should be available only on execution paths that support interrupt/resume semantics.
- Builtin tool constructors that panic are acceptable only for static startup wiring; dynamic wiring should use constructors that return `error`.

### Knowledge / RAG

- Knowledge service disabled state should be explicit and visible to health/meta callers.
- Chunking config must match behavior. If `ChunkOverlapRunes` is exposed, ingestion must apply overlap and tests must prove it.
- Retrieval context injection should be deterministic and bounded by configured top-k/max sizes.
- Session-bound knowledge base selection belongs in `aisolo` context wiring; vector store and search tool behavior belong in `common/einox`.
- Because both `aigtw` and `aisolo` initialize knowledge services, shared deployments must keep backend/data-dir config consistent and expose mismatches through health/meta state.

### Model Factory

- Maintain one canonical factory path for chat model creation.
- Provider-specific defaults should live in one place.
- Unsupported providers should fail clearly at startup or validation, not silently produce nil models.

### Contracts To Stabilize

- Agent options must either be applied or removed. Unused options such as `WithModelOption` should not remain as misleading API.
- Model creation should have one canonical factory or a clear wrapper/deprecated path.
- Protocol adapter must surface stream receive errors to callers rather than silently turning truncated streams into successful turns.
- Runtime and ADK paths must document differences in tool calls, event order, RAG injection, checkpoint, and interrupts.
- Tool selection policy and runtime invocation registry must not allow callers to bypass policy accidentally.
- Knowledge chunk overlap config must either affect chunking or be removed from the public config contract.

## Phased Delivery Plan

### Phase 1.0: Characterization

Goal: freeze current behavior with tests before changing risky wrapper seams.

- Add tests around current ADK protocol adapter behavior.
- Add tests around runtime stream/tool behavior.
- Add tests around knowledge chunking config.
- Add tests that expose `WithModelOption` as unused or define the desired behavior.

Exit criteria: tests identify current gaps and establish expected target behavior.

### Phase 1.1: Contract Cleanup

Goal: remove ambiguity in public wrapper APIs.

- Fix/remove unused agent options.
- Route duplicate model factories to a canonical implementation.
- Make disabled knowledge/tool states explicit at the call-site boundary.

Exit criteria: wrappers have one obvious supported path per concern.

### Phase 1.2: Execution Boundary Enforcement

Goal: prevent Ask/Resume semantic drift.

- Update `aisolo` executor selection to align with ADK-primary design or explicitly named lite runtime behavior.
- Add executor tests proving default Agent Ask and Resume use compatible semantics.

Exit criteria: session modes cannot accidentally mix incompatible execution engines.

### Phase 1.3: Protocol and Tool Reliability

Goal: make stream/tool failures observable and recoverable.

- Propagate adapter stream errors.
- Ensure tool-call-only streams emit visible events.
- Add policy-to-registry tests.

Exit criteria: stream truncation/tool denial does not appear as a successful turn.

### Phase 1.4: Consumer Verification

Goal: confirm service and gateway contracts still match the foundation.

- Run targeted `aisolo` tests.
- Run targeted `aigtw` tests if gateway mapping is touched.
- Update PRD/design if remaining exploration reveals contract mismatches.

Exit criteria: consumers align with stabilized common contracts.

### Data Flow

```text
aigtw HTTP/SSE contract
  -> aigtw logic validates/maps request
  -> aisolo RPC client
  -> aisolo logic validates request and creates stream emitter
  -> turn.Executor owns session state machine
  -> common/einox ADK agent/runtime boundary
  -> protocol.Event JSON frames
  -> aisolo stream chunk
  -> aigtw SSE frame
```

### Validation Strategy

- Prefer fake chat models and fake tools over live providers.
- Test wrapper contracts in `common/einox` first.
- Add or update `aisolo` tests only where service behavior depends on wrapper boundary decisions.
- Run targeted packages before widening to larger builds.

## Rollout / Rollback

Phase 1 is internal code stabilization. Rollout should be by tests and local service startup only; no production rollout is implied. If a wrapper fix risks broad behavior change, guard it with tests that capture the intended contract before editing service consumers.

## Risk Register

| Risk | Impact | Mitigation |
| --- | --- | --- |
| RuntimeRunner and ADK semantics diverge | Resume/interrupt bugs and inconsistent events | ADK-primary decision plus executor tests |
| Refactor breaks existing simple chat path | Regressions in current Solo usage | Characterization tests before edits |
| Tool policy bypass remains possible | Unsafe/unexpected tool invocation | One policy-to-registry flow and tests |
| Knowledge config lies to operators | RAG quality and debugging issues | Implement/remove overlap and expose disabled state |
| Live provider nondeterminism hides failures | Flaky or meaningless tests | Fake model/tool deterministic tests |
| Generated contract drift | Gateway/RPC mismatch | Only change `.api/.proto` with `gen.sh` and diff review |
| Gateway and Agent knowledge stores drift | CRUD/query results differ from Agent retrieval | Shared config contract plus health/meta visibility |
| SSE errors after headers are hidden | Clients see truncated streams as success | Emit protocol error events before final frame where possible |
