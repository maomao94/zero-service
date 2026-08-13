# Research: MCP Raw `_meta` Policy Surface (P5 gating)

- **Query**: Where to gate `CollectFromCtx` sending raw Authorization into `_meta`; does it have an options param; would a policy require changing its signature or the client call site
- **Scope**: internal
- **Date**: 2026-08-14

## Findings

### Current behavior

- `common/mcpx/context_meta.go:12-24` `CollectFromCtx(ctx)` — **no options parameter**. Signature is `func CollectFromCtx(ctx context.Context) map[string]any`. It iterates `authctx.ContextKeys` (`common/authctx/context.go:18-24`, first key `authorization`), copies every non-empty string into a new map, returns nil if empty.
- Call sites (both inside `common/mcpx/client.go`):
  - `:788` `callTool` — `meta := CollectFromCtx(ctx); trace.Inject(ctx, trace.NewAnyCarrier(meta)); params.SetMeta(meta)`.
  - `:825` `callToolWithProgress` — identical pattern.
- All public client methods funnel through these two: `Client.CallTool` (`:346`), `Client.CallToolWithProgress` (`:369`), `Client.CallToolAsync` (`:961` → async `CallToolWithProgress`), `Client.CallToolAsyncAwait` (`:1094` → `CallToolWithProgress`).
- Repository callers: aichat (`chatcompletionlogic.go:95`, `chatcompletionstreamlogic.go:224`, `asynctoolcalllogic.go:28`), aisolo runtime (`common/einox/tool/mcp.go:45` via `MCPTool.InvokableRun`).

### Gating options (evidence for each)

**Option A — filter inside `CollectFromCtx` (change behavior for all callers).**
- Single chokepoint; changing the loop to skip `authctx.CtxAuthorizationKey` (or only include claims keys `user-id`/`user-name`/`dept-code`) would stop raw-token `_meta` propagation for **all** MCP clients (aichat + aisolo).
- Constraint: `common/mcpx/context_meta_test.go:25-29` (`TestMetaCollectExtractAndRawStorage`) asserts `CollectFromCtx` returns `{"authorization": "Bearer token", "user-id": "user-1"}` — **this test locks the raw-token inclusion** and must be updated with the policy change.
- Signature unchanged.

**Option B — add an options parameter to `CollectFromCtx`.**
- Signature change: `CollectFromCtx(ctx, opts ...CollectOption)`; the two call sites in `client.go` would need to pass per-connection or per-call policy. Since `callTool`/`callToolWithProgress` are `*Connection` methods and the `Connection` already carries `serviceToken` (`client.go:185`) and `cfg` (`client.go:186`), the policy could ride on `Connection`/`ServerConfig` (`common/mcpx/config.go:11-16`) without changing the public `Client.CallTool` API — only the two internal call sites change.
- This preserves per-server policy (each `ServerConfig` can declare claims-only vs none) — aligns with audit P5 "每个外部 MCP server owner 确认" (per-server granularity is required).

**Option C — gate at public API / per-call context flag.**
- Add a ctx flag or `CallTool`-level option read in `callTool`/`callToolWithProgress` before `CollectFromCtx`; most invasive, touches all callers (aichat/aisolo logic files). Not preferred; per-call policy for MCP is not needed today (no allowlist exists).

**Option D — receiver-side: stop restoring raw token.**
- `CallToolWrapper` (`common/mcpx/wrapper.go:247-251`) could skip restoring `CtxAuthorizationKey` from `_meta` (i.e. `ExtractFromMeta` filters it), and/or `ExtractFromMeta` (`context_meta.go:26-37`) could skip the `authorization` key. This blocks the nested-gRPC re-transmission path (P5/P6) even if a sender still sends raw `_meta`.
- Constraint: `context_meta_test.go:35-38` asserts `ExtractFromMeta` restores `authorization` — test lock.
- Combined with Option A/B this gives sender-side (no raw in `_meta`) + receiver-side (no restore) defense.

### Relationship to L3 and ServiceToken (P4)

- **P4 service-token layer is independent**: `common/mcpx/client.go:1192-1201` `ctxHeaderTransport.RoundTrip` sets `Authorization: Bearer <serviceToken>` for the HTTP connection; this is the configured service credential and must remain untouched by `_meta` policy.
- **L3 entanglement**: `common/mcpx/auth.go:61` `extra[authctx.CtxAuthorizationKey] = token` puts raw token into `TokenInfo.Extra`, which feeds the MCP SDK auth boundary and can reach tool context. `auth_test.go:46-48` locks this. Whether `Extra` should retain the raw token is a policy decision for this task's P5/P6 work — the L3 **log** fix (line 65) is separate (see `log-leakage-fixes.md`).
- `mcpx.ServerConfig.ServiceToken` (`config.go:14`) is the only per-server auth field today; a new `MetaMode`/`AuthPolicy` field on `ServerConfig` would sit naturally next to it.

## Caveats / Not Found

- No `_meta`-policy config exists; no per-tool allowlist; `GetMeta` has no business consumers (only test, `context_meta.go:44-50`).
- The MCP SDK's own `_meta` handling is external; repo can only gate at `CollectFromCtx`, `params.SetMeta` (client.go:790,827), and wrapper restore.
- Which exact shape (A/B/D or combination) is a design decision — the research only maps the chokepoints and test locks.
