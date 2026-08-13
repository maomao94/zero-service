# Research: Transport boundaries (gRPC + MCP)

- **Query**: `common/grpcx/metadata.go` (inject/extract, order, b64), interceptors, `common/mcpx/context_meta.go` (CollectFromCtx, extraction, WithMeta, raw meta), `common/mcpx/wrapper.go` (restore to context keys). Confirm which keys are wire strings vs process keys.
- **Scope**: internal
- **Date**: 2026-08-14

## Findings

### gRPC wire keys (`common/grpcx/metadata.go:13-20`)

```go
const (
	HeaderUserId        = "x-user-id"
	HeaderUserName      = "x-user-name"
	HeaderDeptCode      = "x-dept-code"
	HeaderAuthorization = "authorization"
	HeaderAuthType      = "x-auth-type"
	base64Prefix        = "b64:"
)
```

### Field order (wire contract — must stay)

`metadata.go:22-34` — `metadataField{contextKey, grpcKey}` pairs:

| Order | contextKey (process) | grpcKey (wire) |
|---|---|---|
| 1 | `authctx.CtxAuthorizationKey` (`authorization`) | `HeaderAuthorization` (`authorization`) |
| 2 | `authctx.CtxUserIdKey` (`user-id`) | `HeaderUserId` (`x-user-id`) |
| 3 | `authctx.CtxUserNameKey` (`user-name`) | `HeaderUserName` (`x-user-name`) |
| 4 | `authctx.CtxDeptCodeKey` (`dept-code`) | `HeaderDeptCode` (`x-dept-code`) |
| 5 | `authctx.CtxAuthTypeKey` (`auth-type`) | `HeaderAuthType` (`x-auth-type`) |

### InjectToGrpcMD (`metadata.go:46-60`)

- Reads `ctx.Value(f.contextKey).(string)` for each field.
- **Skips** non-string and empty values.
- Non-printable (control chars / non-ASCII, `hasNotPrintable` line 36-43) → `b64:` + standard Base64.
- `md.Set(f.grpcKey, str)` overwrites the key; unlisted keys preserved (`md.Copy()` first).
- Returns `metadata.NewOutgoingContext(ctx, md)`.

### ExtractFromGrpcMD (`metadata.go:63-74`)

- Reads incoming metadata `md.Get(f.grpcKey)`; **takes `values[0]`** only (duplicates not compared/rejected).
- Empty first value → whole field skipped (later values ignored).
- `b64:` prefix → Base64 decode (decode errors silently pass raw string through — no error raised).
- Writes back to process context with `context.WithValue(ctx, f.contextKey, val)` — **process keys = authctx string constants**.
- Existing non-empty process values are NOT overwritten by absent/empty metadata (context outer values win by WithValue semantics only when the inner key is absent; `WithValue` here shadows on non-empty).

### Server interceptors (`common/grpcx/server_interceptor.go`)

- `LoggerInterceptor` (line 12-19): `ctx = ExtractFromGrpcMD(ctx)` then handler; logs errors.
- `StreamLoggerInterceptor` (line 24-31): wraps `grpc.ServerStream` with `wrappedStream` (line 34-40) overriding `Context()` to return extracted ctx.
- Installed on all gRPC servers (26 sites): `socketgtw.go:93`, `gis.go:68`, `file.go:65`, `socketpush.go:63`, `podengine.go:63`, `trigger.go:72`, `zerorpc.go:44`, `iecagent.go:48`, `ispserver.go:39`, `lalproxy.go:65`, `aichat.go:41-42` (unary+stream), `bridgekafka.go:82`, `ieccaller.go:80`, `aisolo.go:48-49` (unary+stream), `djicloud.go:65`, `bridgemqtt.go:65`, `logdump.go:64`, `bridgemodbus.go:63`, `streamevent.go:65`, `ispagent.go:39`, `iecstash.go:73`, `file.go`, etc.

### Client interceptors (`common/grpcx/client_interceptor.go`)

- `UnaryMetadataInterceptor` (line 11-14): `invoker(InjectToGrpcMD(ctx), ...)`.
- `StreamTracingInterceptor` (line 18-21): `streamer(InjectToGrpcMD(ctx), ...)`.
- Installed at (23+ sites): `gtw/internal/svc/servicecontext.go:60,62`, `aigtw/internal/svc/servicecontext.go:70-74` (unary+stream, both clients), `socketgtw/internal/svc/servicecontext.go:26`, `mcpserver/internal/svc/servicecontext.go:33-34`, `ssegtw/internal/svc/servicecontext.go:34`, `iecstash/internal/svc/servicecontext.go:27`, `ieccaller/internal/svc/servicecontext.go:88`, `trigger/internal/svc/servicecontext.go:84`, `trigger/internal/invoke/grpc_invoker.go:35`, `trigger/internal/task/deferTriggerProtoTask.go:57`, `bridgekafka/internal/svc/servicecontext.go:31`, `djicloud/internal/svc/servicecontext.go:74`, `bridgemqtt/internal/svc/servicecontext.go:27,38`, `common/socketiox/container.go:86`.

