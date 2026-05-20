# Eino / ADK Best-Practice Notes

Source: Context7 queries for `/websites/cloudwego_io` and `/cloudwego/eino-examples`.

## Official Pattern: Runner + CheckPointStore For Interrupt / Resume

CloudWeGo Eino ADK examples create an `adk.Runner` with `RunnerConfig{Agent, EnableStreaming, CheckPointStore}`. A first run calls `runner.Query(..., adk.WithCheckPointID("session-1"))`; when an interrupt appears in `event.Action.Interrupted`, resume later calls `runner.Resume(ctx, "session-1", ...)` using the same checkpoint id.

Design implication: any Solo flow that exposes human-in-the-loop interrupt/resume should use one coherent ADK Runner/checkpoint execution path. Mixing a non-ADK Ask path with ADK Resume is not a best-practice shape.

## Official Pattern: Streaming Events Are Iterated And Mapped

Examples iterate `AgentEvent` streams with `iter.Next()`, check `event.Err`, inspect message output and actions, and publish JSON SSE events. This supports the project design of translating ADK events into stable `common/einox/protocol.Event` frames and then forwarding those frames through gRPC/SSE.

Design implication: stream receive errors should be propagated into protocol/session handling, not merely logged and hidden as a successful turn.

## Official Pattern: Tools / Resume Options Are Passed Through Runner APIs

Resume examples pass tool options to `runner.Resume`, and human-in-the-loop examples use interrupt/resume context to carry user decisions.

Design implication: human-interaction tools should live only on execution paths that support ADK interrupt/resume. Lite runtime can support simple tool calls, but should not claim human-in-loop parity unless it implements equivalent checkpoint/resume semantics.

## Official Pattern: Live Provider Setup Is Runtime Configuration

Examples initialize concrete providers such as OpenAI from environment/config and then inject the resulting model into agents. Unit tests should not require live API keys.

Design implication: Phase 1 tests should use fake `model.BaseChatModel` and fake tools; live Eino provider tests should remain optional integration tests.

## Recommendation For This Repo

- Make ADK Runner the primary session Agent engine.
- Keep `RuntimeRunner` only as an explicitly named lite runtime for non-resumable flows, or demote it until a product use case needs it.
- Use `session_id` as the checkpoint id consistently for Agent sessions.
- Preserve `protocol.Event` as the stable transport-neutral event contract.
- Add deterministic tests around adapter event mapping, stream error propagation, interrupt extraction, and resume payload handling.
