# Research: Risk & migration surface analysis

- **Query**: Which call sites break if a public string key disappears; which readers rely on fallback/dual-read; whether code compares context values or writes non-string values; wire keys that MUST remain strings.
- **Scope**: internal (synthesis of `01`–`05`)
- **Date**: 2026-08-14

## Findings

### 1. Typed-key migration surface

#### 1.1 Candidate unified API (observational only — design is implement-phase's job)

Current public surface that a unified typed setter/getter would replace or wrap:
- 5 getters: `GetUserId`, `GetUserName`, `GetAuthorization`, `GetDeptCode` (no `GetAuthType`).
- Writers are NOT exposed as setters — the repo has **no `SetUserId`-style API**; writers use `context.WithValue` with string keys directly (gateways, socketiox, transport extraction) or go-zero middleware (out-of-repo).
- Claim helpers: `ExtractFromClaims`, `ApplyClaimMapping`, `ApplyClaimMappingToCtx`, `ClaimString`, `ContextKeys`.

#### 1.2 Which public string constants are read OUTSIDE `common/authctx` (i.e. would need shim/migration)

Compile-time (symbol references to `authctx.Ctx*` / `authctx.ContextKeys`):
- `common/grpcx/metadata.go:29-33` — `CtxAuthorizationKey`, `CtxUserIdKey`, `CtxUserNameKey`, `CtxDeptCodeKey`, `CtxAuthTypeKey` (wire mapping)
- `common/mcpx/context_meta.go:15,31` — `ContextKeys` iteration
- `common/mcpx/auth.go:30,43,46-48,58,61` — `CtxAuthTypeKey`, `ApplyClaimMapping`, `ContextKeys`, `ClaimString`, `CtxUserIdKey`, `CtxAuthorizationKey`
- `zerorpc/internal/logic/generatetokenlogic.go:49` — `CtxUserIdKey`
- `socketapp/socketpush/internal/logic/gentokenlogic.go:61,68` — `CtxUserIdKey`
- `gtw/gtw.go:60`, `aiapp/aigtw/aigtw.go:83-84,94`, `socketapp/socketgtw/socketgtw.go:68` — `CtxAuthTypeKey`, `CtxAuthorizationKey`, `ApplyClaimMappingToCtx`
- `common/socketiox/server.go:537,558,579,594,610,673,698,730,754` — `CtxAuthorizationKey`
- `common/tool/userutil.go:11,38,65` — getters
- 16 aigtw solo logic files — `GetUserId`
- `gtw/internal/logic/user/getcurrentuserlogic.go:34` — `GetUserId`
- `aiapp/mcpserver/internal/tools/echo.go:26-27` — `GetAuthorization`, `GetUserName`
- `facade/streamevent/internal/logic/upsocketmessagelogic.go:30` — `GetAuthorization`
- 11 trigger logic files via `tool.GetCurrentUserId` — indirect `GetUserId`
- Test files: `common/grpcx/*_test.go`, `common/mcpx/*_test.go`, `common/authctx/*_test.go`, 3 aigtw validation test files.

Runtime (value-equality reads, not symbol references — these break only behaviorally):
- go-zero `rest/handler/authhandler.go:72-78` writes raw claim names (`user-id`, `user_id`, `user-name`, `dept_code`, …) as string ctx keys.
- Wire metadata/MCP `_meta` keys are the string values themselves; receivers extract by string key.
- Tests using literal `string("user-id")` (authctx/context_test.go:21, claims_test.go:35, metadata_test.go:88-90, mcpx/context_meta_test.go:47-53).

#### 1.3 Removal criteria (observational)

A public string constant can only disappear after BOTH:
1. No symbol references outside `common/authctx` remain (compile check).
2. No runtime writer/reader relies on value-equality with the string literal — specifically the **out-of-repo go-zero JWT claims writer** and **wire extraction restoring to string keys** (grpcx `ExtractFromGrpcMD`, mcpx `ExtractFromMeta`) must be migrated to typed keys, and any external/legacy producers neutralized.

