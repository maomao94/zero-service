# aigtw Integration Findings

Source: direct inspection after background exploration failed.

## Summary

`aiapp/aigtw` is the HTTP/SSE gateway for Solo. It exposes REST/SSE routes from `aiapp/aigtw/doc/aigtw.api`, forwards session and stream operations to `aisolo` through `AiSoloCli`, and directly owns HTTP knowledge-base CRUD/query through `common/einox/knowledge`.

## API Surface

- `aiapp/aigtw/doc/aigtw.api` defines `solo/v1` routes for modes, skills, sessions, messages, interrupt lookup/resume, chat SSE, and knowledge CRUD/query.
- `aiapp/aigtw/internal/handler/routes.go` wires generated routes and marks chat/resume as `rest.WithSSE()`.
- `aiapp/aigtw/internal/handler/solo/chathandler.go` and `resumehandler.go` are hand-written custom SSE handlers and explicitly marked `DO NOT re-generate`.

## Gateway To RPC Mapping

- `createsessionlogic.go` maps JWT user id plus HTTP fields to `aisolo.CreateSessionReq`, including mode, UI language, and knowledge-base binding fields.
- `bindsessionknowledgelogic.go` maps `POST /sessions/:sessionId/knowledge` to `aisolo.BindKnowledgeBaseReq`.
- `chatlogic.go` maps `SoloChatRequest` to `aisolo.AskReq`, trims fields, parses mode through `modeweb.Parse`, and streams `AskStream` chunks as SSE `data: <json>\n\n`.
- `resumelogic.go` maps `SoloInterruptRequest` to `aisolo.ResumeReq`, parses `yes/no`, forwards resume payloads, and streams `ResumeStream` chunks as SSE.
- List/get/delete session, messages, modes, skills, and interrupt lookup are thin RPC adapters.

## SSE Behavior

- Request validation happens before SSE headers are written.
- After headers are opened, stream errors are logged by handlers but cannot be returned as HTTP errors.
- `chatlogic.go` and `resumelogic.go` trim trailing CR/LF from `aisolo` NDJSON frames before wrapping as SSE data frames.
- Empty chunks and nil chunks are skipped.
- Streaming returns after the first `is_final` chunk and ignores later chunks.
- Tests in `chat_resume_validation_test.go` cover validation-before-stream, SSE formatting, field trimming, mode parsing, final-frame stop, and resume payload forwarding.

## Knowledge Boundary

`aigtw` initializes its own `common/einox/knowledge.Service` in `internal/svc/servicecontext.go`, while `aisolo` also initializes knowledge for Agent retrieval. The config comment says it must be consistent with `aisolo` data dir when shared.

This means knowledge has two service instances across gateway and agent service. That can be acceptable if backed by the same store, but the design must document config parity and health visibility. It is a cross-service consistency risk if one side is disabled/misconfigured or points to different storage.

## Tests Present

- `internal/logic/solo/chat_resume_validation_test.go`: chat/resume request validation and SSE forwarding.
- `internal/handler/solo/streamhandler_test.go`: rejects invalid requests before SSE headers.
- `internal/logic/solo/session_validation_test.go`: session/message validation and RPC mapping.
- `internal/logic/solo/knowledge_validation_test.go`: knowledge validation before SDK calls.
- `internal/svc/servicecontext_test.go`: dependency and knowledge state reporting.
- `internal/handler/solo/metahandler_test.go`: meta endpoint reports knowledge misconfiguration.

## Top Risks

1. Post-header stream errors are only logged. If `aisolo` fails after SSE starts, clients may need an explicit protocol error event from upstream to observe failure reliably.
2. Knowledge service is initialized in both `aigtw` and `aisolo`; config drift can make gateway CRUD/query and Agent retrieval see different data.
3. Custom SSE handlers are hand-written and marked not generated, so `gen.sh` may require careful diff review to preserve them.
4. Gateway strips empty chunks; if upstream later uses empty final/error/control frames, the gateway may hide them.
5. Mode parsing defaults or unknown-mode behavior depends on `modeweb.Parse`; gateway and `aisolo` proto semantics must remain aligned.
