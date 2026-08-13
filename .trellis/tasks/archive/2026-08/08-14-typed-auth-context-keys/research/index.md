# Research Index: typed-auth-context-keys

- **Query**: Migrate process-context auth identity from public string keys (`common/authctx`) to package-private typed context keys with unified setters/getters, preserving wire keys and behavior.
- **Scope**: internal (repo-wide code inventory)
- **Date**: 2026-08-14

## Files

| File | Contents |
|---|---|
| `01-authctx-public-api-inventory.md` | Exported constants/keys, getters, setters, `ContextKeys` order, claim helpers, locking tests |
| `02-writers-inventory.md` | Every `context.WithValue(ctx, authctx.XKey, v)` / claims writer: HTTP gateways, go-zero JWT middleware, socketiox events, authctx claims, transport extraction |
| `03-readers-inventory.md` | Every `authctx.GetX(ctx)` / `ctx.Value` read: process-context readers, transport readers, explicit request-field consumers |
| `04-transport-boundaries.md` | gRPC metadata (inject/extract, order, b64), server/client interceptors, MCP `_meta` (CollectFromCtx, ExtractFromMeta, WithMeta, wrapper restore), wire-vs-process key split |
| `05-tests-locking-contract.md` | Tests that lock the string-key contract and wire key names |
| `06-risks-and-migration-surface.md` | Migration surface, compatibility shim need, removal criteria, must-not-combine boundaries from audit §8.1, known unknowns, negative searches |

## Top-level summary

- Auth identity lives in process context under **5 public string constants** (`common/authctx/context.go:5-11`), propagated over gRPC via 5 wire metadata keys (`common/grpcx/metadata.go:13-20`) and over MCP via `_meta` map (`common/mcpx/context_meta.go:10`).
- **Writers**: 3 HTTP gateways (global middleware), go-zero v1.10.3 JWT middleware (writes raw claim names as string keys — outside repo), 9 socketiox event contexts, authctx claim helpers, grpcx/mcpx extraction.
- **Readers**: 16 aigtw solo logic files (data isolation), gtw getCurrentUser, MCP echo tool, StreamEvent log, `tool/userutil.go` (used by ~11 trigger logic files), transport extraction.
- **Critical external writer**: go-zero `rest.WithJwt` writes every non-standard JWT claim into context using the **raw claim name string** as the key (`go-zero@v1.10.3/rest/handler/authhandler.go:72-78`). This is the source of `user-id` / `user_id` values in process context at gateways. It is out-of-repo code, so a typed-key-only migration must add a bridging layer (own middleware or legacy read fallback) or the JWT identity will silently vanish.
- **Highest-value risk**: **the go-zero JWT claims→context writer (raw string claim names) is outside the repo and bypasses any typed-key setter; removing public string keys without a bridge breaks all JWT-authenticated identity (P1/P2), because gateways never call a typed setter for user-id/user-name/dept-code — they rely on the middleware's raw-key writes.**

## Critical caveat

- Task script `task.py current` reports no active task, but the task directory `.trellis/tasks/08-14-typed-auth-context-keys/` exists with `prd.md`. Research written to `{TASK_DIR}/research/` as specified.
