# Research: Type Whitelist + Reject-vs-Ignore Semantics (OPTIONS for user confirmation)

- **Query**: Per claim, mandatory-vs-optional classification and recommended reject-vs-ignore semantics for illegal types — options only, NOT a final decision
- **Scope**: internal analysis; decision reserved for the user (PRD requirement `prd.md:10`)
- **Date**: 2026-08-14

> The audit PRD gate requires the user to confirm the type matrix and reject-vs-ignore semantics **before dev** (`prd.md:10`). This file provides the evidence and option space; it does not decide.

## 1. Mandatory vs optional classification

| Claim | Security attribute (audit §6) | Issued by repo issuers? | Mandatory / optional |
|---|---|---|---|
| `user-id` | **security / data isolation** | yes (zerorpc int64, socketpush string) | **Mandatory** for JWT-gated routes (aigtw/gtw reject empty); optional for service-token MCP calls (auth-type=service has no user-id) |
| `user-name` | informational | not issued by repo issuers | Optional |
| `dept-code` | potential scope field, unproven | not issued by repo issuers | Optional |
| `auth-type` | informational, forgeable | written by gateways/verifier as string | Optional (writers always set it) |
| `authorization` | credential | raw token | Out of scope for type normalization (always string) |

## 2. Options: whitelist per claim

### Option A — strict numeric-capable whitelist (recommended for `user-id`)

Allowed input types for `user-id`:
- `string` (decimal digits only? or verbatim? — see Option A1/A2)
- `json.Number` (integer only, range-checked)
- `int`/`int64`/`uint`/`uint64` (exact)
- `float32`/`float64` **only if integer-valued and ≤ 2^53** (reject fractional/oversized)
- Reject: `bool`, arrays, maps, structs, `nil` (treated as missing)

### Option B — string-only whitelist (strictest for `user-id`)

Allow `string` only; reject all numeric types. This would **break** zerorpc int64 user-ids on the gateway path? — No: go-zero decodes int64 JSON numbers to `json.Number`, not string. So string-only would reject legitimate int64 ids. **Not viable without issuer change.** (Evidence: gateway path values are json.Number for numbers; socketpush values are strings.)

### Option C — permissive-strings whitelist (current behavior, Optionally tightened)

Allow string + all numeric types as today, but reject bool/array/map/struct. Keeps `"42"`, `"1.5"`, `"9007199254740992"` possible.

### Option D — whitelist identical to A for all five claims

Apply the same whitelist to `user-name`, `dept-code`, `auth-type`. Since none are security-sensitive today, strictness is optional; a string-only whitelist may be simpler for these.

## 3. Reject vs ignore semantics (per boundary)

For each illegal value type (bool/array/map/struct/fractional/oversized), two end-to-end semantics:

| Semantic | Meaning at extraction/bridge/verifier | Consequence |
|---|---|---|
| **REJECT** | Return error/unauthorized for the whole request/verification; do not establish identity | Hard failure; explicit; token with malformed claim is unusable |
| **IGNORE** | Drop the claim; treat as missing; continue with remaining claims | Soft failure; identity may be empty → downstream aigtw/gtw return unauthenticated anyway (same observable outcome for user-id) |

Boundary-by-boundary options:

| Boundary | Function | REJECT possible? | IGNORE possible? | Notes |
|---|---|---|---|---|
| JWT→ctx (gateway) | `BridgeJWTClaims` | no error return today — would need new error path; middleware has no failure channel (bridge currently best-effort) | yes — skip invalid claim, keep valid ones | go-zero middleware already ran; bridge has no way to 401 |
| JWT→TokenInfo (MCP verifier) | `NewDualTokenVerifier` | **yes** — verifier can return `auth.ErrInvalidToken` | yes — omit claim from Extra/UserID | Natural rejection point for JWT-based MCP auth |
| `_meta`→ctx (MCP server wrapper) | `ExtractFromMeta` | possible — but `_meta` is service-authenticated caller metadata; rejection would kill tool calls on metadata junk | yes — skip invalid `_meta` values | Needs wrapper-level decision |
| claims map copy | `ApplyClaimMapping` | no error return (mutates map) | yes — skip copy of invalid source | Currently copies raw value unchanged |
| gRPC metadata→ctx | `ExtractFromGrpcMD` | possible (server interceptor) | yes — skip malformed wire value | wire is string; type validation limited to format checks |
| Socket claims | `server.go:517-527` | no | yes — already string-only filter | already ignores non-strings |

## 4. Per-claim recommended default (option, for user confirmation)

| Claim | Recommended whitelist | Illegal value default | Rationale |
|---|---|---|---|
| `user-id` | A (string+integer numerics, integer-valued float ≤2^53) | **REJECT at JWT verifier** (MCP); **IGNORE at bridge/_meta** (no failure channel); **IGNORE+skip at mapping** | Security claim; reject where possible, never accept bool/array/map/fractional/oversized as a user key |
| `user-name` | string only (or A) | IGNORE | informational; strictest string-only is safe |
| `dept-code` | string only (or A) | IGNORE | unproven security value; no consumer today |
| `auth-type` | string only | IGNORE | forgeable; no consumer; writers emit string |
| `authorization` | string only | IGNORE (never convert non-strings) | credential; must remain string on wire |

## 5. Compatibility evidence needed before dev (PRD acceptance)

- Existing legal tokens: zerorpc (int64), socketpush (string) — both must continue working under the chosen whitelist.
- The chosen whitelist must be validated against the **actual production issuer claim-key names** (`user-id` vs `user_id`) and types (audit R1, U7) — currently the config mapping is `user-id ← user_id` (`aigtw.yaml:15`, `mcpserver.yaml:22`) while repo issuers write `user-id`. This is an input decision the user must confirm before the type matrix is finalized.
- Tests to update with the new semantics: `claims_test.go:20-29` (bool→"true", 1.5→"1.5"), `context_test.go:134-144` (bool/array converted), `mcpx/context_meta_test.go:39-41` (float64 dept-code), `mcpx/auth_test.go:37` (Extra raw float64).
