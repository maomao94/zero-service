# Research: Consumer Impact Table

- **Query**: Which consumers read each identity claim, and what a bad value (bool/array/map/fractional/oversized/empty) does
- **Scope**: internal
- **Date**: 2026-08-14

## user-id — the only security-relevant claim

### aigtw → aisolo data isolation (17 files, security-critical)

All `aiapp/aigtw/internal/logic/solo/*.go` read `authctx.GetUserId(l.ctx)` and place it into the aisolo **request field** `UserId`:

| File | Behavior when user-id present | Behavior when empty |
|---|---|---|
| `createsessionlogic.go:33-36` | `userID` → `aisolo.CreateSessionReq.UserId` | `unauthenticatedError("missing user id in context")` |
| `chatlogic.go:37-48` | → `AiSoloCli.Ask` req `UserId` | unauthenticated |
| `getinterruptlogic.go:32-45` | → `GetInterruptReq.UserId` | unauthenticated |
| `listsessionslogic.go:27-34` | → `ListSessionsReq.UserId` | unauthenticated |
| `getsessionlogic.go:30-43` | → `GetSessionReq.UserId` | unauthenticated |
| `deletesessionlogic.go:30-43` | → `DeleteSessionReq.UserId` | unauthenticated |
| `listmessageslogic.go:30-43` | → `ListMessagesReq.UserId` | unauthenticated |
| `resumelogic.go:36-46` | → `ResumeReq.UserId` | unauthenticated |
| `bindsessionknowledgelogic.go:26-43` | → `BindSessionKnowledgeReq.UserId` | unauthenticated |
| `createknowledgebaselogic.go:31` | `uid` → `Knowledge.CreateBase(ctx, uid, name)` | `unauthenticatedError("missing user id")` |
| `deleteknowledgebaselogic.go:30` | → knowledge store scoped by user | unauthenticated |
| `deleteknowledgedocumentlogic.go:30` | → knowledge store scoped by user | unauthenticated |
| `ingestknowledgedocumentlogic.go:30` | → knowledge store scoped by user | unauthenticated |
| `ingestknowledgedocumentslogic.go:32` | → knowledge store scoped by user | unauthenticated |
| `listknowledgebaseslogic.go:27` | → knowledge store scoped by user | unauthenticated |
| `listknowledgedocumentslogic.go:30` | → knowledge store scoped by user | unauthenticated |
| `queryknowledgebaselogic.go:30` | → knowledge store scoped by user | unauthenticated |

**Receiver-side owner checks** (aisolo, request-field based):
- `aisolo/internal/logic/getinterruptlogic.go:48`: `rec.UserID != userID` → FORBIDDEN.
- Session/knowledge stores filter by `user_id` (e.g. `aisolo/internal/session/gormx_store.go:52,96,122-150`; `common/einox/knowledge/store_gorm.go:130-267`; `store_milvus.go:71,187,213-317`).

**Impact of bad user-id values**:
| Value type → string | Effect |
|---|---|
| empty `""` | aigtw rejects (unauthenticated) — safe failure |
| bool `"true"` | aigtw passes `UserId="true"`; aisolo creates/reads data under literal user `"true"` — **cross-user data isolation corruption/possible isolation confusion if an attacker can influence claims** (see caveat) |
| array `["a"]` | same as bool — literal-string key in aisolo stores |
| fractional `"1.5"` | user `"1.5"` — collision risk with distinct user ids that format identically |
| oversized rounded `"9007199254740992"` (float64 path) | aliases to user id `9007199254740992`; a **different** int64 id (>2^53) would collapse onto it — potential **audit/ownership mismatch** on the MCP/socket float64 path |
| map `{"a":1}` | literal JSON object string as user id — garbage isolation key |

Caveat: JWT is signed, so claim *type* is issuer-controlled; the permissive risk is mainly (a) issuer bugs/misconfiguration emitting wrong types, (b) MCP `_meta` path where a caller-provided `_meta` (service-authenticated only, `wrapper.go:241-250`) can inject arbitrary `user-id` types/values that then propagate to gRPC metadata and downstream gRPC services. The isolation itself is enforced against the **request field**, which aigtw derives from verified context — the gRPC-direct forgery risk is documented in the audit report §6 (trusts aigtw as sole trusted entry).

