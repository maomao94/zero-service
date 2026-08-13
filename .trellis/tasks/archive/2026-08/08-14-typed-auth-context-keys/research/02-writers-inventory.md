# Research: Direct writers of auth context string keys

- **Query**: Every call site doing `context.WithValue(ctx, authctx.XKey, value)` or equivalent across the repo.
- **Scope**: internal (repo-wide)
- **Date**: 2026-08-14

## Findings

### 1. HTTP gateways (global middleware)

| File:Line | Key | Value | Gated by verification? |
|---|---|---|---|
| `aiapp/aigtw/aigtw.go:83` | `CtxAuthTypeKey` (`auth-type`) | `"user"` | No — runs for every request; only gated by non-empty `Authorization` header |
| `aiapp/aigtw/aigtw.go:84` | `CtxAuthorizationKey` (`authorization`) | raw `Authorization` header value | No (writes raw header; audit R2: downstream gRPC only fires after `rest.WithJwt` on protected routes, so unverified token is written to ctx but not forwarded) |
| `aigtw/aigtw.go:90-98` | via `authctx.ApplyClaimMappingToCtx` | config `ClaimMapping` (internal→external, e.g. `user-id`←`user_id`) | No — runs for every request when `JwtAuth.ClaimMapping` configured |
| `gtw/gtw.go:60` | `CtxAuthTypeKey` (`auth-type`) | `"user"` | No — unconditional global middleware |
| `socketapp/socketgtw/socketgtw.go:68` | `CtxAuthTypeKey` (`auth-type`) | `"user"` | No — unconditional global middleware (HTTP server half) |

Config mapping values (internal key → external JWT claim key):
- `aiapp/aigtw/etc/aigtw.yaml:15-18`: `user-id→user_id`, `user-name→user_name`, `dept-code→dept_code`
- `aiapp/mcpserver/etc/mcpserver.yaml:22-25`: same trio

### 2. go-zero JWT middleware (OUT-OF-REPO writer — critical)

`github.com/zeromicro/go-zero@v1.10.3/rest/handler/authhandler.go:72-78` (dependency, not repo code):

```go
ctx := r.Context()
for k, v := range claims {
    switch k {
    case jwtAudience, jwtExpire, jwtId, jwtIssueAt, jwtIssuer, jwtNotBefore, jwtSubject:
        // ignore the standard claims
    default:
        ctx = context.WithValue(ctx, k, v)
    }
}
```

- **Every non-standard JWT claim is written into process context using the raw claim name as a string key, with the raw claim value type** (string, int64, float64, …).
- Used via `rest.WithJwt(secret)` at:
  - `gtw/internal/handler/routes.go:151` (route group `/app/user/v1`)
  - `aiapp/aigtw/internal/handler/routes.go:37,50,83,198,211,226`
- This is the **actual producer** of `user-id` / `user-name` / `dept-code` (or `user_id` / `user_name` / `dept_code`, depending on token claim names) in gateway process context. Token issuers:
  - `zerorpc/internal/logic/generatetokenlogic.go:49` — `claims[authctx.CtxUserIdKey] = userId` where `userId int64` (**int64 claim value**; audit R1: `authctx.GetUserId` string-assert → `""`).
  - `socketapp/socketpush/internal/logic/gentokenlogic.go:61` — `claims[authctx.CtxUserIdKey] = uid` (**string claim value**).
- Implication: a typed-key-only migration **cannot intercept or replace this writer** — it is dependency code writing raw string keys. The gateway process context will keep receiving string-keyed claims unless a bridge is added.

### 3. Socket.IO event context writes (`common/socketiox/server.go`)

All write `CtxAuthorizationKey` (`authorization`) with the connection handshake token. Gated by `OnAuthentication` (server.go:496-507) which runs `tokenValidator` at connection time.

