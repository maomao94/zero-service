# Research: Must-not-combine Verification + Index

- **Query**: Confirm no current code in this task's scope should touch b64 codec, wire key names/order, `normalizeClaimString`/`ClaimString`, typed-key structure, method policy registry (none exists); confirm the three log fixes are independently deployable
- **Scope**: internal (verification)
- **Date**: 2026-08-14

## Research Index

| File | Contents |
|---|---|
| `senders-inventory.md` | 17 gRPC sender install sites + 2 MCP `_meta` call sites; per-site raw-token emission; no per-call suppression exists |
| `receivers-inventory.md` | 21 gRPC receiver servers (23 interceptor registrations) + MCP `WithExtractUserCtx` restore path; single-chokepoint extraction |
| `duplicate-conflict-surface.md` | Exact `ExtractFromGrpcMD` discard behavior (`values[0]`, empty-first, b64, no cross-check); observability gaps; contract test locks |
| `log-leakage-fixes.md` | L1/L2/L3 exact lines, redact shapes, test locks (L3 `auth_test.go:46-48` locks Extra, not log text) |
| `config-gray-rollout-surface.md` | `conf.MustLoad` YAML pattern, existing flag patterns, natural homes for policy mode/allowlist |
| `mcp-meta-policy-surface.md` | Where to gate `_meta` raw token; `CollectFromCtx` has no options param; test locks; per-server `ServerConfig` natural home |

## Must-not-combine verification

| Item | Status | Evidence |
|---|---|---|
| b64 codec (`b64:` prefix, base64 encode/decode) | **Do NOT touch** — audit §3/§7.3 and `08-13-extract-grpc-raw-codec/prd.md:65-80` keep it; tests lock it | `common/grpcx/metadata.go:19,36-43,54-56,68-70`; `metadata_test.go:44-47,98-101`; `common/grpcx/rawcodec.go` is a separate codec (trigger invoke) also untouched |
| Wire key names/order (`authorization`, `x-user-id`, `x-user-name`, `x-dept-code`, `x-auth-type`) | **Do NOT touch** — order is a contract | `metadata.go:27-34`; `metadata_test.go:15-31` `TestMetadataFieldContract`; `authctx/context.go:18-24` |
| `normalizeClaimString` / `ClaimString` | **Do NOT touch** — belongs to `normalize-auth-claims` child task (audit §8.2) | `common/authctx/claims.go:37-56`; `claims_test.go` (referenced in audit) |
| Typed-key structure (`WithKey`/`GetByKey`/typed keys) | **Do NOT touch** — belongs to `typed-auth-context-keys` child task (audit §8.1) | `common/authctx/context.go:26-135`; `BridgeJWTClaims` `:146-158` |
| Method policy registry | **None exists** — no registry to modify; an allowlist would be net-new (do not imply one exists) | grep for `MethodPolicy|methodPolicy|PolicyMode|policyMode|ClaimOnly|DelegationMode` = 0 matches in non-test Go code |
| L1/L2/L3 log fixes independent | **Yes — separate files**, independent of each other and of interceptor/policy work | `facade/streamevent/internal/logic/upsocketmessagelogic.go:30-31`; `aiapp/mcpserver/internal/tools/echo.go:25-28`; `common/mcpx/auth.go:45-65` |
| Raw-token propagation sender policy | In-scope (this task) — but must be **config/cohort-gated**, not a blanket deletion (audit §8.3: 禁止在无逐边界证据下全局删除 Authorization) | see `senders-inventory.md`, `config-gray-rollout-surface.md` |
| Duplicate/conflict handling | In-scope (this task) — report-only observability first (audit §4.2/§7.1 step 2), enforcement per approved rule | see `duplicate-conflict-surface.md` |

### Independence of the three log fixes

- L1 file: `upsocketmessagelogic.go` — touched by nothing else in this task.
- L2 file: `echo.go` (tools package) — touched by nothing else in this task (meta-policy work touches `context_meta.go`/`wrapper.go`/`client.go`, not echo.go's handler).
- L3 log format: `auth.go:65` only — safe. L3 Extra-content (`auth.go:61`) is entangled with `auth_test.go:46-48` and the `_meta`/tool-context contract; treat separately (see `log-leakage-fixes.md` and `mcp-meta-policy-surface.md`).
- No test asserts any of the three log text strings.

## Highest-value design decision for the user

**The single highest-value decision the user must make before dev: which per-boundary propagation modes are approved for P1 (aigtw→aichat/aisolo), P3 (socketgtw→streamevent), P5 (MCP `_meta`), and P7 (generic nested gRPC) — i.e. the delegation allowlist question (audit U1).**

Rationale: every other workstream hangs off this classification:
- Receiver-first compatibility mode (what the receiver must keep accepting) depends on which senders still emit raw tokens.
- The sender-cohort switch (S1/S2/S5/S6-S9) depends on which paths are `claims-only`/`none`.
- The MCP `_meta` gate (`CollectFromCtx` filter vs per-server config) depends on whether any external MCP server/tool is allowed to receive raw user credentials (U5).
- Default-deny baseline (audit §3.1) says **no path is `user-token`** unless the business owner explicitly grants delegation; today no allowlist exists, so the default answer is claims-only/none per P1-P7, service-token for P4/P8.

A second, tightly-related decision the user must confirm (audit U4): the duplicate/conflict rule — reject duplicates / require equality / first-value-only — which the enforcement phase cannot pick unilaterally (audit §4.2).

## Caveats / Not Found

- No method-level policy registry exists (verified); do not design this task around one.
- The "26 receiver sites" in the task brief does not match grep evidence (23 interceptor registrations across 21 service files); the full list is in `receivers-inventory.md`.
- All findings are from current source on 2026-08-14; archived audit report + its research docs are the authoritative classification input.
