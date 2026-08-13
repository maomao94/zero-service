# Research: Direct readers of auth context

- **Query**: Every `ctx.Value(authctx.XKey)` / `authctx.GetX(ctx)` call site across the repo. Classify: process-context reads vs gRPC/MCP-extracted vs explicit request fields.
- **Scope**: internal (repo-wide)
- **Date**: 2026-08-14

## Findings

### A. Process-context readers (direct `authctx.GetX(ctx)` on the request/event ctx)

#### A1. aigtw solo logic — data isolation (user-id → request field)

All read `user-id` from the JWT-populated request context and reject when empty (`unauthenticatedError`):

| File:Line | Getter | Use |
|---|---|---|
| `aiapp/aigtw/internal/logic/solo/createsessionlogic.go:33` | `GetUserId` | `CreateSessionReq.UserId` (data isolation) |
| `aiapp/aigtw/internal/logic/solo/chatlogic.go:37` | `GetUserId` | `AskReq.UserId` |
| `aiapp/aigtw/internal/logic/solo/listsessionslogic.go:29` | `GetUserId` | list filter |
| `aiapp/aigtw/internal/logic/solo/getsessionlogic.go:30` | `GetUserId` | get filter |
| `aiapp/aigtw/internal/logic/solo/deletesessionlogic.go:30` | `GetUserId` | delete filter |
| `aiapp/aigtw/internal/logic/solo/listmessageslogic.go:30` | `GetUserId` | message filter |
| `aiapp/aigtw/internal/logic/solo/resumelogic.go:36` | `GetUserId` | resume validation |
| `aiapp/aigtw/internal/logic/solo/getinterruptlogic.go:32` | `GetUserId` | interrupt validation |
| `aiapp/aigtw/internal/logic/solo/createknowledgebaselogic.go:31` | `GetUserId` | KB isolation |
| `aiapp/aigtw/internal/logic/solo/deleteknowledgebaselogic.go:30` | `GetUserId` | KB delete |
| `aiapp/aigtw/internal/logic/solo/listknowledgebaseslogic.go:27` | `GetUserId` | KB list |
| `aiapp/aigtw/internal/logic/solo/ingestknowledgedocumentslogic.go:32` | `GetUserId` | KB ingest |
| `aiapp/aigtw/internal/logic/solo/ingestknowledgedocumentlogic.go:30` | `GetUserId` | KB ingest single |
| `aiapp/aigtw/internal/logic/solo/listknowledgedocumentslogic.go:30` | `GetUserId` | KB docs list |
| `aiapp/aigtw/internal/logic/solo/deleteknowledgedocumentlogic.go:30` | `GetUserId` | KB doc delete |
| `aiapp/aigtw/internal/logic/solo/queryknowledgebaselogic.go:30` | `GetUserId` | KB query |
| `aiapp/aigtw/internal/logic/solo/bindsessionknowledgelogic.go:26` | `GetUserId` | session-KB bind |

(16 files; all reject empty user-id with `unauthenticatedError`, see `helpers.go:20`.)

#### A2. gtw current-user logic

| File:Line | Getter | Use |
|---|---|---|
| `gtw/internal/logic/user/getcurrentuserlogic.go:34` | `GetUserId` | calls `ZeroRpcCli.GetUserInfo(Id: userId)`; rejects when empty |

#### A3. tool/userutil.go — indirect reader used by trigger logic

`common/tool/userutil.go`:
- `GetCurrentUserId(ctx, currentUser)` → `authctx.GetUserId(ctx)` first, then reflect fallback to object field (line 11)
- `GetCurrentUserName(ctx, currentUser)` → `authctx.GetUserName(ctx)` (line 38)
- `GetCurrentDeptCode(ctx, currentUser)` → `authctx.GetDeptCode(ctx)` (line 65)

Callers of `tool.GetCurrentUserId` (all pass `currentUser=nil`, i.e. rely on process context user-id):
- `app/trigger/internal/logic/createplantasklogic.go:103` (CreateUser/UpdateUser audit)
- `app/trigger/internal/logic/resumeplanlogic.go:75,92`
- `app/trigger/internal/logic/pauseplanlogic.go:72,89`
- `app/trigger/internal/logic/terminateplanlogic.go:80,101`
- `app/trigger/internal/logic/terminateplanexecitemlogic.go:91`
- `app/trigger/internal/logic/pauseplanexecitemlogic.go:91`
- `app/trigger/internal/logic/pauseplanbatchlogic.go:84`
- `app/trigger/internal/logic/resumeplanbatchlogic.go:82,98`
- `app/trigger/internal/logic/resumeplanexecitemlogic.go:77`
- `app/trigger/internal/logic/terminateplanbatchlogic.go:90`

