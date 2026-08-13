# Research: Real Value-Type Inventory per Claim Boundary

- **Query**: What actual Go type each identity claim has at issuer → wire → parse → context
- **Scope**: internal
- **Date**: 2026-08-14

## Summary table

| Claim | Issuer(s) | Wire type | Parse engine | Value type at parse | Value type in process ctx |
|---|---|---|---|---|---|
| `user-id` | zerorpc: **int64** (`generatetokenlogic.go:45-52`); socketpush: **string** (`gentokenlogic.go:57-76`) | JWT claim JSON number or string; gRPC `x-user-id` string; MCP `_meta["user-id"]` | go-zero middleware → **json.Number**; `tool.ParseToken` → **float64** | json.Number / float64 / string | typed string via BridgeJWTClaims / ExtractFromMeta / verifier |
| `user-name` | JWT claims (optional), MCP `_meta` | string | same as above | string (or permissive-converted) | string |
| `dept-code` | JWT claims (optional), MCP `_meta` | string | same as above | string / float64 / bool / etc. | string |
| `auth-type` | Gateways write `"user"` (`gtw/gtw.go:59`, `aigtw/aigtw.go:82`, `socketgtw.go:67`); MCP verifier writes `"service"`/`"user"` (`mcpx/auth.go:30,47`) | gRPC `x-auth-type`; MCP `_meta` | — | string (writers always set string) | string |
| `authorization` | raw token string from HTTP header / socket handshake / verifier | gRPC `authorization`; MCP `_meta` | — | string | string |

## Per-boundary detail

### 1. Issuers

**zerorpc** `zerorpc/internal/logic/generatetokenlogic.go:45-52`:
```go
func (l *GenerateTokenLogic) getJwtToken(secretKey string, iat, seconds, userId int64) (string, error) {
	claims := make(jwt.MapClaims)
	claims["exp"] = iat + seconds
	claims["iat"] = iat
	claims[authctx.CtxUserIdKey] = userId   // int64 → JSON number "user-id":123
```
`GenerateTokenReq.UserId` is proto `int64` (`zerorpc/zerorpc/zerorpc.pb.go:622`). Caller `loginlogic.go:97-106` passes DB auto-increment `LastInsertId()` (int64) and an existing `userId` (int64). `GetUserInfo` consumer converts the string back with `convertor.ToInt` (`getuserinfologic.go:33-36`).

**socketpush** `socketapp/socketpush/internal/logic/gentokenlogic.go:57-76`:
```go
claims[authctx.CtxUserIdKey] = uid   // uid is string (GenTokenReq.Uid string, pb.go:130)
```
Payload map values are `map[string]string`; standard claim keys are skipped (`gentokenlogic.go:68`).

### 2. Wire format

- JWT payload: JSON. zerorpc int64 → JSON number literal (no quotes). socketpush string → JSON string literal.
- gRPC metadata: all values are strings (`grpcx/metadata.go:13-20` maps to `x-user-id` etc.; b64-encoded if non-printable).
- MCP `_meta`: `map[string]any` serialized to JSON object (`common/mcpx/context_meta.go:12-23` CollectFromCtx; `client.go:774-796` SetMeta). Values collected from ctx are already strings.

### 3. Parse engines (the divergence)

**go-zero gateway path** (`rest.WithJwt` on gtw/aigtw routes) — **json.Number**:
- `go-zero@v1.10.3/rest/token/tokenparser.go:122-124`:
```go
func newParser() *jwt.Parser {
	return jwt.NewParser(jwt.WithJSONNumber())
}
```
- `authhandler.go:72-78` writes each non-standard claim into HTTP context under the raw claim-name string key (`context.WithValue(ctx, k, v)`), value type **json.Number** for numbers.
- Verified by run: `json.Number("9007199254740993")` survives exactly; `jwt.WithJSONNumber()` causes `dec.UseNumber()` in `golang-jwt/jwt/v4@v4.5.2/parser.go:153-157`.
- Consequences:
  - `BridgeJWTClaims` → `toStringClaim(json.Number)` → exact decimal string. No precision loss for int64 user ids, even > 2^53.
  - The claims map reached by `ExtractFromClaims`/`ApplyClaimMapping` in the **gateway** context would contain `json.Number` (if those were wired here — they are not; gateway uses BridgeJWTClaims on context values, not the claims map).

**Repo `tool.ParseToken` path** (MCP verifier `mcpx/auth.go:36`, socket validators `socketgtw` servicecontext.go:55-74) — **float64**:
- `common/tool/authutil.go:19-47` uses `jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, ...)` with the **default** parser (`parser.go:31: p := &Parser{}`), `UseJSONNumber=false`.
- Numbers decode to **float64** (`parser.go:151-157`). Confirmed by `mcpx/auth_test.go:17,37` (claims built with `float64(42)` and asserted `float64(42)`).
- Consequences: user-id > 2^53 loses precision **at this boundary**. This is the MCP HTTP verifier path (P6) and the socket gateway path.

### 4. Process context (typed keys)

All typed context getters store strings (`context.go:36-118`). Conversion to string happens at:
- Gateway: `BridgeJWTClaims` (`context.go:148-160`), which reads raw string-key ctx values written by go-zero middleware. Guard: `v != "" && GetByKey(ctx,key)==""` — existing typed value wins; empty result skipped.
- MCP server wrapper restore: `ExtractFromMeta` (`context_meta.go:27-37`) via `ClaimString` when `WithExtractUserCtx` enabled (`wrapper.go:249-251`); raw `_meta` map also stored unchanged via `WithMeta` (`wrapper.go:241-245`).
- MCP verifier: `info.UserID = ClaimString(claims, user-id)` (`auth.go:58`); `Extra` keeps **raw unconverted values** (`auth.go:46-52`), including `Extra[authorization] = token` (string) and any raw claim values.
- Socket: only string claims copied (`server.go:521-527`).

### 5. Receiver side (gRPC)

`grpcx.ExtractFromGrpcMD` (`metadata.go:63-74`) — wire is always string; no numeric path exists at this boundary. `b64:` prefix decoded.

## Caveats / Not Found

- No code path produces `uint64`/`uint` claims from the two issuers. `convertor.ToString` supports them if they appear via other JWT issuers, but repo issuers only emit int64/string.
- `json.Number` type is only produced by go-zero's parser; the repo's `tool.ParseToken` never produces it.
- MCP SDK `_meta` decode type (float64 vs json.Number) depends on SDK internals — repo test seeds float64 (`context_meta_test.go:34`); see `07-unknowns`.
