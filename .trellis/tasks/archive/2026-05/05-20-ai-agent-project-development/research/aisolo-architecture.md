# aisolo Architecture Findings

Source: Explore task `bg_0569f77c`.

## Summary

`aiapp/aisolo` is a gRPC service wrapping Eino ADK/runtime agents into session-oriented streaming turns. The stable external contract is `aiapp/aisolo/aisolo.proto`. `ServiceContext` assembles persistence, model, tools, knowledge, modes, and executor. `turn.Executor` owns the session status machine.

## High-Risk Findings

- Default config uses memory for messages, sessions, and checkpoint, so process restart loses resumability.
- Default Agent has two execution paths: Ask can use `RuntimeRunner`, while Resume always uses the ADK pool.
- Skills behavior differs by mode; Plan mode explicitly does not pass `SkillsDir`.
- Missing skills directory silently disables skills unless strict mode is enabled.
- Mode pool caches one Agent instance per mode, requiring confidence that underlying Eino agents are reusable/concurrency-safe.

## Files Of Interest

- `aiapp/aisolo/aisolo.proto`
- `aiapp/aisolo/internal/svc/servicecontext.go`
- `aiapp/aisolo/internal/turn/executor.go`
- `aiapp/aisolo/internal/modes/*.go`
- `aiapp/aisolo/internal/logic/*streamlogic.go`
- `aiapp/aisolo/internal/session/*`
