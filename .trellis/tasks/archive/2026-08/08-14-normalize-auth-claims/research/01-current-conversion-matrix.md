# Research: Current Claim Conversion Matrix

- **Query**: Exact current semantics of claim → string conversion for every input type, and which code paths perform it
- **Scope**: internal
- **Date**: 2026-08-14

## Conversion engine

All claim conversion funnels through lancet `convertor.ToString` v2.3.9
(`common/authctx/claims.go:35-41` → `ClaimString`; `common/authctx/context.go:162-165` → `toStringClaim`).

Source: `/Users/hehanpeng/go/pkg/mod/github.com/duke-git/lancet/v2@v2.3.9/convertor/convertor.go:107-156`

| Input type | Current output (`convertor.ToString`) | Format |
|---|---|---|
| `nil` | `""` | early return |
| nil `*T` pointer | `""` | deref → nil → `""` |
| `string` | as-is | `case string` |
| `[]byte` | string cast | `case []byte` |
| `int`, `int8/16/32/64` | base-10 integer string | `strconv.FormatInt` |
| `uint`, `uint8/16/32/64` | base-10 integer string | `strconv.FormatUint` |
| `float32` | `strconv.FormatFloat(f,'f',-1,32)` | decimal notation, no exponent |
| `float64` | `strconv.FormatFloat(f,'f',-1,64)` | decimal notation, no exponent |
| `bool` | `"true"` / `"false"` | falls to default → `json.Marshal` |
| slice/array | JSON array string, e.g. `["a"]` | default → `json.Marshal` |
| map | JSON object string, e.g. `{"k":"v"}` | default → `json.Marshal` |
| struct | JSON object string | default → `json.Marshal` |
| `json.Number` | raw JSON number literal, e.g. `json.Number("9007199254740993")` → `"9007199254740993"` | default → `json.Marshal(json.Number)` — **exact literal preserved** |
| missing key | `""` | `ClaimString` early return on `!ok` (`claims.go:37-39`) |

Key behavioral notes:

- **Integer-valued float64**: `FormatFloat(42,'f',-1,64)` → `"42"` (no `.0`). Verified.
- **Fractional float64**: `FormatFloat(1.5,'f',-1,64)` → `"1.5"`. Verified.
- **Large float64**: `FormatFloat(9007199254740993,'f',-1,64)` (a float64 that already rounded) → `"9007199254740992"` — the **value was already rounded at JSON parse time**; `'f'` formatting does not invent digits. Verified: `2^53+1` float64 → `"9007199254740992"`.
- **json.Number is NOT matched by `case string`** — it is a distinct `type Number string`. It falls through to `default` → `json.Marshal`, and `encoding/json` marshals a `json.Number` by emitting its raw literal. So `json.Number` round-trips **exactly**, including values > 2^53. Verified by run.

## Boundaries where conversion happens

| # | Path | Function | Line | Value type at that point |
|---|---|---|---|---|
| B1 | JWT claims → typed ctx (used by MCP verifier, unused elsewhere) | `authctx.ExtractFromClaims` | `claims.go:10-20` | claims from `tool.ParseToken` → **float64** numbers |
| B2 | external claim → internal key (claims map copy) | `authctx.ApplyClaimMapping` | `claims.go:25-31` | copies raw value, **no type change** |
| B3 | go-zero JWT raw string-key ctx → typed keys (gateways) | `authctx.BridgeJWTClaims` | `context.go:148-160` | claims from go-zero middleware → **json.Number** numbers |
| B4 | MCP `_meta` map → typed ctx | `mcpx.ExtractFromMeta` | `context_meta.go:27-37` | `_meta` JSON-decoded → **float64** numbers (or json.Number if SDK uses UseNumber — see unknowns) |
| B5 | MCP verifier: JWT claims → `TokenInfo.UserID` | `mcpx.NewDualTokenVerifier` | `auth.go:58` | claims from `tool.ParseToken` → **float64** numbers |
| B6 | MCP verifier: claims → `Extra` map (raw values) | `mcpx.NewDualTokenVerifier` | `auth.go:46-52` | raw value copied, **no conversion** |
| B7 | Socket claims → session metadata | `socketiox/server.go:517-527` | string-only gate: `v.(string)` else skipped, then `convertor.ToString(v)` | claims from `tool.ParseToken` → **float64** numbers, but only strings pass |
| B8 | typed ctx → gRPC metadata | `grpcx.InjectToGrpcMD` | `metadata.go:46-59` | already string (getters) |
| B9 | gRPC metadata → typed ctx | `grpcx.ExtractFromGrpcMD` | `metadata.go:63-74` | wire strings, b64 decoded |

## Current contract tests (lock permissive behavior)

| Test | Input | Locked output |
|---|---|---|
| `claims_test.go:21` | `float64(42)` mapped user-id | `"42"` |
| `claims_test.go:24` | `CtxDeptCodeKey: 1.5` | `"1.5"` |
| `claims_test.go:27` | `CtxAuthTypeKey: true` | `"true"` (permissive bool) |
| `claims_test.go:38-40` | missing key | `""` |
| `claims_test.go:41-42` | `nil` value | `""` (note: audit report earlier claimed `"<nil>"`; current lancet impl returns `""` — test asserts `""`) |
| `context_test.go:120-122` | `int64(42)`, `int(42)`, `float64(42)` user-id | `"42"` |
| `context_test.go:134-144` | `bool true` user-id; `[]string{"a"}` user-name | `"true"`; `["a"]` (both converted) |
| `context_test.go:160-176` | existing typed value + raw string key | typed value wins, no overwrite |
| `mcpx/context_meta_test.go:39-41` | `meta[dept-code]=float64(12)` | `"12"` |
| `mcpx/auth_test.go:17,37` | `user_id: float64(42)` | `Extra[user-id] == float64(42)` (raw), `UserID == "42"` |

Implication: replacing permissive conversion with a whitelist **requires updating these tests**; they currently assert the permissive outputs.

## Caveats / Not Found

- `ExtractFromClaims` has no production callers (only `claims_test.go:20`). It is the intended JWT-claims entry point but currently unused outside tests.
- `ApplyClaimMapping` production caller: only `mcpx/auth.go:43`.
- `BridgeJWTClaims` production callers: `aigtw.go:93` (with mapping) and `gtw.go:67` (with `nil` mapping). Socket gateway does not call it.
