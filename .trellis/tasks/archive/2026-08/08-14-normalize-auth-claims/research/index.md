# Research Index: normalize-auth-claims

- **Task**: `.trellis/tasks/08-14-normalize-auth-claims` (claim normalization planning)
- **Scope**: internal (read-only code research)
- **Date**: 2026-08-14
- **Parent gate**: audit archived at `.trellis/tasks/archive/2026-08/08-14-audit-authorization-propagation/audit-report.md` §8.2; typed-key task archived at `.trellis/tasks/archive/2026-08/08-14-typed-auth-context-keys/`.

## Files

| File | Contents |
|---|---|
| `01-current-conversion-matrix.md` | Input type → current `convertor.ToString` output → affected code path. Includes the **json.Number discovery** (go-zero gateway path) |
| `02-real-value-type-inventory.md` | Per claim: issuer → wire → parse → context value type at every boundary |
| `03-consumer-impact-table.md` | Which consumer reads which claim, and what a bad value does (security vs cosmetic) |
| `04-numeric-precision-analysis.md` | float64 vs json.Number precision, user-id ranges, concrete FormatFloat evidence, recommended conversion rules |
| `05-whitelist-and-reject-ignore-options.md` | Type whitelist + reject-vs-ignore semantics per claim — **options for user confirmation, NOT a decision** |
| `06-must-not-combine-and-wire-contract.md` | Out-of-scope boundaries (audit §8.2) and wire-level constraints |
| `07-unknowns-and-negative-searches.md` | Known unknowns and searches that returned nothing |

## Top-level summary

- **Current semantics are permissive by design and locked by tests**: `ClaimString` (via lancet `convertor.ToString` v2.3.9) converts bool → `"true"`, arrays → `["a"]`, maps → `{"k":"v"}`, fractional floats → `"1.5"`, integer floats → `"42"`. Tests `claims_test.go`, `context_test.go`, `mcpx/context_meta_test.go`, `mcpx/auth_test.go` explicitly lock these outputs.
- **Critical discovery**: go-zero v1.10.3 `rest/token/tokenparser.go:122-124` parses JWT with `jwt.WithJSONNumber()`, so claims reach HTTP context as `json.Number` (not `float64`). `convertor.ToString(json.Number("9007199254740993"))` → `"9007199254740993"` — **exact, no precision loss**. Only the repo's own `tool.ParseToken` (MCP verifier + socket validator) uses the default parser → numbers become `float64` → precision loss above 2^53.
- **user-id is the only security-relevant claim** (aigtw→aisolo data isolation via request field, 17 solo files). `user-name`/`dept-code` are informational; `auth-type` has no consumer. Empty user-id at aigtw → explicit unauthenticated error; at gtw → 401.
- **The single highest-value decision the user must make**: confirm the type whitelist and reject-vs-ignore semantics per claim (PRD requirement), specifically whether non-string, non-numeric (bool/array/map/fractional/oversized) values should **reject the request** or **be ignored (dropped → treated as missing)**. This decision is required before dev and before the permissive tests can be updated.

## Task script note

`task.py current` reports no active task; research written to `{TASK_DIR}/research/` as specified.