These run in trigger gRPC handlers; context comes from `grpcx.LoggerInterceptor` → `ExtractFromGrpcMD` (incoming gRPC metadata). So they read **gRPC-extracted process context** (P7 relay), not local JWT.

#### A4. MCP echo tool

`aiapp/mcpserver/internal/tools/echo.go:26-27` — `GetAuthorization(ctx)`, `GetUserName(ctx)`; logged at Debug (audit L2). Context is restored by `mcpx.CallToolWrapper(..., WithExtractUserCtx())` (echo.go:44) from `_meta`.

#### A5. StreamEvent log

`facade/streamevent/internal/logic/upsocketmessagelogic.go:30` — `GetAuthorization(l.ctx)`; logged at Info (audit L1). Context from gRPC metadata extraction (socketgtw→streamevent, P3).

### B. Transport-extraction readers (restore to context keys, not consumers)

| File:Line | Function | Reads |
|---|---|---|
| `common/grpcx/metadata.go:50` | `InjectToGrpcMD` | `ctx.Value(f.contextKey).(string)` — reads all 5 process keys to emit wire metadata |
| `common/grpcx/server_interceptor.go:13,25` | `LoggerInterceptor` / `StreamLoggerInterceptor` | `ExtractFromGrpcMD(ss.Context())` — incoming gRPC metadata → process ctx |
| `common/mcpx/context_meta.go:16` | `CollectFromCtx` | `ctx.Value(key).(string)` for each `ContextKeys` — process ctx → `_meta` map |
| `common/mcpx/context_meta.go:32` | `ExtractFromMeta` | `meta` map → process ctx |
| `common/mcpx/wrapper.go:250` | `CallToolWrapper` (opt-in) | `ExtractFromMeta` |

### C. Explicit request-field consumers (NOT process context — important distinction)

These read user identity from **explicit gRPC request fields** (populated by aigtw from process context, audit §6):

- `aiapp/aisolo/internal/logic/getinterruptlogic.go:48` — `in.GetUserId()` owner check
- `aiapp/aisolo/internal/turn/executor.go:154,210,294,339,444` — `in.UserID` session lookups, `einoxkb.WithAgentTurn(ctx, in.UserID, ...)`
- `aiapp/aisolo/internal/session/*` (gormx_store.go, jsonl_store.go, memory.go) — `sess.UserID` / `userID` filters
- `common/einox/knowledge/store_gorm.go`, `store_milvus.go`, `memory/gormx_storage.go` — `user_id` filters (userID passed as arg, not read from ctx)
- `common/einox/knowledge/turnctx.go:17,25` — `WithAgentTurn` / `UserIDFrom` (typed private key `turnCtxKey struct{}`; unrelated to authctx but a precedent of typed ctx key in repo)
- `app/trigger/...` — `in.DeptCode` explicit fields (audit §6: dept-code as explicit request field vs context metadata)

### D. gormx user_context — separate typed context (NOT authctx)

`common/gormx/user_context.go` — its own package-private `contextKey string` (`"gormx:user"`, line 8) with `WithUserContext`/`GetUserContext`; consumed by `model_audit.go`, `model_legacy.go`, `tenant_scope.go`. **Independent of authctx** — no authctx reads here. `tenant_scope.go:67,71` provides `WithTenantContext` / `WithUserAndTenantContext` as explicit writers.

### Classification summary

| Class | Readers | Source of value |
|---|---|---|
| Process-context, JWT-sourced | A1 (16× aigtw), A2 (gtw), A3 (trigger via userutil) | go-zero JWT middleware raw claim writes (+ optional `ApplyClaimMappingToCtx`) |
| Process-context, transport-sourced | A3 (trigger), A4 (MCP echo), A5 (StreamEvent) | gRPC `ExtractFromGrpcMD` / MCP `ExtractFromMeta` |
| Transport extraction (read+rewrite) | B | process ctx ↔ wire |
| Explicit request fields | C | request structs (aigtw→aisolo), NOT authctx |
| Independent typed ctx | D (gormx) | own key, no overlap |

### Negative findings

- No `ctx.Value(authctx.XKey)` direct call sites outside `common/authctx`, `common/grpcx/metadata.go`, `common/mcpx/context_meta.go`, and tests.
- No repo code reads `auth-type` (no getter exists).
- `mcpx.GetMeta` has no business consumers (only tests + wrapper comments).
- No authctx usage in `common/einox`, `aiapp/aichat`, `app/iec*`, `app/gis`, `app/file`, `app/logdump` (verified by import grep).