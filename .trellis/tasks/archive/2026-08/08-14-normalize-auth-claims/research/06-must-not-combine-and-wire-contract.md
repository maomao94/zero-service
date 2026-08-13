# Research: Must-Not-Combine Boundaries and Wire Contract

- **Query**: Confirm in-scope vs out-of-scope code paths for claim normalization; wire-level constraints
- **Scope**: internal
- **Date**: 2026-08-14

## 1. Out-of-scope (must NOT be changed by this task) — from audit §8.2 and parent PRD

| # | Boundary | Evidence | Status for this task |
|---|---|---|---|
| N1 | **Do NOT combine typed-key rollout** | typed-key task archived (`08-14-typed-auth-context-keys`); `context.go` already uses package-private typed keys; `ContextKeys` order locked by `context_test.go:10-13` | Done in prior task; do not revisit |
| N2 | **Do NOT shrink `Authorization`** | parent PRD `prd.md:12` "不在本任务改变 Authorization 跨服务传播策略"; audit §8.2 | Out of scope — authorization stays propagated as today |
| N3 | **Do NOT enforce transport conflicts** | audit §8.2; §4.2 conflict rules (duplicate values, token-vs-x-user-*, empty-first) are for `enforce-authorization-policy` | Out of scope — leave `grpcx/metadata.go:63-74` first-value behavior unchanged |
| N4 | **Do NOT change wire key names / order** | `ContextKeys` (`context.go:20-26`), gRPC `metadataFields` (`metadata.go:27-34`), MCP `ContextKeys` — all locked by contract tests | Out of scope |
| N5 | **Do NOT change b64: codec** | audit §7.3/§8.3 | Out of scope |
| N6 | **Do NOT change method/tool policy registry** | belongs to `enforce-authorization-policy` | Out of scope |
| N7 | **Do NOT change log redaction** | L1–L3 are independent fixes (`audit-report.md:18`) | Out of scope (separate rollback unit) |

## 2. In-scope (claim conversion and its callers)

| # | Path | File:line | What claim normalization may touch |
|---|---|---|---|
| S1 | `ClaimString` | `common/authctx/claims.go:35-41` | central permissive converter → whitelist-aware converter |
| S2 | `ApplyClaimMapping` | `common/authctx/claims.go:25-31` | raw copy of external claims; may gain type validation on source values |
| S3 | `ExtractFromClaims` | `common/authctx/claims.go:10-20` | claims→ctx; currently unused in prod but in scope |
| S4 | `BridgeJWTClaims` + `toStringClaim` | `common/authctx/context.go:148-165` | gateway raw-key→typed-key conversion; must keep best-effort/ignore semantics (no error channel) unless a new error path is added |
| S5 | `NewDualTokenVerifier` | `common/mcpx/auth.go:22-72` | verifier may reject malformed claims (`auth.ErrInvalidToken`) or ignore them; `UserID`/`Extra` construction |
| S6 | `ExtractFromMeta` / `CollectFromCtx` | `common/mcpx/context_meta.go:12-37` | `_meta` restore/collect; whitelist application on restore side |
| S7 | `CallToolWrapper` extract path | `common/mcpx/wrapper.go:231-251` | only calls `ExtractFromMeta` when `WithExtractUserCtx`; behavior controlled by S6 |
| S8 | Socket claims gate | `common/socketiox/server.go:517-527` | currently string-only copy; may keep or tighten (already ignores non-strings) |
| S9 | Test contracts | `claims_test.go`, `context_test.go`, `mcpx/context_meta_test.go`, `mcpx/auth_test.go` | must be updated to the confirmed matrix |
| S10 | gRPC metadata inject/extract | `common/grpcx/metadata.go:46-74` | values are already strings; only format validation possible, not type conversion — leave unless wire-format check required |

## 3. Wire contract constraints

1. **gRPC metadata values are strings** (`grpcx/metadata.go:13-20`); inject reads `GetByKey` (string getters), extract writes strings. Normalization is **process-side**; no change to wire format.
2. **MCP `_meta` values are strings when collected from ctx** (`CollectFromCtx` only collects non-empty string getters, `context_meta.go:12-23`); server-side restore parses the JSON map. Wire remains JSON; process-side conversion only.
3. **JWT claim values on the wire** are JSON — numbers vs strings are an issuer decision. Normalization must accept both numeric-JSON (`json.Number`) and string-JSON forms to remain compatible with the two repo issuers (zerorpc int64 → number; socketpush string → string).
4. **Key naming on wire**: JWT claims use `user-id` (repo issuers) or `user_id` (config mapping) — see `02`; the mapping copy (`ApplyClaimMapping`) is in scope, but the wire key contract itself stays.
5. `b64:` prefix in gRPC metadata is untouched (N5).

## 4. Sequence constraint (parent PRD `prd.md:9-12`)

- This task must wait for user confirmation of the audit + typed-key tasks (already archived) and the type matrix + reject/ignore semantics (this research input).
- After dev, user must confirm compatibility/test results before archiving; only then does `enforce-authorization-policy` begin (audit §8.4).
