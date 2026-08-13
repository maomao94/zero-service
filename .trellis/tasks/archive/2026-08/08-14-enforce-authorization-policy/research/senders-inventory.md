# Research: Sender-side Injection Inventory (raw-token propagation surface)

- **Query**: Every site that installs `grpcx.UnaryMetadataInterceptor` / `StreamTracingInterceptor` (client side) and every MCP `CollectFromCtx` call; per-site which metadata fields are sent, whether raw Authorization is emitted, and whether a per-call suppression option exists
- **Scope**: internal
- **Date**: 2026-08-14

## Findings

### Injection mechanism (shared)

- `common/grpcx/metadata.go:45-60` `InjectToGrpcMD`: copies outgoing MD, iterates `metadataFields` (order: `authorization`, `x-user-id`, `x-user-name`, `x-dept-code`, `x-auth-type` — lines 27-34), reads each value via `authctx.GetByKey`, **skips empty strings**, b64-encodes non-printable values with `b64:` prefix, then `md.Set(f.grpcKey, str)` — **overwrites the whole key** with context value.
- `common/grpcx/client_interceptor.go:11-14` `UnaryMetadataInterceptor` and `:18-21` `StreamTracingInterceptor` are thin wrappers calling `InjectToGrpcMD(ctx)`. **No option / no per-call suppression parameter exists.** Any call with `authctx.WithAuthorization` in context will emit raw `authorization` metadata.
- **Consequence for P1/P3/P5:** because the interceptors are the only client-side gate, a policy that stops raw-token propagation must either (a) filter inside `InjectToGrpcMD` (affects all senders), or (b) add a per-call/per-client option (new mechanism), or (c) stop writing `authorization` into context at the source (per-site, brittle). No option (b) exists today.

### Sender installation sites (gRPC client interceptors)

| # | Site | Targets | Interceptor | Raw Authorization emitted? |
|---|---|---|---|---|
| S1 | `aiapp/aigtw/internal/svc/servicecontext.go:70-71` | aichat RPC | unary + stream | **Yes** — global middleware writes raw header into context (`aiapp/aigtw/aigtw.go:82-84`), then claims bridge `:90-97`; `InjectToGrpcMD` propagates it. P1. |
| S2 | `aiapp/aigtw/internal/svc/servicecontext.go:73-74` | aisolo RPC | unary + stream | **Yes** — same context. P1. |
| S3 | `gtw/internal/svc/servicecontext.go:60` | zerorpc RPC | unary | Claims only — gtw middleware writes only `auth-type=user` (`gtw/gtw.go:57-62`), no raw header into context. Raw auth absent from context, so not emitted. P2. |
| S4 | `gtw/internal/svc/servicecontext.go:62` | file RPC | unary | Claims only — same as S3. P2. |
| S5 | `socketapp/socketgtw/internal/svc/servicecontext.go:26` | streamevent RPC | unary | **Yes** — socketiox writes raw token into event ctx (`common/socketiox/server.go:537,558,579,594,610,673,698,730,754`), forwarded by `UpSocketMessage` calls `:84,109,129`. P3. |
| S6 | `app/trigger/internal/svc/servicecontext.go:84` | streamevent RPC | unary | Depends on ctx — trigger is both gRPC receiver and HTTP; if incoming metadata restored `authorization` (`ExtractFromGrpcMD`), it will re-emit. P7. |
| S7 | `app/trigger/internal/invoke/grpc_invoker.go:35` | dynamic target (task.GrpcServer) | unary | Depends on ctx — dynamic invoke from trigger ctx; raw codec `invoke_raw`. P7. |
| S8 | `app/trigger/internal/task/deferTriggerProtoTask.go:57` | dynamic target (asynq task) | unary | Depends on ctx — asynq handler ctx; raw codec `proto_raw`. P7. |
| S9 | `aiapp/mcpserver/internal/svc/servicecontext.go:33-34` | bridgemodbus RPC | unary + stream | **Yes** — MCP wrapper `WithExtractUserCtx` restores `authorization` from `_meta` (`common/mcpx/wrapper.go:247-251`), then nested client re-emits. Echo/modbus/progress tools use this. P5/P6/P7. |
| S10 | `aiapp/ssegtw/internal/svc/servicecontext.go:34` | zerorpc RPC | unary | Claims only — ssegtw HTTP ctx; no raw header writer found in ssegtw. P2-like. |
| S11 | `app/iecstash/internal/svc/servicecontext.go:27` | streamevent RPC | unary | Not found — background goroutine uses `context.Background()` (`:76`); no auth values in ctx. Practically no emission. |
| S12 | `app/ieccaller/internal/svc/servicecontext.go:88` | streamevent RPC | unary | Not found — `context.Background()` + `StartForwardSpan` (`:137`); no auth values. |
| S13 | `app/bridgekafka/internal/svc/servicecontext.go:31` | streamevent RPC | unary | Not found — background consumer; no auth values. |
| S14 | `app/bridgemqtt/internal/svc/servicecontext.go:27` | streamevent RPC | unary | Not found — MQTT handler; no auth values. |
| S15 | `app/bridgemqtt/internal/svc/servicecontext.go:38` | socketpush RPC | unary | Not found — MQTT handler; no auth values. |
| S16 | `app/djicloud/internal/svc/servicecontext.go:74` | socketpush RPC | unary | Not found — DJI SDK hooks with `context.WithoutCancel`; no auth values. |
| S17 | `common/socketiox/container.go:86` | socketgtw RPC (pub container, used by `socketapp/socketpush`) | unary | Not found — pub container calls from service ctx without auth values. |

