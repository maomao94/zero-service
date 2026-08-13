# Research: authctx public API inventory

- **Query**: Every exported constant (context keys), every exported getter/setter, signatures, string-key usage, and tests locking behavior.
- **Scope**: internal (`common/authctx/`)
- **Date**: 2026-08-14

## Findings

### Package location

`common/authctx/` — 4 files:
- `context.go` (48 lines)
- `claims.go` (57 lines)
- `context_test.go` (40 lines)
- `claims_test.go` (52 lines)

### Exported context key constants (all plain `string`)

`common/authctx/context.go:5-11`:

```go
const (
	CtxUserIdKey        = "user-id"
	CtxUserNameKey      = "user-name"
	CtxDeptCodeKey      = "dept-code"
	CtxAuthorizationKey = "authorization"
	CtxAuthTypeKey      = "auth-type"
)
```

| Constant | Value | Kind |
|---|---|---|
| `CtxUserIdKey` | `"user-id"` | process ctx string key + wire claim key |
| `CtxUserNameKey` | `"user-name"` | process ctx string key + wire claim key |
| `CtxDeptCodeKey` | `"dept-code"` | process ctx string key + wire claim key |
| `CtxAuthorizationKey` | `"authorization"` | process ctx string key + wire claim key |
| `CtxAuthTypeKey` | `"auth-type"` | process ctx string key + wire claim key |

### Exported `ContextKeys` slice (propagation order — a wire contract)

`common/authctx/context.go:13-20`:

```go
var ContextKeys = []string{
	CtxAuthorizationKey, // "authorization" first
	CtxUserIdKey,        // "user-id"
	CtxUserNameKey,      // "user-name"
	CtxDeptCodeKey,      // "dept-code"
	CtxAuthTypeKey,      // "auth-type"
}
```

Order is consumed by:
- `common/grpcx/metadata.go:28-34` (metadataFields mirrors this order)
- `common/mcpx/context_meta.go:15,31` (CollectFromCtx / ExtractFromMeta iteration)
- `common/mcpx/auth.go:46-52` (Extra map population)

### Exported getters (all `func(ctx context.Context) string`, string-assert; empty string fallback)

`common/authctx/context.go:22-48`:

| Function | Returns | Signature | Behavior |
|---|---|---|---|
| `GetUserId` | `string` | `func GetUserId(ctx context.Context) string` | `ctx.Value(CtxUserIdKey).(string)` else `""` |
| `GetUserName` | `string` | `func GetUserName(ctx context.Context) string` | `ctx.Value(CtxUserNameKey).(string)` else `""` |
| `GetAuthorization` | `string` | `func GetAuthorization(ctx context.Context) string` | `ctx.Value(CtxAuthorizationKey).(string)` else `""` |
| `GetDeptCode` | `string` | `func GetDeptCode(ctx context.Context) string` | `ctx.Value(CtxDeptCodeKey).(string)` else `""` |

**No `GetAuthType` getter exists.** `auth-type` is written (gateways, mcpx auth) but has zero readers in the repo (audit report §6: "当前无消费者").

### Exported setters / claim helpers (`common/authctx/claims.go`)

| Function | Signature | Behavior |
|---|---|---|
| `ExtractFromClaims` | `func ExtractFromClaims(ctx context.Context, claims map[string]any) context.Context` | Iterates `ContextKeys`; for each, `ClaimString(claims, key)` non-empty → `context.WithValue(ctx, key, v)` (claims.go:9-19) |
| `ApplyClaimMapping` | `func ApplyClaimMapping(claims map[string]any, mapping map[string]string)` | Maps internal→external key inside claims map (claims.go:22-28) |
| `ApplyClaimMappingToCtx` | `func ApplyClaimMappingToCtx(ctx context.Context, mapping map[string]string) context.Context` | Copies externally-named ctx values onto internal keys (claims.go:31-38) |
| `ClaimString` | `func ClaimString(claims map[string]any, key string) string` | Permissive conversion: string→as-is; float64 integer→`%d`; other float64→`%g`; everything else→`%v` (claims.go:41-56) |

Notes:
- `ExtractFromClaims` writes only **non-empty converted strings**; numeric claims are stringified (e.g. `float64(42)` → `"42"`, `1.5` → `"1.5"`, `true` → `"true"`).
- `ApplyClaimMapping` / `ApplyClaimMappingToCtx` never delete existing values; absent external key is a no-op.
- These helpers write **string keys** into context (`context.WithValue(ctx, key, v)` where `key` is the plain string).

### Value types written under these keys

- **Strings** in production: gateways (`"user"` auth-type, raw `Authorization`), socketiox (`token`), extraction (metadata strings, MCP meta strings).
- **Non-string values are possible** in tests and via go-zero JWT middleware:
  - `context_test.go:37` — `GetUserId` on `int` value must return `""` (string assertion).
  - `metadata_test.go:88-90` — `string("user-name")=42` (int) is filtered by grpcx inject.
  - `context_meta_test.go:23` — `CtxDeptCodeKey` holds `float64(3)`; CollectFromCtx filters it (non-string).
  - go-zero `authhandler.go:72-78` — JWT claims written raw (int64/float64/string depending on issuer); see `02-writers-inventory.md`.

### Tests locking the API (`context_test.go`, `claims_test.go`)

`common/authctx/context_test.go:9-39` `TestContextKeyContractAndGetters`:
- Locks `ContextKeys` to exactly `["authorization","user-id","user-name","dept-code","auth-type"]` (line 10-13).
- Locks each key's dynamic type to `string` (line 14-18).
- Locks that a **literal `string("user-id")` write is readable by `GetUserId`** (line 21) — i.e. key equality is by value, not by constant identity.
- Locks non-string `user-id` (`int`) yields `""` (line 37).

`common/authctx/claims_test.go`:
- `TestClaimsContract` (line 9): locks `ApplyClaimMapping` mapping of `user-id`←`external_user`; locks `ExtractFromClaims` numeric coercion (`"42"`, `"1.5"`, bool `"true"`); locks `ApplyClaimMappingToCtx` stores value under `string("user-id")` (line 35) — **value-equality key contract**.
- `TestClaimMappingDoesNotDeleteOrSynthesizeClaims` (line 40): locks no-op on missing key, `ClaimString` of missing = `""`, `ClaimString(nil)` = `"<nil>"`.

### Related specs

- `.trellis/tasks/archive/2026-08/08-14-audit-authorization-propagation/audit-report.md` §6, §8.1 — claim security attributes; typed-key task inputs/must-not-combine.
- `.trellis/tasks/archive/2026-08/08-14-audit-authorization-propagation/research/identity-claims-matrix.md` — claim naming/type matrix.