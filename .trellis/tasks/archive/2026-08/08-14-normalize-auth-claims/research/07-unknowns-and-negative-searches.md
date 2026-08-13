# Research: Known Unknowns and Negative Searches

- **Query**: What is unknown/uncertain for claim normalization planning; searches that found nothing
- **Scope**: internal + external (lancet/go-zero/golang-jwt versions)
- **Date**: 2026-08-14

## Known unknowns (need user/owner input before finalizing matrix)

| # | Unknown | Evidence | Needed from |
|---|---|---|---|
| U1 | **Actual production token claim key names**: `user-id` (dash, repo issuers) vs `user_id` (underscore, config ClaimMapping). Audit R1. If production tokens carry `user_id`, `BridgeJWTClaims` dash-copy is a no-op and only mapping works; if `user-id`, mapping is a no-op. | `zerorpc/internal/logic/generatetokenlogic.go:49` (dash), `socketapp/socketpush/internal/logic/gentokenlogic.go:61` (dash); `aigtw.yaml:15-18`, `mcpserver.yaml:22-25` (mapping dash←underscore); `gtw/gtw.go:67` (nil mapping) | business/deployment owner (audit U7) |
| U2 | **Actual production user-id value ranges/formats**: zerorpc DB auto-increment int64 (small); socketpush uid string format unknown. If any issuer emits ids > 2^53, the `tool.ParseToken` float64 path silently rounds. | `model/usermodel_gen.go:54-60` int64 id; `socketpush.pb.go:130` string uid | deployment owner (audit U8) |
| U3 | **MCP SDK `_meta` decode type**: does the MCP Go SDK decode `_meta` numbers to `float64` or `json.Number`? Repo tests seed float64 (`context_meta_test.go:34`), but the SDK's own JSON decode is external. | `common/mcpx/wrapper.go:231-251`, `context_meta.go:27-37` | SDK inspection / external dependency |
| U4 | **Whether `auth-type`/`dept-code` are security attributes** (audit U6) — affects whether strict whitelisting matters. | `identity-claims-matrix.md:15-16` | product/security owner |
| U5 | **Whether any external (non-repo) JWT issuers exist** that feed the gateways/MCP with other claim types (bool user-id etc.). | no repo evidence | deployment owner |
| U6 | **Actual `SocketMetaData` claim keys**: config `socketgtw.yaml:29` lists `uid, deviceId, userId, user_id, dept_id, dept_code, user_name` — none are the standard `user-id`/`user-name` dash keys. Socket path copies only strings anyway; keys don't align with authctx wire names. | `socketgtw.yaml:29`; `socketiox/server.go:517-527` | deployment owner (audit U8) |

## Negative searches (no findings)

- **user-id values ≥ 2^53 anywhere in repo/tests**: none (grep for `9007199254740991`/`2^53`/`9223372036854775807` literals found only `.opencode/node_modules` noise).
- **`ExtractFromClaims` production callers**: none (only `claims_test.go`).
- **`auth-type` business consumer**: none (only writers + generic transport).
- **`dept-code` authorization decision from authctx**: none (Trigger uses explicit request fields).
- **`user-name` authorization decision**: none (echo display + gormx audit only).
- **`UseJSONNumber`/`json.Number` usage in repo code**: none; only go-zero dependency internally.
- **Direct uint64/uint claims**: none produced by repo issuers.
- **`ApplyClaimMappingToCtx`** (referenced in older audit notes): no longer present in current `claims.go` — superseded by typed-key task (`BridgeJWTClaims`).

## Version pins (for conversion-behavior claims)

| Dep | Version | Evidence |
|---|---|---|
| lancet | v2.3.9 | `go.mod:32`; `convertor.go:107-156` |
| golang-jwt/jwt/v4 | v4.5.2 | `go.mod:38`; `parser.go:31,151-157` |
| go-zero | v1.10.3 | `go.mod:72`; `rest/token/tokenparser.go:122-124`, `rest/handler/authhandler.go:72-78` |

All conversion-behavior claims in `01`/`04` were verified by direct execution against lancet v2.3.9 (`json.Number`, `FormatFloat` cases) in addition to source reading.
