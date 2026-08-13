# Research: Tests locking the string-key contract

- **Query**: Tests that lock context key names, wire key names, value types, and propagation behavior.
- **Scope**: internal
- **Date**: 2026-08-14

## Findings

### 1. `common/authctx/context_test.go` — key contract

`TestContextKeyContractAndGetters` (line 9-39):
- Locks `ContextKeys == ["authorization","user-id","user-name","dept-code","auth-type"]` (line 10-13).
- Locks each key dynamic type is `string` (line 14-18).
- Locks `string("user-id")` literal write readable by `GetUserId` (line 21) — **key equality by value, not constant identity**.
- Locks non-string `user-id` (int 1) → `GetUserId() == ""` (line 37).

### 2. `common/authctx/claims_test.go` — claim conversion

- `TestClaimsContract` (line 9-38): locks `ApplyClaimMapping` mapping direction; `ExtractFromClaims` coercion (`float64(42)→"42"`, `1.5→"1.5"`, bool→`"true"`); `ApplyClaimMappingToCtx` stores original value under `string("user-id")` (line 35).
- `TestClaimMappingDoesNotDeleteOrSynthesizeClaims` (line 40-51): no-op on missing key; `ClaimString` missing → `""`; `ClaimString(nil)` → `"<nil>"`.

### 3. `common/grpcx/metadata_test.go` — wire key + behavior contract

- `TestMetadataFieldContract` (line 15-31): locks `metadataFields` exactly `[(authorization,authorization),(user-id,x-user-id),(user-name,x-user-name),(dept-code,x-dept-code),(auth-type,x-auth-type)]`; locks gRPC keys lowercase.
- `TestMetadataRoundTripAllFields` (line 33-56): round-trip of all 5 fields; locks non-ASCII `用户名` → `b64:`+Base64 (line 44-46); locks restored `roundTrip.Value(field.contextKey)` equals input (line 51-54) — **uses the string context keys directly**.
- `TestMetadataOverwriteAndFirstValueContract` (line 58-84): outgoing context `authorization` overwrites existing metadata; other keys preserved; source MD not mutated; incoming `values[0]` used; empty-first skipped.
- `TestMetadataFilteringAndPrintableContract` (line 86-101): empty `string("user-id")` skipped; non-string `string("user-name")=42` skipped; `dept-code="line\nbreak"` → `b64:`.

### 4. `common/grpcx/client_interceptor_test.go`

- `TestUnaryMetadataInterceptor` (line 14-45): `CtxUserIdKey` value → outgoing `HeaderUserId`.
- `TestStreamTracingInterceptor` (line 47-81): `CtxDeptCodeKey` value → outgoing `HeaderDeptCode`.

### 5. `common/grpcx/server_interceptor_test.go`

- `TestLoggerInterceptor` (line 14-42): incoming `HeaderUserName` → `authctx.GetUserName(ctx) == "alice"`.
- `TestStreamLoggerInterceptor` (line 44-72): incoming `HeaderUserId` → `authctx.GetUserId(stream.Context()) == "user-2"`; verifies stream is context-wrapping wrapper.

### 6. `common/mcpx/context_meta_test.go`

- `TestMetaCollectExtractAndRawStorage` (line 15-60):
  - Locks `ctxMetaKey == "_meta"` (line 16-18).
  - `CollectFromCtx` from string-keyed ctx → map `{"authorization":..., "user-id":...}`; empty `user-name` dropped; `float64(3)` dept-code dropped (non-string) (line 20-29).
  - `ExtractFromMeta` numeric dept-code `float64(12)` → `GetDeptCode() == "12"` (line 34-41).
  - `WithMeta`/`GetMeta` round-trip; **locks raw `string("_meta")` lookup works** (line 47-49); locks `GetMeta` reads from externally-created `string("_meta")` (line 50-53).
- `TestExtractTraceFromMeta` (line 62-72): trace propagation (non-auth).

### 7. `common/mcpx/auth_test.go`

- `TestDualTokenVerifierJWTPropagationContract` (line 13-48): locks `info.Extra` keys by **authctx constants** (`CtxUserIdKey`, `CtxUserNameKey`, `CtxAuthTypeKey`, `CtxAuthorizationKey`); `UserID == "42"` (numeric coercion); `Extra[CtxAuthorizationKey]` is raw token.

### 8. aigtw logic tests seeding context via string key

- `session_validation_test.go` (8×), `chat_resume_validation_test.go` (4×), `knowledge_validation_test.go` (2×): `context.WithValue(context.Background(), authctx.CtxUserIdKey, "user-1")` — these tests will compile-break if `CtxUserIdKey` changes type/visibility.

### 9. Interceptor installation test coverage

- `client_interceptor_test.go`, `server_interceptor_test.go` verify behavior but not installation sites (installation verified by grep of `zrpc.WithUnaryClientInterceptor` / `s.AddUnaryInterceptors` — see `04-transport-boundaries.md`).

### Summary of locked contracts relevant to typed-key migration

| Contract | Locked by |
|---|---|
| `ContextKeys` order + string type | authctx/context_test.go:10-18 |
| value-equality of string keys (`string("user-id")` ≡ `CtxUserIdKey`) | authctx/context_test.go:21, claims_test.go:35, metadata_test.go:88-90, mcpx/context_meta_test.go:47-53 |
| gRPC wire keys + order + lowercase | metadata_test.go:15-31 |
| b64: encoding for non-printable | metadata_test.go:44-46,98-99 |
| inject/extract value semantics (empty/non-string skip, first-value, overwrite) | metadata_test.go:58-101 |
| MCP `_meta` map keys = authctx strings; non-string filtered in Collect, coerced in Extract | mcpx/context_meta_test.go:20-41 |
| MCP Extra keys = authctx constants; raw token stored under `authorization` | mcpx/auth_test.go:37-47 |
| interceptor extraction (metadata → getters) | grpcx/server_interceptor_test.go:27,60 |
| interceptor injection (ctx → metadata) | grpcx/client_interceptor_test.go:33,66 |

All these tests use the **public string constants or literal string keys**. A typed-key migration that removes the public constants breaks compile of: authctx tests, grpcx tests, mcpx tests, and 3 aigtw logic test files.