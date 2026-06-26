# Research: Candidates for Removal

- **Query**: Sections that are obsolete, describe deleted features, or are empty boilerplate
- **Scope**: internal (spec files)
- **Date**: 2026-06-26

## Findings

### 1. coding-standards.md: "Go 1.26" version claim

**File**: `.trellis/spec/backend/coding-standards.md`, line 7

**What's wrong**: Spec says "Go 1.26" but this is an extremely new/future Go version. Verified `go.mod:3` says `go 1.26.0`. This is current but will become outdated quickly as Go evolves.

**Recommendation**: This is correct for now. Add version reference to `go.mod` so it's self-documenting.

### 2. quality-guidelines.md: "Go 泛型约定" belongs elsewhere

**File**: `.trellis/spec/backend/coding-standards.md`, lines 48-87

**What's wrong**: The "Go 泛型约定" section (lines 48-87) is detailed implementation guidance about `ConvertSlice`, `Integer` constraint, and bytex-specific patterns. This duplicates info from `bytex-contracts.md`.

**Recommendation**: Move Go generics conventions to `bytex-contracts.md` or a standalone file. Keep only the general principle in `coding-standards.md`.

### 3. database-guidelines.md: "Scenario: GaussDB PG 空串即 NULL" is service-specific

**File**: `.trellis/spec/backend/database-guidelines.md`, lines 31-69

**What's wrong**: The GaussDB PG NULL handling is specific to the GaussDB deployment scenario and the DJI service. It's a vendor-specific database quirk, not a general database guideline.

**Recommendation**: Consider moving to a deployment-specific spec or the `djicloud-models.md` spec, or keep as-is if GaussDB is the project's primary database.

### 4. gisx-guidelines.md: "Docker / CGO 镜像构建契约" is deployment, not code spec

**File**: `.trellis/spec/backend/gisx-guidelines.md`, lines 108-247

**What's wrong**: The Docker/CGO build contract section (139 lines) is about Dockerfile construction, multi-arch builds, deploy.sh patterns, and cache mount configuration. This is a deployment/build concern, not a code development spec. It belongs in a deployment guide, not a code spec.

**Recommendation**: Move to a separate `docker-build-guidelines.md` or the project's deployment documentation. Keep a short reference in gisx-guidelines.

### 5. uix-framework.md: marked experimental — potentially obsolete

**File**: `.trellis/spec/backend/uix-framework.md`, line 3

**What's wrong**: Flagged as "⚠️ 实验性代码" — "cli/uix/、cli/dtui/ 及其子模块由 AI 自动生成，未经人工审查，存在状态机边界问题、测试盲区和架构缺陷。生产环境不可用。非必要不要使用。"

If the code is not production-ready and not actively maintained, the spec file may be dead weight.

**Recommendation**: Determine if uix/dtui are still maintained. If not, archive the spec. If yes, update the experimental warning with current status.

### 6. trellis-template-policy.md: references platforms that may not be active

**File**: `.trellis/spec/backend/trellis-template-policy.md`, lines 19-29

**What's wrong**: References `.qoder/**` (line 29) — "Qoder 平台 hooks、agents、skills、settings". If Qoder is not actively used in this project, this line is noise.

**Recommendation**: Verify if Qoder is active. If not, note that the line is template boilerplate.

## Verified Correct (No Issues)

- `error-handling.md`: `extproto.Code` references are correct ✓
- `logging-guidelines.md`: Context injection patterns verified against actual code ✓
- `drc-concurrency.md`: Lock model description matches expected patterns ✓
- `antsx-*` specs: Core API signatures verified against `common/antsx/` ✓
- `bytex-contracts.md`: `Integer` constraint and `ConvertSlice` verified ✓
- `messaging-guidelines.md`: MQTT reply router patterns verified against `common/mqttx/` ✓
- `socketiox-guidelines.md` + `socketiox-contracts.md`: Event constants verified in `server.go` ✓