Count: 17 install locations, 19 interceptor registrations (aigtw and mcpserver and bridgemqtt register 2).

### MCP `_meta` sender surface (P5)

- `common/mcpx/context_meta.go:12-24` `CollectFromCtx`: builds map of all non-empty `authctx.ContextKeys` — **includes raw Authorization** (`authorization` is first key, `authctx/context.go:18-24`). **No options parameter.** Returns nil when empty.
- Call sites:
  - `common/mcpx/client.go:788` in `callTool` — `meta := CollectFromCtx(ctx); trace.Inject(...); params.SetMeta(meta)`.
  - `common/mcpx/client.go:825` in `callToolWithProgress` — same pattern.
- Public entry points that reach these: `Client.CallTool` (`client.go:346`), `Client.CallToolWithProgress` (`client.go:369`), `Client.CallToolAsync` (`client.go:961` → async `CallToolWithProgress`), `Client.CallToolAsyncAwait` (`client.go:~1094` → `CallToolWithProgress`). All funnel through `callTool`/`callToolWithProgress`; none accepts a policy option.
- Repository callers of the public API:
  - `aiapp/aichat/internal/logic/chatcompletionlogic.go:95` — `CallToolWithProgress` (sync tool loop).
  - `aiapp/aichat/internal/logic/chatcompletionstreamlogic.go:224` — `CallToolAsyncAwait`.
  - `aiapp/aichat/internal/logic/asynctoolcalllogic.go:28` — `McpClient` async tool result.
  - `common/einox/tool/mcp.go:45` — `MCPTool.InvokableRun` → `CallTool` (aisolo runtime wrapper; wired at `aiapp/aisolo/internal/svc/servicecontext.go:272`).
- Service-token layer is separate: `common/mcpx/client.go:1192-1201` `ctxHeaderTransport.RoundTrip` sets `Authorization: Bearer <serviceToken>` on every MCP HTTP request (P4). Raw user token travels in `_meta` **in addition** to this.

### Existing per-call suppression mechanisms

- None in `common/grpcx` (interceptors have no options; no call-option filter). Grep for `CallOption|WithDefaultCallOptions|ForceCodec` in grpcx/gtwx shows only the raw-codec dial options used in trigger.
- None in `common/mcpx` public API. `ServerConfig` (`common/mcpx/config.go:11-16`) has `ServiceToken`/`UseStreamable`/`Endpoint` but no meta policy.
- No method-level policy registry anywhere (grep for `MethodPolicy|PolicyMode|Allowlist|DelegationMode` returned zero).

## Caveats / Not Found

- Whether any specific RPC actually has a context containing `authorization` depends on runtime ctx — senders S3/S4/S10-S17 have no raw-write path found in-repo, but the interceptor would propagate if such a value existed in ctx.
- No per-call option exists today; introducing one is a new mechanism, not a refactor.
