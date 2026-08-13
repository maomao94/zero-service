# Research: Known Unknowns and Negative Searches

- **Query**: State unresolved questions and searches with no findings
- **Scope**: internal
- **Date**: 2026-08-14

## Findings

### Known Unknowns

| Unknown | Why source cannot answer it |
|---|---|
| Per-RPC and per-MCP-tool need for user delegation | No policy registry or method annotations exist; product/security intent is not encoded. |
| Internal gRPC peer authentication and encryption | No complete deployment mesh/TLS/mTLS configuration was located in the inspected code paths. |
| External MCP server ownership and data-handling guarantees | Server URLs/config values do not establish organizational trust or retention. |
| Runtime log exposure | Effective log level, sink ACL, retention, sampling, and redaction are deployment properties. |
| MCP SDK retention/serialization internals | SDK source was not vendored in the repository paths inspected. |
| `cryptor.Base64StdDecode` invalid-input semantics | Helper implementation is external; current code ignores errors because API returns string. |
| Which Socket metadata keys are configured in every deployment | `SocketMetaData` is config-driven; production values were not established. |
| Whether any out-of-repository service revalidates forwarded user tokens | This repository cannot prove external receiver behavior. |

### Searches With No Findings

- No application code persisted raw Authorization, authctx Authorization, or MCP raw `_meta` to SQL/DB models, Redis/cache, files, or event-bus payloads.
- No application metric label/value containing Authorization or bearer token was found.
- No trace span attribute or baggage write containing raw Authorization or the full MCP `_meta` map was found.
- No application business caller of `mcpx.GetMeta` was found outside tests; raw meta is stored as an extension point.
- No generic gRPC receiver JWT parsing/revalidation was found in `common/grpcx` or its server interceptors.
- No conflict reconciliation between Authorization-derived claims and `x-user-id`/`x-user-name`/`x-dept-code`/`x-auth-type` was found.
- No business authorization/data-isolation use of `auth-type` was found.
- No confirmed authctx-based authorization use of `dept-code` was found.
- No direct incoming `metadata.MD` Authorization reader elsewhere in repository Go code was found.
- No raw HTTP Authorization forwarding from the generic `gtw` middleware was found; unlike AI gateway, it only sets `auth-type` globally (`gtw/gtw.go:57-63`).

### False Positives Excluded

- Provider API-key Authorization in `aiapp/aichat/internal/provider/openai.go:99` and `common/einox/knowledge/embedder.go:96` is service credential use, not user-token propagation.
- MCP `progress token` logs (`common/mcpx/wrapper.go:68-151`, `client.go:380,831-840`) are correlation tokens, not Authorization based on construction at `client.go:808-829`.
- Generated protobuf `GetUserId`/`GetDeptCode` methods are data-field accessors, not auth context reads.
- Test fixtures with `Bearer test-token` document behavior but are not runtime leakage, except insofar as tests lock propagation contracts.

### Search Coverage

Inspected categories included Go sources, YAML/config, proto/generated symbol occurrences, SQL-like/persistence keywords, logging/formatting, tracing carriers, grpcx/mcpx/authctx tests, gateway routes/middleware, Socket.IO handlers, direct consumers, parent/child PRDs, specs, and archived extraction PRD.

## Caveats / Not Found

Negative findings are repository-source findings, not proof about runtime infrastructure, dependencies, or services maintained elsewhere.