The two wire-facing functions (`grpcx.InjectToGrpcMD`/`ExtractFromGrpcMD`, `mcpx.CollectFromCtx`/`ExtractFromMeta`) are the only places that must translate between typed process keys and string wire keys. `ContextKeys` (string slice) exists solely to serve those two + mcpx auth Extra — a typed-key design would replace it with a typed list + string-name mapping table.

### 2. Risks — where typed-key-only migration would change behavior

| # | Risk | Evidence | Why it breaks |
|---|---|---|---|
| R1 | **go-zero JWT claims→context writer is out-of-repo and writes raw string keys.** | `authhandler.go:72-78`; gateways use `rest.WithJwt` (gtw/routes.go:151, aigtw/routes.go:37,50,83,198,211,226) and NEVER call a typed setter for user-id/user-name/dept-code. | If `CtxUserIdKey` becomes a typed key and `GetUserId` no longer reads the string key, JWT-authenticated identity (P1/P2) silently disappears at every aigtw/gtw handler — unless a bridge middleware or legacy-read fallback is added. **Highest-value risk.** |
| R2 | Non-string claim values (int64 user-id from zerorpc issuer). | `generatetokenlogic.go:49` (int64), `gentokenlogic.go:61` (string); `context_test.go:37` locks `""` for non-string. | A typed setter that stores only strings would preserve the current empty-string behavior (audit R1: int64 tokens already yield `""`). But a typed setter that converts types (like `ClaimString`) would **change behavior** and mix in claim-normalization (forbidden by audit §8.2). |
| R3 | Socket.IO writes raw token to 9 event contexts. | `server.go:537,558,579,594,610,673,698,730,754`. | All 9 sites must move to typed setter together; missing one creates asymmetric reads (some events carry auth, others don't). Also `OnAuthentication` gates these writes only at connect (tokenValidator), not per event. |
| R4 | `auth-type` is write-only; removing/changing its key has no reader impact, but `x-auth-type` wire key still emitted. | grep: no `GetAuthType`; only writers (gtw.go:60, aigtw.go:83, socketgtw.go:68, mcpx/auth.go:30,47). | Low risk for reads, but the wire key must remain `x-auth-type` for metadata contract (metadata_test.go:15-31). |
| R5 | MCP `CollectFromCtx` currently serializes raw Authorization into `_meta` to external servers. | `client.go:788,825`; audit P5. | Typed-key migration must keep `CollectFromCtx` behavior (or policy decides otherwise in `enforce-authorization-policy`, NOT here — audit §8.1 forbids changing raw-token propagation in this task). |
| R6 | Dual-read fallback pattern in `tool/userutil.go`. | `userutil.go:11,38,65` — reads `authctx.GetX(ctx)` first, then reflect fallback to request object fields. | If typed getters change, the ctx-first precedence must stay; trigger logic depends on gRPC-extracted ctx (P7) — trigger handlers have no local JWT, so their user-id comes exclusively from `ExtractFromGrpcMD`. |
| R7 | Value-equality tests lock literal-string reads. | `context_test.go:21`, `claims_test.go:35`, `metadata_test.go:88-90`, `context_meta_test.go:47-53`. | These tests must be rewritten alongside the migration or they fail; they also prove external writers (go-zero) can still be read via string key — a behavior some code may rely on implicitly. |
| R8 | Trigger gRPC invoker relays ctx via `InjectToGrpcMD`. | `grpc_invoker.go:35`; `trigger/internal/svc/servicecontext.go:84`. | Any relay service depends on `InjectToGrpcMD` reading typed keys; if extraction writes typed keys but injection reads string keys (or vice versa), the chain silently drops identity. |
| R9 | `gormx` uses its own typed key (`gormx:user`) — unrelated but shows a second identity carrier. | `user_context.go:8`, `tenant_scope.go:67-76`. | No overlap with authctx today; must not be accidentally conflated in migration (audit §6: gormx reads its own UserContext, not authctx). |
| R10 | Claim mapping direction (`internalKey → externalKey`) encoded in configs. | `aigtw.yaml:15-18`, `mcpserver.yaml:22-25`; `ApplyClaimMapping`/`ApplyClaimMappingToCtx`. | Typed internal keys change the "internal" side of the mapping keys; config strings (YAML keys `user-id`, `user-name`, `dept-code`) are wire/claim names and must stay strings. |

### 3. Must-not-combine boundaries (audit §8.1, §8.2, §8.3)

From `audit-report.md`:
- **§8.1 typed-auth-context-keys — 禁止混合**: do NOT change claim acceptance types, metadata duplicate behavior, raw-token propagation, `b64:`, or method policy. Preserve 5 wire strings and order (`authctx/context.go:14-20`, `grpcx/metadata.go:27-34` contract tests lock them).
- **§8.2 normalize-auth-claims — 禁止混合**: do NOT switch typed-key and claim normalization simultaneously; do not shrink Authorization; do not force transport conflicts.
- **§8.3 enforce-authorization-policy — 禁止混合**: do not upgrade `b64:`; do not bundle claim normalization or typed-key semantic changes; no global Authorization deletion without per-boundary evidence.
- **§7.3**: keep wire key + `b64:` contract during this task.
- **§8.4 sequence**: audit → typed keys → claim normalization → policy enforcement, each with manual gates. This task is step 2.

### 4. Wire keys that MUST remain strings

- MCP/JWT/process claim-name keys: `user-id`, `user-name`, `dept-code`, `authorization`, `auth-type` (authctx constants — wire names for MCP `_meta`, JWT claims, and gRPC context-key mapping).
- gRPC metadata keys: `x-user-id`, `x-user-name`, `x-dept-code`, `authorization`, `x-auth-type` (grpcx constants — wire only, never process keys).
- MCP `_meta` marker: `_meta` (ctxMetaKey).
- Config claim-mapping keys in YAML: `user-id`, `user-name`, `dept-code` (internal side) and `user_id`, `user_name`, `dept_code` (external side).

### 5. Known unknowns / decision inputs for design phase

| # | Unknown | Notes |
|---|---|---|
| U1 | Does any deployment rely on reading go-zero raw claim keys (e.g. `user_id` with underscore) directly via `ctx.Value`? | Repo code only reads via authctx getters; go-zero writes raw claim names; `ApplyClaimMappingToCtx` copies `user_id`→`user-id` only when token carries `user_id`. |
| U2 | Which token issuer is in production (zerorpc int64 `user-id` vs socketpush string `user-id` vs external `user_id`)? | audit R1/U7 — decides whether typed getter must handle non-string values or whether normalization is a later task. |
| U3 | Are there external services (out of repo) that inject authctx string keys into gRPC metadata? | Would be restored to typed keys by `ExtractFromGrpcMD`; wire compatibility keeps this working. |
| U4 | Does the aigtw raw-header middleware write need to remain (R2)? | audit R2: keep in this task; route-gated write restriction belongs to `enforce-authorization-policy`. |
| U5 | `SocketMetaData` config claim keys vs authctx keys. | `socketgtw.yaml:29` lists claim keys for Session metadata (string copy) — separate from authctx; verify no overlap expectations. |

### 6. Negative searches (no results)

- No `ctx.Value(authctx.XKey)` direct call sites outside authctx/grpcx/mcpx/tests.
- No `GetAuthType` getter or reader of `auth-type` value.
- No business consumer of `mcpx.GetMeta` (raw `_meta` map).
- No string-literal `context.WithValue(ctx, "user-id", ...)` etc. outside tests and go-zero dependency.
- No authctx usage in `common/einox`, `aiapp/aichat`, `app/iec*`, `app/gis`, `app/file`, `app/logdump`.
- No duplicate-value comparison or rejection of gRPC metadata anywhere.
- No code comparing context values for equality (only type assertions).
- Non-string context values exist only in: go-zero JWT claim writes (raw claim types), tests (int/float64), `ApplyClaimMappingToCtx` (copies any value type).