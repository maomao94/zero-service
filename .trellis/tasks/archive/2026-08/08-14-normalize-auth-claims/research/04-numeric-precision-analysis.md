# Research: Numeric Precision Analysis

- **Query**: Precision behavior of user-id numbers through JWT JSON parse; whether repo ranges exceed 2^53; exact recommended conversion semantics for integer-valued / fractional / oversized numbers
- **Scope**: internal + verification runs
- **Date**: 2026-08-14

## 1. Where numbers lose precision

### float64 path (repo `tool.ParseToken` — MCP verifier `mcpx/auth.go:36`, socket validators)

- `golang-jwt/jwt/v4@v4.5.2/parser.go:31` `p := &Parser{}` → `UseJSONNumber=false`; `parser.go:151-157` decodes numbers into `float64` via `json.Decoder` without `UseNumber`.
- **Any integer > 2^53 (= 9007199254740992) is rounded at decode time.** The rounding is permanent; `convertor.ToString` cannot recover the original.

### json.Number path (go-zero gateway middleware)

- `go-zero@v1.10.3/rest/token/tokenparser.go:122-124` uses `jwt.WithJSONNumber()` → `dec.UseNumber()` → numbers arrive as `json.Number` (raw literal).
- `convertor.ToString(json.Number("9007199254740993"))` → `"9007199254740993"` — **exact**, verified by run. No precision loss on the gateway path even for >2^53 values.

### FormatFloat evidence (lancet v2.3.9 convertor.ToString default for float64)

Verified outputs (`strconv.FormatFloat(f,'f',-1,64)`):

| Input float64 | Output string | Note |
|---|---|---|
| `42` | `"42"` | integer-valued, clean |
| `1.5` | `"1.5"` | fractional kept |
| `9007199254740991` (2^53−1) | `"9007199254740991"` | exact |
| `9007199254740992` (2^53) | `"9007199254740992"` | exact |
| `9007199254740993` (2^53+1 as float64) | `"9007199254740992"` | **already rounded before formatting** |
| `1152921504606846976` (2^60) | `"1152921504606847000"` | **silently rounded — trailing digits invented/truncated** |
| `1000000000000000000` (1e18) | `"1000000000000000000"` | exact (1e18 < 2^53? no — 1e18 > 2^53; here 1e18 is exactly representable? no; FormatFloat prints shortest round-trip of the float64, which happens to equal the literal) |
| `123456789012345678` | `"123456789012345680"` | **rounded to nearest float64** |
| `1000000000000000000000` (1e21) | `"1000000000000000000000"` | prints as full decimal, no exponent, but value may be rounded at parse |

Key fact: `FormatFloat(...,'f',-1,...)` never emits an exponent; it prints the shortest decimal that round-trips to the same float64. So for a float64 that *was already rounded* at JSON parse, the string is the rounded value.

`%g` contrast: `FormatFloat(1e21,'g',-1,64)` → `"1e+21"` — would change string format for big values; **current code uses `'f'`, so no `1e+21`-style strings appear**.

## 2. Are repo user-id ranges > 2^53?

**No evidence found.**

- zerorpc issuer: `userId int64` from DB `user` table auto-increment `Id int64` (`model/usermodel_gen.go:54-60`, `loginlogic.go:97-106` `LastInsertId()`). Realistic DB auto-increment values are far below 2^53.
- socketpush issuer: `uid string` (`gentokenlogic.go:61`; `GenTokenReq.Uid` string `socketpush.pb.go:130`); string uid could theoretically be any format — **unknown production format** (U8).
- `zerorpc.pb.go` `GetUserInfoReq.Id` is string; `GenerateTokenReq.UserId` is int64.
- Negative search: no test fixture, config, or code path in the repo hard-codes a user-id ≥ 2^53 (`grep` for 2^53 / MaxInt64 literals found nothing relevant; only `.opencode/node_modules` noise).

Conclusion: precision loss > 2^53 is **not currently reachable** in the repo's own issuers on the gateway path, but **is reachable on the MCP/socket float64 path if a token issuer ever emits ids > 2^53** or if an external MCP `_meta` supplies a huge numeric `user-id`. The go-zero path is already exact; the repo `tool.ParseToken` path is the lossy one.

## 3. Conversion semantics used today vs recommended

| Value shape | Current (lancet ToString) | Precision-safe? | Notes |
|---|---|---|---|
| `json.Number` int | exact literal string | yes | gateway path |
| `float64` integer ≤ 2^53 | exact | yes | MCP/socket path |
| `float64` integer > 2^53 | rounded string | **no** | already lost at parse |
| `float64` fractional (e.g. 1.5) | `"1.5"` | n/a | accepted today |
| `int64`/`uint64` direct (only in unit tests) | exact base-10 | yes | not produced by issuers |

## 4. Downstream numeric re-parsing

- `gtw/getcurrentuserlogic.go:37` passes user-id string into `GetUserInfoReq.Id` (string) → `zerorpc/getuserinfologic.go:33` `convertor.ToInt(in.GetId())`.
- `convertor.ToInt` (`lancet convertor.go:171-200`):
  - string → `strconv.ParseInt(v.String(), 0, 64)` — **base 0**: a leading `0` (e.g. `"0123"`) is parsed as octal → **silent wrong user id**; `"0x.."` as hex. Non-numeric → error (request fails).
  - float64 → `int64(v.Float())` **truncates** fraction silently (`"1.5"` → 1).
  - So a fractional/octal-prefixed user-id string can map to a *different* valid user id without error — audit mismatch, addressed by claim normalization whitelist (reject fractional; require decimal integer form).

## 5. Recommended exact conversion rules (analysis basis for the options file)

For integer-shaped numeric claims, the normalization should, at minimum, distinguish:

1. **String** — accept verbatim (must remain the dominant whitelisted type for `user-id` given socketpush emits strings).
2. **json.Number** — validate against `^[0-9]+$` (integer), optionally length/range check; emit literal.
3. **int/int64/uint/uint64** — emit exact base-10 (never produced by repo issuers, but unit tests use them).
4. **float64/float32 integer-valued** (no fractional part) — emit via FormatInt after range check; reject if `math.Trunc(v) != v` (fractional) or `|v| > MaxInt64` (oversized, already-lossy).
5. **float64/float32 fractional** — **reject** (produce `1.5`-style user-ids today).
6. **bool/array/map/struct/other** — **reject** (produce `"true"`/`["a"]`/`{"k":"v"}` today).
7. **nil / missing** — treated as absent (skip; today yields `""` which is then skipped by empty-check in callers).

Oversized policy choice: because float64 > 2^53 is already corrupted, recommend **reject** any `float64` numeric claim whose integer value exceeds 2^53 (or, more strictly, whose magnitude exceeds `math.MaxInt64`), since the exact id cannot be recovered on that path. json.Number values can be range-checked exactly.

Range check target for user-id: no repo constraint; aisolo stores `varchar(64)` (`session/gormx_store.go:223`), knowledge uses varchar — a length cap (e.g. 64 chars) prevents oversized-string abuse on the MCP `_meta` path.
