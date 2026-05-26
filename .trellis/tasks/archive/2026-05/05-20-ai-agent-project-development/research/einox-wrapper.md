# einox Wrapper Findings

Source: Explore task `bg_4bba43ed`.

## Summary

`common/einox` is a mixed abstraction layer over CloudWeGo Eino. It contains a thin ADK wrapper and a separate custom runtime runner. This dual architecture is the main foundation risk because tool calling, streaming, event emission, RAG behavior, metrics, checkpoint, and interrupt semantics can diverge.

## High-Risk Findings

- `WithModelOption` appears unused by agent creation.
- `model/chatmodel.go` and `model/chatmodel_option.go` duplicate provider factory APIs.
- ADK execution and custom runtime execution coexist without a clearly documented boundary.
- `tool.Kit` / `tool.Policy` and `runtime.ToolRegistry` overlap and can diverge.
- `Knowledge.ChunkOverlapRunes` config is defined but chunking does not apply overlap.
- `knowledge.NewSearchTool(nil)` and disabled `knowledge.NewService` return nil/nil, requiring all callers to handle disabled state carefully.
- Runtime streaming and ADK protocol adapter have different behavior for tool-call-only assistant chunks.
- Protocol adapter logs some stream receive errors without propagating them to callers.
- Test coverage is uneven for agent factories, protocol adapter, interrupt/resume middleware, model factories, memory/checkpoint backends, and metrics.

## Prioritized Remediation

1. Decide ADK-primary vs documented lite-runtime boundary.
2. Fix/remove unused `WithModelOption`.
3. Consolidate or clearly deprecate duplicate chat model factories.
4. Implement or remove chunk overlap config.
5. Add protocol adapter tests for stream errors, tool-call-only streams, interrupts, and `RunResult`.
6. Unify or document tool policy/registry flow.
7. Add ADK agent factory and interrupt/resume middleware tests before refactoring.
