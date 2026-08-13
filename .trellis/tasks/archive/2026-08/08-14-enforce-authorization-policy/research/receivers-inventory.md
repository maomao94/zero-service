# Research: Receiver-side Extraction Inventory (duplicate/conflict detection surface)

- **Query**: Every gRPC server that installs `grpcx.LoggerInterceptor` / `StreamLoggerInterceptor` (which call `ExtractFromGrpcMD`) and the MCP `ExtractFromMeta` / `WithExtractUserCtx` restore path
- **Scope**: internal
- **Date**: 2026-08-14

## Findings

### Extraction mechanism (shared)

- `common/grpcx/server_interceptor.go:12-19` `LoggerInterceptor` — unary: `ctx = ExtractFromGrpcMD(ctx)` then `handler(ctx, req)`; logs error via `logx.WithContext(ctx).Errorf("rpc error: %+v", err)` (no metadata content).
- `common/grpcx/server_interceptor.go:24-31` `StreamLoggerInterceptor` — stream: `ctx := ExtractFromGrpcMD(ss.Context())`, wraps stream so `Context()` returns enriched ctx (`wrappedStream`, `:34-41`).
- `common/grpcx/metadata.go:62-75` `ExtractFromGrpcMD` — reads `metadata.FromIncomingContext`; for each `metadataField` takes `values[0]` when `len>0 && values[0] != ""`; b64-decodes `b64:` prefix; writes via `authctx.WithKey`. No verification, no duplicate handling (see `duplicate-conflict-surface.md`).

### gRPC receiver installation sites

23 interceptor registrations across 21 unique gRPC servers (files in `main` packages; note the task brief said "26 sites" — actual in-repo count is 23 lines / 21 files):

| # | Service (file:line) | Unary | Stream | Receives raw-token? (by sender inventory) |
|---|---|---|---|---|
| R1 | `zerorpc/zerorpc.go:44` | ✅ | — | P2 (gtw claims) / P7 possible |
| R2 | `facade/streamevent/streamevent.go:65` | ✅ | — | **P3 socket raw token** (L1 log), P7 |
| R3 | `socketapp/socketpush/socketpush.go:63` | ✅ | — | P7 |
| R4 | `app/lalproxy/lalproxy.go:65` | ✅ | — | P7 |
| R5 | `app/trigger/trigger.go:72` | ✅ | — | P7 (also re-emits via S6-S8) |
| R6 | `app/bridgekafka/bridgekafka.go:82` | ✅ | — | P7 |
| R7 | `app/gis/gis.go:68` | ✅ | — | P7 |
| R8 | `socketapp/socketgtw/socketgtw.go:92` | ✅ | — | P7 (pub container S17) |
| R9 | `app/ispserver/ispserver.go:39` | ✅ | — | P7 |
| R10 | `app/ieccaller/ieccaller.go:80` | ✅ | — | P7 |
| R11 | `app/iecstash/iecstash.go:73` | ✅ | — | P7 |
| R12 | `app/bridgemqtt/bridgemqtt.go:65` | ✅ | — | P7 |
| R13 | `app/bridgemodbus/bridgemodbus.go:63` | ✅ | — | **P5/P6 via mcpserver nested client** (modbus tool) |
| R14 | `app/logdump/logdump.go:64` | ✅ | — | P7 |
| R15 | `app/ispagent/ispagent.go:39` | ✅ | — | P7 |
| R16 | `aiapp/aichat/aichat.go:41,42` | ✅ | ✅ | **P1 (aigtw S1)** — chat + stream |
| R17 | `app/file/file.go:65` | ✅ | — | P2 (gtw S4) |
| R18 | `app/podengine/podengine.go:63` | ✅ | — | P7 |
| R19 | `app/iecagent/iecagent.go:48` | ✅ | — | P7 |
| R20 | `aiapp/aisolo/aisolo.go:48,49` | ✅ | ✅ | **P1 (aigtw S2)** — data isolation |
| R21 | `app/djicloud/djicloud.go:65` | ✅ | — | P7 |

All 21 servers therefore run `ExtractFromGrpcMD` on every incoming call; any of them would be the enforcement point for receiver-side duplicate/conflict policy. The audit's P7 (`gRPC incoming → nested gRPC outgoing`) is generic — every server above is both a receiver (via its interceptor) and potentially a sender (if its client interceptors are installed and ctx retains auth values).

### MCP receiver restore path

- `common/mcpx/wrapper.go:215-269` `CallToolWrapper`:
  - `:234-239` reads `req.Params.GetMeta()`, extracts trace (`ExtractTraceFromMeta`).
  - `:241-245` `WithMeta(ctx, meta)` — stores the **raw `_meta` map unmodified** in ctx (no clone, no redaction).
  - `:247-251` when `cfg.extractUserCtx` (`WithExtractUserCtx` option, `:176-180`) and meta non-empty: `ctx = ExtractFromMeta(ctx, meta)`.
- `common/mcpx/context_meta.go:26-37` `ExtractFromMeta` — for each `authctx.ContextKeys` reads `authctx.ClaimString(meta, key)` (ClaimString → `normalizeClaimString`), writes via `authctx.WithKey`. **Restores raw Authorization from `_meta` into ctx** (key `authorization`, first in `ContextKeys`).
- Tools registered with `WithExtractUserCtx` (the only ones able to restore raw token):
  - `aiapp/mcpserver/internal/tools/echo.go:44` — echo (L2 log site)
  - `aiapp/mcpserver/internal/tools/testprogress.go:89` — progress
  - `aiapp/mcpserver/internal/tools/modbus.go:52,75` — modbus read tools (nested gRPC S9)
- MCP HTTP auth boundary (`NewDualTokenVerifier`, `common/mcpx/auth.go:22-72`) is a separate receiver surface: service-token constant-time compare (`:24-32`) or JWT parse (`:34-67`). It does **not** validate `_meta`; the wrapper restores `_meta` content unconditionally for `WithExtractUserCtx` tools.

### Observability insertion points (content-free)

- Duplicate/conflict signal for gRPC: single chokepoint `ExtractFromGrpcMD` (`common/grpcx/metadata.go:62-75`) — called by both unary and stream interceptors; adding report-only observability here covers all 21 servers.
- Duplicate/conflict signal for MCP: `ExtractFromMeta` (`common/mcpx/context_meta.go:26-37`) and/or `CallToolWrapper` meta handling (`common/mcpx/wrapper.go:241-251`).

## Caveats / Not Found

- Actual site count is 23 interceptor registration lines in 21 service files; the "26" in the task brief does not match the grep evidence (no other `AddUnaryInterceptors`/`AddStreamInterceptors` sites exist).
- Receiver-side verification (JWT re-verify, claims reconciliation) does not exist today; `ExtractFromGrpcMD` and `ExtractFromMeta` are pure extraction.
- MCP wrapper has no per-tool policy registry; suppression would apply to all `WithExtractUserCtx` tools or require new per-tool options.
