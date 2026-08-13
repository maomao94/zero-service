# Research: Policy Options and Child-Task Inputs

- **Query**: Document target policy options, receiver-first migration, content-free observability, rollback, required decisions, and next-task constraints
- **Scope**: mixed (code evidence plus decision framing)
- **Date**: 2026-08-14

## Findings

The following are options for user/security approval, not selected policy.

### Target Mode Options

| Mode | Meaning | Evidence-based candidate use |
|---|---|---|
| `user-token` | Forward the original end-user credential; receiver validates it and derives identity | Only calls where downstream must perform end-user authorization/delegation. No repository-wide list exists. |
| `claims-only` | Forward normalized identity/tenant claims without bearer credential; trust service boundary | AI gateway ownership-scoped RPCs and most internal calls that already put user ID in request fields are candidates. |
| `service-token` | Authenticate service-to-service transport with a distinct credential; carry user identity separately if needed | Existing MCP HTTP connection already follows this pattern (`common/mcpx/client.go:1192-1201`). |
| `none` | Forward neither raw token nor user claims | Socket -> StreamEvent raw token is currently used only for logging, making it an evidence-based candidate. Background/infrastructure calls require per-call review. |
| `unresolved` | Insufficient product/security evidence | Generic gRPC interceptor installations and external MCP tools without documented delegation requirements. |

### Explicit Decisions Required

1. Which RPC methods and MCP tools are authorized to receive an end-user token?
2. Must downstream services independently validate user JWTs, or may they trust claims from authenticated internal callers?
3. What authenticates internal gRPC peers: network location, TLS/mTLS, service token, or another mechanism not in this repository?
4. For duplicate/conflicting metadata, should receivers reject, require equal duplicates, or select a canonical source? Does token-derived identity override `x-user-*`?
5. Is a service-authenticated MCP client permitted to assert user identity through `_meta`, and are external MCP servers permitted to receive raw user credentials?
6. Are `dept-code` and `auth-type` security attributes or informational attributes? What is mandatory when absent?
7. What compatibility window and rollback duration are required for mixed-version deployments?

### Receiver-First Migration Sequence

1. Inventory/approve a method/tool policy table and identify owners. Do not change senders yet.
2. Add receiver observability for presence, mode, duplicates, conflicts, validation outcome, and caller service, without recording token values.
3. Add receiver compatibility for both legacy raw-token metadata and target claims/service credentials. Define deterministic conflict handling in report-only mode first.
4. Enable receiver enforcement per service/method behind configuration or allowlist, retaining a legacy compatibility switch.
5. Change senders by narrow route/tool cohorts: stop raw token for `none`/`claims-only`, add service credentials where approved, and retain user token only on delegated paths.
6. Validate mixed-version behavior, then remove compatibility paths only after observed legacy traffic reaches the approved threshold/window.

### Observability Without Token Content

Potential fields requiring approval: transport (`http`, `socketio`, `grpc`, `mcp`), caller service, receiver service, method/tool, selected mode, credential-present boolean, claim-presence bitset, duplicate count, empty-first boolean, conflict boolean, validation result/reason category, and policy version. Do not log token, prefix/suffix, length if length can fingerprint issuers, JWT claims payload wholesale, `_meta`, headers, or hashes usable for cross-system tracking unless security explicitly approves a keyed short-lived correlation scheme.

### Rollback Inputs

- Keep receiver dual-mode acceptance while sender cohorts change.
- Version policy/config independently from code and record the selected policy version in content-free telemetry.
- Roll sender cohort back before disabling receiver compatibility.
- Preserve current wire keys and `b64:` contract during this parent task.
- A rollback must not restore the three known token-value logs; log redaction should be independently deployable.

### Inputs for Next Three Child Tasks

| Child task | Required inputs from audit | Must not be combined |
|---|---|---|
| `typed-auth-context-keys` | Complete direct string-key writer/getter list: HTTP (`aigtw.go:72-97`, gateway auth-type middleware), Socket.IO event writes (`server.go:533-754`), authctx claims, grpcx extraction, mcpx extraction/raw-meta. Preserve five wire strings and ordering. Define typed setter/getters and migration fallback/removal criteria. | Do not alter claim accepted types, metadata duplicate behavior, raw-token propagation, `b64:`, or method policy. |
| `normalize-auth-claims` | Current permissive `ClaimString` behavior, mapping direction, JWT sources, MCP verifier, Socket string-only claim copy, required identity consumers (especially AI data isolation). Decide type whitelist, numeric precision, missing/invalid required vs optional behavior. | Do not switch typed-key rollout simultaneously; do not shrink Authorization or enforce transport conflicts. |
| `enforce-authorization-policy` | Approved per-method/tool mode table, receiver trust/auth mechanism, duplicate/conflict rules, raw-meta policy, leakage fixes, metrics/config/gray rollout, and rollback window. Receiver-first compatibility is a PRD requirement (`enforce-authorization-policy/prd.md:9-14`). | Do not upgrade `b64:`; do not bundle claim normalization or typed-key semantic changes; do not globally remove Authorization without per-boundary evidence. |

### Ordering Constraint

The parent PRD mandates audit -> typed keys -> claim normalization -> policy enforcement with separate pre/post user approvals (`.trellis/tasks/08-14-auth-context-hardening/prd.md:9-20`). The active audit PRD also requires waiting for review before proceeding (`audit-authorization-propagation/prd.md:18-29`).

## External References

No external web reference was required to establish current behavior. Policy selection depends on deployment trust architecture and product delegation requirements not represented by generic standards alone.

## Caveats / Not Found

- This report does not select any candidate mode.
- No deployment mechanism for policy flags/gray rollout was identified; configuration ownership must be established in the enforcement design.