### gtw getCurrentUser (`gtw/internal/logic/user/getcurrentuserlogic.go:33-49`)

- Reads `authctx.GetUserId(l.ctx)`; empty → `UNAUTHORIZED 未登录`.
- Passes string `Id` into `zerorpc.GetUserInfoReq.Id` (proto string, `zerorpc.pb.go:950`); zerorpc `GetUserInfoLogic` converts with `convertor.ToInt` (`getuserinfologic.go:33-36`).
- **Impact**: fractional user-id `"1.5"` → `ToInt` → float64 `1.5` → `int64(1.5)` = **1** (truncated, no error!) → returns **wrong user's profile**. Oversized/rounded ids similarly collapse. Non-numeric `"true"`/`["a"]` → `ToInt` error → request fails (safe). This is a real correctness/audit-impact consumer: a bad user-id string yields a *different valid user* silently.
- Note: `convertor.ToInt` on string uses `strconv.ParseInt(v.String(), 0, 64)` (base 0) — see `04-numeric-precision-analysis.md`.

### trigger logic (~11 files via `tool.GetCurrentUserId`)

- `common/tool/userutil.go:10-34` `GetCurrentUserId` reads `authctx.GetUserId(ctx)` first; falls back to `CurrentUser.UserId` struct field (string only).
- Trigger consumers write the value into `CreateUser`/`UpdateUser` audit fields (e.g. `app/trigger/internal/logic/createplantasklogic.go:103-107`), or other business fields.
- **Impact**: bad value is stored in DB audit columns — data-integrity/audit mismatch, not direct authorization (no owner-check consumer found that compares it for isolation). Cosmetic-to-moderate.

### gormx audit helper

`common/gormx/user_context.go` `AuditUserValue` (116-143): string accepted; `""` → nil. Independent of authctx claims (fed by `WithUserContext`), not in claim path.

## user-name

| Consumer | Location | Bad-value impact |
|---|---|---|
| MCP echo tool | `aiapp/mcpserver/internal/tools/echo.go:25-40` | displays in tool output + Debug log — cosmetic; array `["a"]` renders as text |
| gormx audit helper | `common/gormx/user_context.go:85-91` | name display only |
| — | no authorization decision found in repo | none |

No security impact found. Worst case: cosmetic/display noise or log content.

## dept-code

- **No authorization consumer found** sourced from authctx. `tool/userutil.go:63-96` fallback reads first `Dept.DeptCode` from `CurrentUser` struct (unrelated to claims).
- Trigger has explicit `dept_code` request fields with validation (`trigger.pb.go:3899`, `submitcronjoblogic.go:56-57`) — these are request fields, **not** fed from authctx claims (audit §6: explicit request field and verified context metadata must not substitute for each other).
- Bad value impact: currently none (no consumer). If future scope filters by dept-code, permissive `"1.5"`-style conversions would matter.

## auth-type

- **No getter/consumer found** in repo (negative search: only writers at `gtw/gtw.go:59`, `aigtw/aigtw.go:82`, `socketgtw/socketgtw.go:67`, `mcpx/auth.go:30,47`; carried through `ContextKeys` generically).
- Bad value impact: none today; can be forged via gRPC/MCP metadata (audit §6).

## authorization (raw token)

- Not a "claim" in the numeric sense — always a string. Consumers: MCP echo Debug log (`echo.go:26`), StreamEvent Info log (`upsocketmessagelogic.go:31`), transport propagation. Covered by the audit's log-redaction items, out of scope for claim normalization.

## Summary of impact severity

| Claim | Consumers | Security impact of bad type/value | Cosmetic impact |
|---|---|---|---|
| `user-id` | aigtw solo (17), gtw getCurrentUser, trigger audit, knowledge stores | **HIGH** — cross-user isolation key; silent wrong-user lookup via `ToInt` truncation | low |
| `user-name` | echo, gormx audit | none found | yes |
| `dept-code` | none (authz) | none today | yes |
| `auth-type` | none | none today | yes |
