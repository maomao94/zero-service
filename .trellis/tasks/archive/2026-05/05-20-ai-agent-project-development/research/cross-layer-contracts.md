# Cross-Layer Contract Findings

Source: direct inspection after background exploration failed, plus prior `aisolo` and `einox` findings.

## Core Flows

### Create Session

```text
aigtw /solo/v1/sessions
  -> SoloCreateSessionRequest
  -> CreateSessionLogic
  -> aisolo.CreateSessionReq
  -> aisolo CreateSessionLogic
  -> session.Store
```

Key shared fields: `mode`, `ui_lang`, `knowledge_base_id`, `knowledge_base_name`.

### Chat Stream

```text
aigtw /solo/v1/chat SSE
  -> ValidateChatRequest before headers
  -> aisolo.AskStream
  -> turn.Executor.Ask
  -> common/einox ADK/runtime execution
  -> protocol.Event JSON
  -> aisolo AskStreamChunk.data
  -> aigtw SSE data frame
```

Key risk: `aisolo` can currently choose custom `RuntimeRunner` for default Agent Ask while Resume uses ADK. This is the largest cross-layer semantic mismatch.

### Resume Interrupt

```text
aigtw /solo/v1/interrupt/:interruptId/resume SSE
  -> ValidateResumeRequest before headers
  -> aisolo.ResumeStream
  -> turn.Executor.Resume
  -> ADK runner Resume with checkpoint/session id
  -> protocol.Event JSON
  -> aigtw SSE data frame
```

Key contract: interrupt/resume requires ADK checkpoint semantics. This supports ADK-primary execution for session Agent flows.

### Knowledge

```text
aigtw knowledge CRUD/query
  -> common/einox/knowledge.Service in aigtw

aisolo session knowledge binding
  -> session metadata
  -> turn context
  -> common/einox knowledge search tool/retriever in aisolo
```

Key risk: gateway knowledge operations and Agent retrieval are separate service instances. They must share backend config and expose misconfiguration clearly.

### Modes / Skills

```text
aigtw ListModes/ListSkills
  -> aisolo RPC
  -> modes.Registry / filesystem SKILL.md scan
  -> HTTP response mapping
```

Key risk: UI may show skills globally, while some modes such as Plan may not load skills. Mode capability metadata should avoid over-promising.

## Contract Mismatches To Track

- ADK vs runtime execution path for default Agent Ask/Resume.
- Knowledge config parity between `aigtw` and `aisolo`.
- Skill availability by mode vs globally listed skills.
- SSE error visibility after HTTP headers are opened.
- Hand-written SSE handlers vs go-zero generation boundaries.

## Recommended Cross-Layer Gate

Before implementation completion, run tests that prove:

- invalid chat/resume requests fail before SSE headers;
- valid chat/resume forward exact JSON events from `aisolo` without re-marshalling;
- default Agent Ask/Resume use compatible execution semantics;
- knowledge disabled/misconfigured state is visible in both services;
- generated-code workflows do not overwrite custom SSE handlers without review.
