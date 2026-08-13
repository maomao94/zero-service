# Research: Identity Claims Matrix

- **Query**: Trace `user-id`, `user-name`, `dept-code`, and `auth-type` sources, mapping, propagation, consumers, and authorization/data-isolation use
- **Scope**: internal
- **Date**: 2026-08-14

## Findings

### Claims Matrix

| Claim | Sources and mapping | Propagation | Consumers | Authorization / isolation use |
|---|---|---|---|---|
| `user-id` | go-zero JWT middleware writes JWT claims to string-keyed HTTP context; AI gateway maps external `user_id` to internal key (`aiapp/aigtw/etc/aigtw.yaml:14-18`, `aigtw.go:90-97`). Socket JWT claims are copied only when configured in `SocketMetaData` and only when already strings (`common/socketiox/server.go:517-527`). MCP verifier applies internal->external mapping and permissive conversion (`common/mcpx/auth.go:42-59`). Token generators set this claim (`zerorpc/internal/logic/generatetokenlogic.go:45-52`, `socketapp/socketpush/internal/logic/gentokenlogic.go:57-76`). | gRPC `x-user-id`; MCP `_meta["user-id"]`; process context (`common/grpcx/metadata.go:27-74`, `common/mcpx/context_meta.go:12-36`) | Generic getters (`common/authctx/context.go:22-27`, `common/gormx/user_context.go:39-85`, `common/tool/userutil.go:9-33`); gateway current-user lookup (`gtw/internal/logic/user/getcurrentuserlogic.go:34`); many AI gateway session/knowledge operations | Confirmed data isolation: AI gateway requires current user ID and places it in AISolo request fields (`aiapp/aigtw/internal/logic/solo/createsessionlogic.go:31-37`, `listsessionslogic.go:27-34`, `chatlogic.go:35-42`). AISolo checks request user ID against stored records, e.g. `aiapp/aisolo/internal/logic/getinterruptlogic.go:48`. Therefore propagated/derived user ID affects record ownership checks. |
| `user-name` | JWT/MCP claims, mapping from `user_name`; Socket metadata when configured/string; fallback from `CurrentUser.UserName` (`common/tool/userutil.go:36-60`) | gRPC `x-user-name`; MCP `_meta["user-name"]`; process context | MCP echo reads/displays it (`aiapp/mcpserver/internal/tools/echo.go:25-40`); GORM audit helper exposes name (`common/gormx/user_context.go:85`) | No repository evidence of an authorization decision based on user name. It is identity/display/audit data. |
| `dept-code` | JWT/MCP claims, mapping from `dept_code`; Socket metadata when configured/string; fallback first department on user object (`common/tool/userutil.go:63-96`) | gRPC `x-dept-code`; MCP `_meta["dept-code"]`; process context | Generic authctx/gormx/tool helpers; Trigger proto contains explicit department fields and validations, but a direct authctx-to-Trigger authorization decision was not found in this search | Potential tenant/data scope field, but no confirmed context-based authorization decision found. Explicit request `dept_code` fields are separate and must not be assumed equivalent to verified context metadata. |
| `auth-type` | Gateway global middleware marks browser requests `user` (`gtw/gtw.go:57-63`, `socketapp/socketgtw/socketgtw.go:65-71`); AI gateway marks only when Authorization exists (`aiapp/aigtw/aigtw.go:82-85`); MCP verifier sets `service` or `user` (`common/mcpx/auth.go:22-47`) | gRPC `x-auth-type`; MCP `_meta["auth-type"]`; process context | No getter or business consumer found; it is carried generically through `ContextKeys` | No authorization/data-isolation use found. A caller can supply it through gRPC/MCP metadata unless a verified boundary overwrites it. |

### Type and Conflict Behavior

- `authctx.ExtractFromClaims` iterates all five keys, uses `ClaimString`, and writes only non-empty converted values (`common/authctx/claims.go:8-18`).
- `ClaimString` accepts strings and float64 specially, then formats every other type with `%v`, including booleans, arrays, maps, and nil-like values (`common/authctx/claims.go:40-56`; contract examples in `claims_test.go:9-28`).
- `ApplyClaimMapping` copies external values onto internal keys when present and leaves an existing internal value untouched only when the external key is absent (`common/authctx/claims.go:21-27,40-56`).
- `ApplyClaimMappingToCtx` copies any non-nil external context value to the internal string key without normalization (`common/authctx/claims.go:30-37`). A non-string mapped value will later be skipped by strict getters and gRPC/MCP collectors.
- gRPC receiver values override any prior process context only when the first metadata value is non-empty; MCP extraction similarly overwrites with each non-empty converted value.

### Trust Classification

| Source | Evidence level |
|---|---|
| JWT claims after go-zero middleware or `tool.ParseToken` | Cryptographically verified at that boundary, subject to configured secret and claim semantics |
| MCP service-token request `_meta` identity | Caller-provided metadata from a client authenticated as a service; user claims are not cryptographically verified by wrapper |
| gRPC `x-user-*` / `x-auth-type` | Caller-provided metadata; generic receiver does not verify or reconcile it with Authorization |
| Process-derived values / explicit proto fields | Trust depends on constructing handler; not inherently linked to token |

### Related Specs

- `.trellis/tasks/08-14-normalize-auth-claims/prd.md:3-19` owns type whitelist and invalid-value behavior.
- `.trellis/tasks/08-14-typed-auth-context-keys/prd.md:3-20` owns separation of process keys from wire keys.

## Caveats / Not Found

- No business consumer of `auth-type` was found.
- No confirmed `dept-code` authorization decision sourced from authctx was found.
- Generated protobuf `GetUserId`/`GetDeptCode` symbols were excluded from context-consumer conclusions unless a logic file used them for ownership checks.
