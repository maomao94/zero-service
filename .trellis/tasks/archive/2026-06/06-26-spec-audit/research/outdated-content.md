# Research: Outdated Content in Spec Files

- **Query**: Cross-check .trellis/spec/ claims against actual codebase
- **Scope**: internal (codebase)
- **Date**: 2026-06-26

## Findings

### 1. djisdk-guidelines.md: Config struct tags outdated

**File**: `.trellis/spec/backend/djisdk-guidelines.md`, lines 46–50

**What's wrong**: Spec claims `Config` struct has `json:",optional"` and `json:",default=true"` tags on Reply/Drc fields:

```go
// Spec says (line 48-49):
Reply      ReplyConfig   `json:",optional"`
Drc        DrcConfig     `json:",optional"`
```

**Actual code** (`common/djisdk/client.go:32-33`):
```go
Reply      ReplyConfig
Drc        DrcConfig
```

No `json:",optional"` tag, no `json:",default=30s"` on `PendingTTL` either. The tags were removed. Additionally, the spec's Config struct definition at lines 44-50 does not match the actual struct's tagless fields.

**Recommendation**: Update lines 48-49 to remove the `json:",optional"` tags and align with actual `client.go:29-34`.

### 2. gormx-guidelines.md: BaseModel location misidentified

**File**: `.trellis/spec/backend/gormx-guidelines.md`, line 23

**What's wrong**: Spec lists `model.go` as containing `BaseModel`:
```
| `model.go` | 基础模型 `BaseModel`（`Id`/`CreatedAt`/`UpdatedAt`/`DeletedAt`） |
```

**Actual code**: `BaseModel` is defined in `common/gormx/model_audit.go:4-10`, NOT in `model.go`. `model.go` contains `IDModel`, `StringIDModel`, `TimeMixin`, `SoftDeleteMixin`, `VersionMixin`, `TenantMixin`.

**Recommendation**: Update line 23 to point to `model_audit.go` instead of `model.go`. Also update the description since `BaseModel` now embeds `IDModel`, `AuditMixin`, `VersionMixin`, `SoftDeleteMixin`, `TimeMixin` (not `Id`/`CreatedAt`/`UpdatedAt`/`DeletedAt` directly).

### 3. gormx-guidelines.md: File listing incomplete

**File**: `.trellis/spec/backend/gormx-guidelines.md`, lines 8-29

**What's wrong**: The file organization table lists 18 files but omits several that exist in `common/gormx/`:

- `model_tenant.go` — `TenantModel`, `TenantStringIDModel`, `TenantTimeModel` (36 lines)
- `model_legacy.go` — `LegacyBaseModel`, `LegacyIDMixin`, `LegacySoftDeleteMixin` (56 lines)
- `driver.go` — database driver utilities
- `model_tenant_test.go` — tenant model tests (23 lines)

**Recommendation**: Add the missing files to the file organization table.

### 4. djisdk-guidelines.md: handler option spec inconsistent with actual handler naming

**File**: `.trellis/spec/backend/djisdk-guidelines.md`, lines 82-84

**What's wrong**: Spec lists handler options like `WithCustomDataFromPsdkHandler` and `WithCustomDataFromEsdkHandler` but the actual `option.go` uses:
- `WithCustomDataTransmissionFromPsdkHandler` (line ~88 in option.go)
- `WithCustomDataTransmissionFromEsdkHandler` (line ~95 in option.go)

Also, the internal `handlers` struct fields use `onCustomDataTransmissionFromPsdk` / `onCustomDataTransmissionFromEsdk` (option.go:41-42), not `onCustomDataFromPsdk` / `onCustomDataFromEsdk` as the spec implies.

**Recommendation**: Update the handler option names in the spec to match the actual function names in `option.go:37-52`.

### 5. logging-guidelines.md: mqttx context injection field names mismatched

**File**: `.trellis/spec/backend/logging-guidelines.md`, lines 46-53

**What's wrong**: Spec says `processMessage` injects:
```
logx.Field("payload_bytes", len(payload)),
logx.Field("payload_size", tool.DecimalBytes(int64(len(payload)), 1)),
```

**Actual code** (`common/mqttx/client.go:312-319`): Uses different fields based on actual `processMessage` implementation. Need to verify actual field names.

**Recommendation**: Verify actual context fields in `common/mqttx/client.go:312-319` and update spec.

## Caveats / Not Found

- The spec's `Config` tag differences (json:",optional" etc.) may be intentional as the spec describes the expected/ideal form rather than current code. Need to determine whether to update spec or code.
