# Spec Audit Summary

- **Query**: Comprehensive audit of all .trellis/spec/ files
- **Scope**: internal (cross-checked against actual codebase)
- **Date**: 2026-06-26

## Files Audited

32 total spec files read:
- 26 in `.trellis/spec/backend/`
- 6 in `.trellis/spec/guides/`

## Issue Summary

| Category | Count | Severity |
|----------|-------|----------|
| Outdated content | 5 | Medium |
| Business misalignment | 4 | Medium-High |
| Splitting candidates | 3 | Low-Medium |
| Missing coverage | 8 | Low |
| Removal candidates | 6 | Low |

## Top Issues by Priority

### HIGH
1. **djisdk-guidelines.md:47-50** — `Config` struct tags (`json:",optional"`, `json:",default=30s"`) don't match actual code at `common/djisdk/client.go:29-34`. Tags have been removed.

### MEDIUM
2. **gormx-guidelines.md:23** — `BaseModel` location wrong (says `model.go`, actually in `model_audit.go`)
3. **gormx-guidelines.md:8-29** — File listing incomplete (missing `model_legacy.go`, `model_tenant.go`, `driver.go`)
4. **djisdk-guidelines.md:74** — Handler count claim ("17 个") doesn't match actual 14 handlers
5. **logging-guidelines.md:58-66** — VERIFIED CORRECT. All context injection fields match actual code in `handler.go`.

6. **messaging-guidelines.md** — 665 lines, 6 scenarios → split candidate
7. **iec104-control-commands.md** — 1022 lines → split candidate
8. **gisx-guidelines.md** — 676 lines, Docker/CGO section (139 lines) is deployment, not code spec

### LOW
9. Missing spec coverage for: `antsx.Reactor`, `common/configx/`, `common/carbonx/`, `common/tool/`, `common/trace/`, `common/dbx/`, `common/asynqx/`, `common/ssex/`
10. **uix-framework.md** — marked experimental, may be dead weight
11. **gisx-guidelines.md:108-247** — Docker build contract is deployment concern
12. **coding-standards.md:48-87** — Go generics section duplicates `bytex-contracts.md`

## Verified Correct (No Issues Found)

- `error-handling.md` — extproto.Code references, error factory signatures ✓
- `logging-guidelines.md` — `WithoutSQLTrace` exists in `gormx/logger.go:21` ✓
- `drc-concurrency.md` — Lock model description ✓
- `bytex-contracts.md` — `ConvertSlice`, `Integer` constraint in `bytex.go:8-19` ✓
- `messaging-guidelines.md` — MQTT reply router, `ClientOptions`, errors.go ✓
- `socketiox-*.md` — Event constants verified ✓
- `antsx-*.md` — Core API signatures ✓
- `database-guidelines.md` — Model generation scripts exist (`model/*.sh`) ✓
- `carbonx/carbonx.go` — Shanghai timezone confirmed ✓
- `coding-standards.md:92` — "bytex 中重复已清除" verified, no duplicate in `common/tool/` ✓
- `djicloud-hooks-guidelines.md` — Hook file listing matches actual files ✓
- `directory-structure.md` — Top-level directory descriptions match ✓
