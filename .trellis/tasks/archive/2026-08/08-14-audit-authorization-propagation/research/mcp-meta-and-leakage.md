# Research: MCP Meta, Raw Meta, and Token Leakage

- **Query**: Audit MCP `_meta`, raw meta, external boundaries, logs/errors/traces/persistence, and token leakage
- **Scope**: internal
- **Date**: 2026-08-14

## Findings

### MCP Boundary and Lifetime

1. MCP HTTP transport always sets configured service credentials as `Authorization: Bearer <serviceToken>` (`common/mcpx/client.go:1186-1201`). This authenticates the client/service connection.
2. For each tool call, `CollectFromCtx` copies every non-empty auth context string, including raw user Authorization, into a newly allocated map (`common/mcpx/context_meta.go:12-23`). Trace injection adds W3C fields, then `params.SetMeta(meta)` serializes it across the external MCP tool/server boundary (`common/mcpx/client.go:774-796,825-827`).
3. Server wrapper reads `req.Params.GetMeta()`, extracts trace, stores the original map unchanged in context, and optionally copies auth fields to string context keys (`common/mcpx/wrapper.go:231-250`).
4. `WithMeta` retains the map for the request context lifetime; no clone, redaction, or expiry exists (`common/mcpx/context_meta.go:39-49`). The SDK/request/session may retain params longer, but that lifetime is external-library behavior not established here.
5. Tools registered with `WithExtractUserCtx` include echo, progress, and Modbus tools (`aiapp/mcpserver/internal/tools/echo.go:44`, `testprogress.go:89`, `modbus.go:52,75`). Their nested gRPC clients install metadata interceptors (`aiapp/mcpserver/internal/svc/servicecontext.go:30-35`), so extracted raw Authorization can cross MCP then gRPC.

### Direct Token Exposure Findings

| Sink | Evidence | Exposure |
|---|---|---|
| StreamEvent info log | `facade/streamevent/internal/logic/upsocketmessagelogic.go:30-31` | Logs full Socket.IO JWT/token after gRPC propagation at info level. |
| MCP echo debug log | `aiapp/mcpserver/internal/tools/echo.go:25-28` | Logs full raw Authorization from `_meta`/tool context. |
| MCP auth debug log | `common/mcpx/auth.go:45-65` | `extra` contains raw token at line 61; line 65 formats the whole map with `%v`. |
| MCP `_meta` wire serialization | `common/mcpx/context_meta.go:12-23`; `client.go:787-790` | Sends raw caller token to the configured MCP server/tool boundary in request JSON metadata. |
| gRPC metadata | `common/grpcx/metadata.go:27-59` | Sends raw token in request metadata to every intercepted downstream call. |

Progress logs use a field named `token`, but the value is an MCP progress token/correlation identifier, not Authorization (`common/mcpx/wrapper.go:68-151`; `client.go:380,831-840`). It remains potentially sensitive operational metadata but is not a bearer credential based on the code path.

### Errors and Traces

- gRPC server interceptors log returned errors only (`common/grpcx/server_interceptor.go:12-29`); no explicit token formatting exists there. Whether `logx.WithContext` serializes arbitrary context values was not established from repository code.
- MCP call errors include tool name, marshalled tool arguments, and error (`common/mcpx/wrapper.go:215-229`). `_meta` is not part of `args` in this wrapper, so raw token is not directly included by this formatting path.
- MCP client errors format server/tool names but not `_meta` (`common/mcpx/client.go:791-794`).
- Trace carrier injects/extracts values from the same `_meta` map (`common/mcpx/context_meta.go:52-58`, `common/trace/carrier.go:51`). No code attaches the full map or Authorization to spans/attributes.
- No token-valued metric label was found.

### Serialization and Persistence

- The MCP request serializer necessarily serializes `_meta` for transport; this is the primary raw-meta exposure boundary.
- Socket session identity metadata is serialized into transient `UpSocketMessageReq.Payload` JSON (`socketapp/socketgtw/internal/svc/servicecontext.go:75-89,100-134`). Session metadata comes from configured JWT claim keys (`common/socketiox/server.go:517-527`); raw token is stored separately in event context, not session metadata.
- Searches found no DB model, SQL, Redis/cache write, event publish payload, or file write containing Authorization/raw `_meta`.
- No application code calls `mcpx.GetMeta` outside tests; current wrapper stores raw meta for a documented extension point (`common/mcpx/wrapper.go:241-245`) rather than a present business consumer.

### Configuration Evidence

MCP server config defines service token/JWT secrets and claim mapping (`aiapp/mcpserver/etc/mcpserver.yaml:18-25`). AI gateway defines analogous claim mapping (`aiapp/aigtw/etc/aigtw.yaml:14-18`). Secret values may be overridden/environment-managed, but deployment manifests and actual runtime values were not established.

## Caveats / Not Found

- Production log level, sinks, access controls, retention, and redaction are unknown. Debug-only exposure remains source-confirmed even if disabled in a deployment.
- The MCP SDK's internal request/session retention, error formatting, and tracing are external-library behavior not audited from vendored source.
- No raw-meta persistence or business `GetMeta` consumer was found.