| File:Line | Key | Value | Context |
|---|---|---|---|
| `server.go:537` | `CtxAuthorizationKey` | handshake `token` | connection ctx |
| `server.go:558` | `CtxAuthorizationKey` | `token` | EventJoinRoom |
| `server.go:579` | `CtxAuthorizationKey` | `token` | EventLeaveRoom |
| `server.go:594` | `CtxAuthorizationKey` | `token` | EventRoomsPage |
| `server.go:610` | `CtxAuthorizationKey` | `token` | EventUp |
| `server.go:673` | `CtxAuthorizationKey` | `token` | EventRoomBroadcast |
| `server.go:698` | `CtxAuthorizationKey` | `token` | EventGlobalBroadcast |
| `server.go:730` | `CtxAuthorizationKey` | `token` | disconnect |
| `server.go:754` | `CtxAuthorizationKey` | `token` | generic event handlers |

Related (not context, but same identity source): `server.go:517-529` copies configured `contextKeys` (from `SocketMetaData` config, e.g. `socketgtw.yaml:29` `["uid","deviceId","userId","user_id","dept_id","dept_code","user_name"]`) into **Session metadata** (not process context) from validated claims — string-only copy.

### 4. authctx internal helpers (writers by construction)

| File:Line | Key | Value |
|---|---|---|
| `common/authctx/claims.go:15` (`ExtractFromClaims`) | each `ContextKeys` entry | `ClaimString`-converted non-empty string |
| `common/authctx/claims.go:34` (`ApplyClaimMappingToCtx`) | each mapping internal key | copied external ctx value (any type preserved) |

### 5. Transport extraction writers (restore to context keys)

| File:Line | Function | Key | Value |
|---|---|---|---|
| `common/grpcx/metadata.go:71` | `ExtractFromGrpcMD` | `f.contextKey` (each of 5 authctx keys) | first non-empty incoming metadata value, b64-decoded |
| `common/mcpx/context_meta.go:33` | `ExtractFromMeta` | each `ContextKeys` entry | `ClaimString(meta, key)` non-empty string |
| `common/mcpx/wrapper.go:250` | `CallToolWrapper` (opt-in `WithExtractUserCtx`) | via `ExtractFromMeta` | same |
| `common/mcpx/wrapper.go:244` | `CallToolWrapper` | `ctxMetaKey` (`"_meta"`) | raw MCP `_meta` map (not auth ctx key, but same package) |

### 6. JWT claim producers (token side, string-key naming)

| File:Line | Claim key | Value type |
|---|---|---|
| `zerorpc/internal/logic/generatetokenlogic.go:49` | `CtxUserIdKey` = `"user-id"` | `int64` |
| `socketapp/socketpush/internal/logic/gentokenlogic.go:61` | `CtxUserIdKey` = `"user-id"` | `string` |

### 7. Test writers (lock the string-key contract)

- `common/grpcx/client_interceptor_test.go:20,53` — `CtxUserIdKey`, `CtxDeptCodeKey`
- `common/grpcx/metadata_test.go:61,88-90` — `CtxAuthorizationKey`, literal `string("user-id"/"user-name"/"dept-code")`
- `common/mcpx/context_meta_test.go:20-23,34` — `CtxAuthorizationKey`, `CtxUserIdKey`, `CtxUserNameKey`(`""`), `CtxDeptCodeKey`(`float64(3)`)
- `aiapp/aigtw/internal/logic/solo/session_validation_test.go:17,50,84,93,99,124,149,165`, `chat_resume_validation_test.go:61,67,81,115`, `knowledge_validation_test.go:14,125` — `CtxUserIdKey` = `"user-1"` to seed logic tests.

### Summary counts

- 3 HTTP gateway middleware writers
- 1 out-of-repo go-zero JWT claims writer (raw string keys)
- 9 socketiox event-context writes
- 2 authctx helper writers
- 2 transport extract writers + 1 MCP wrapper restore
- 2 token issuers (string-key claim naming)