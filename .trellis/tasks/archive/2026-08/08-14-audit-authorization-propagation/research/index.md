# Research: Authorization Propagation Audit Index

- **Query**: Evidence-backed end-to-end audit of Authorization and identity propagation
- **Scope**: internal
- **Date**: 2026-08-14

## Findings

This research set covers the active audit PRD, parent hardening PRD, current implementation/tests/config, and archived `08-13-extract-grpc-raw-codec` contract.

| File | Contents |
|---|---|
| `propagation-matrix.md` | Raw Authorization sources, verification, context writes, transport hops, destinations, trust, and candidate future modes |
| `identity-claims-matrix.md` | `user-id`, `user-name`, `dept-code`, and `auth-type` source/mapping/consumer inventory |
| `grpc-metadata-conflicts.md` | Duplicate/conflicting metadata, `Set`, first-value, empty-first, and `b64:` behavior |
| `mcp-meta-and-leakage.md` | MCP HTTP auth, `_meta`, raw meta lifetime/boundary, and token leakage search |
| `policy-migration-inputs.md` | Policy options requiring approval, receiver-first sequence, observability, rollback, and child-task boundaries |
| `unknowns-and-negative-searches.md` | Known unknowns and explicit searches with no findings |

## Executive Evidence

- `common/grpcx/metadata.go:27-34,45-74` propagates raw Authorization and four identity values on every client using the interceptor, without receiver-side token verification.
- `common/mcpx/client.go:774-796,825-827,1192-1201` uses a service token for MCP HTTP authentication while separately placing the caller's raw Authorization into per-call `_meta`.
- `common/socketiox/server.go:496-537` verifies a Socket.IO handshake token when configured, then writes the same raw token into event contexts; `socketapp/socketgtw/internal/svc/servicecontext.go:24-33,75-89` forwards it to StreamEvent gRPC.
- Confirmed raw-token logs exist at `facade/streamevent/internal/logic/upsocketmessagelogic.go:30-31`, `aiapp/mcpserver/internal/tools/echo.go:25-28`, and indirectly through the full `extra` map at `common/mcpx/auth.go:45-65`.
- No code-level DB/cache/event persistence sink for Authorization or MCP raw `_meta` was found. Socket session metadata is serialized into transient gRPC request payloads (`socketapp/socketgtw/internal/svc/servicecontext.go:75-89,100-134`), but raw Authorization is not added to session metadata by the shown code.

## Critical Decision

The highest-value unresolved security/product decision is which calls require end-user delegation. Today raw user credentials are generically forwarded over gRPC and MCP `_meta`; narrowing this safely requires an approved per-boundary allowlist separating `user-token`, `claims-only`, `service-token`, and `none`.

## Related Specs

- `.trellis/tasks/08-14-auth-context-hardening/prd.md:9-20` mandates four separate, approval-gated tasks.
- `.trellis/tasks/archive/2026-08/08-13-extract-grpc-raw-codec/prd.md:65-80` records preserved wire keys, `b64:`, first-value, client overwrite, and propagation behavior.
- `.trellis/spec/backend/common-package-design.md` defines common-package ownership context.
- `.trellis/spec/guides/cross-layer-thinking-guide.md` is relevant to end-to-end data-flow verification.

## Caveats / Not Found

- Runtime topology, TLS/mTLS, proxy/header policies, production log level/retention, and downstream services outside this repository are not observable from source.
- Candidate future modes in the matrix are classification inputs, not policy decisions.