### MCP `_meta` boundary (`common/mcpx/context_meta.go`)

- `ctxMetaKey = "_meta"` (line 10) — **private string constant**; `WithMeta`/`GetMeta` use it. Wire `_meta` map keys are the authctx string constants (`authorization`, `user-id`, `user-name`, `dept-code`, `auth-type`).
- `CollectFromCtx` (line 13-24): iterates `authctx.ContextKeys`; `ctx.Value(key).(string)` non-empty → `meta[key] = v`. Nil when empty.
- `ExtractFromMeta` (line 27-37): iterates `authctx.ContextKeys`; `authctx.ClaimString(meta, key)` non-empty → `context.WithValue(ctx, key, v)` (string keys restored).
- `WithMeta` (line 40-42) / `GetMeta` (line 45-50): stores/reads the **raw `_meta` map** under `ctxMetaKey`. No clone/redaction/expiry. **No business `GetMeta` consumer** (only tests; audit §5.1).
- `ExtractTraceFromMeta` (line 53-57): W3C trace via `trace.Extract(ctx, trace.NewAnyCarrier(meta))` — not auth.

### MCP wrapper (`common/mcpx/wrapper.go`)

`CallToolWrapper` (line 206-283):
- line 235-239: `req.Params.GetMeta()` → `ExtractTraceFromMeta`.
- line 243-245: `WithMeta(ctx, meta)` — stores raw `_meta` map in ctx for every tool call with meta.
- line 249-251: **opt-in** `WithExtractUserCtx()` → `ExtractFromMeta(ctx, meta)` restores authctx string keys.
- `WithExtractUserCtx` option (line 176-180).
- Enabled by tools: `echo.go:44`; audit also cites `testprogress.go:89`, `modbus.go:52,75`.

### MCP client (`common/mcpx/client.go`)

- `callTool` line 788 and `callToolWithProgress` line 825: `meta := CollectFromCtx(ctx)`; `trace.Inject`; `params.SetMeta(meta)` — serializes process ctx auth (incl. raw Authorization) to external MCP server (audit P5).
- Connection auth uses configured ServiceToken (audit P4, `client.go:1192-1201`).

### MCP auth (`common/mcpx/auth.go`)

`NewDualTokenVerifier` (line 22-72):
- Service-token match → `Extra = {"auth-type": "service"}` (line 30).
- JWT parse → `ApplyClaimMapping(claims, claimMapping)` (line 43); `Extra` = all `ContextKeys` claims + `auth-type=user` (line 46-52) + `authorization=raw token` (line 61) + `exp`.
- `TokenInfo.UserID = authctx.ClaimString(claims, authctx.CtxUserIdKey)` (line 58).
- Uses **string keys** throughout `Extra` map.

### Wire keys vs process keys — confirmation

| Key | Wire? | Process ctx? | Notes |
|---|---|---|---|
| `user-id` | yes (MCP `_meta`; JWT claim) | yes | gRPC wire is `x-user-id` |
| `user-name` | yes (MCP `_meta`; JWT claim) | yes | gRPC wire is `x-user-name` |
| `dept-code` | yes (MCP `_meta`; JWT claim) | yes | gRPC wire is `x-dept-code` |
| `authorization` | yes (gRPC wire = same name; MCP `_meta`) | yes | same string both sides |
| `auth-type` | yes (MCP `_meta`) | yes | gRPC wire is `x-auth-type` |
| `x-user-id`, `x-user-name`, `x-dept-code`, `x-auth-type` | gRPC only | no | never process keys |
| `b64:` | gRPC only | no | encoding marker, not a key |
| `_meta` | MCP params meta | via `ctxMetaKey` | raw map storage |

### Must-preserve behavior (audit §4.1)

- Outgoing: context wins over existing metadata for listed keys; unlisted keys preserved.
- Incoming: `values[0]` only; empty-first skips field; no duplicate comparison/rejection.
- Empty/non-string outgoing context → key not emitted, existing outgoing metadata preserved.
- Non-printable → `b64:`; decode errors silent.

### Related specs

- `.trellis/tasks/archive/2026-08/08-14-audit-authorization-propagation/audit-report.md` §4.1, §5.1, §8.1.
- `.trellis/tasks/archive/2026-08/08-14-audit-authorization-propagation/research/grpc-metadata-conflicts.md`.