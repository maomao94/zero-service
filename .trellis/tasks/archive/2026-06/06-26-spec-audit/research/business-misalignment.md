# Research: Business Misalignment in Spec Files

- **Query**: Where spec says one thing but code does another
- **Scope**: internal (codebase)
- **Date**: 2026-06-26

## Findings

### 1. gormx-guidelines.md: Upsert/IgnoreRecordNotFoundError behavior

**File**: `.trellis/spec/backend/gormx-guidelines.md`, line 22

**What's wrong**: Spec file listing says `callbacks.go` handles "审计用户注入（Create/Update）" and `RegisterGlobalCallbacks`. However, the spec doesn't mention that GORM's `IgnoreRecordNotFoundError` only controls SQL trace logging (not error suppression). This is correctly documented in `database-guidelines.md:81` but should be referenced from gormx-guidelines too.

**Recommendation**: Cross-reference `database-guidelines.md:81-82` about `IgnoreRecordNotFoundError` from `gormx-guidelines.md`.

### 2. djisdk-guidelines.md: Handler count claim

**File**: `.trellis/spec/backend/djisdk-guidelines.md`, line 74

**What's wrong**: Spec says "17 个 handler 回调" but the actual `handlers` struct at `option.go:37-52` contains 14 fields + 1 onlineChecker = 15 callbacks total. Counting may differ based on how StatusHandler/RequestHandler/DrcUpHandler are counted.

**Actual handlers struct fields** (option.go:37-53):
1. onFlightTaskProgress
2. onFlightTaskReady
3. onReturnHomeInfo
4. onCustomDataTransmissionFromPsdk
5. onCustomDataTransmissionFromEsdk
6. onHmsEventNotify
7. onRemoteLogFileUploadProgress
8. onOtaProgress
9. onUpdateTopo
10. onOsd
11. onState
12. onStatus
13. onRequest
14. onDrcUp
15. onlineChecker (not a true handler)

That's 14 handlers + 1 checker = 15 fields. The spec claims 17.

**Recommendation**: Re-count and update the "17 个" count in line 74.

### 3. database-guidelines.md: Time formatting function

**File**: `.trellis/spec/backend/database-guidelines.md`, line 112

**What's wrong**: Spec says:
```
Go 层使用 `carbon.CreateFromStdTime(m.ReportedAt).ToDateTimeMicroString()` 转换。
```

But `carbonx` package already sets global defaults via `init()` in `carbonx.go:7-15`, making explicit `carbon.CreateFromStdTime` potentially redundant. The spec should mention that `carbonx` init sets Shanghai timezone globally.

**Recommendation**: Update line 112-113 to mention `carbonx` package's global initialization and when explicit `CreateFromStdTime` is actually needed.

### 4. ✓ logging-guidelines.md context injection table — VERIFIED CORRECT

**File**: `.trellis/spec/backend/logging-guidelines.md`, lines 58-66

**Cross-check**: All 6 handlers' context injection fields verified against `common/djisdk/handler.go`:

- `HandleEvents` (line 93-100): `gateway_sn`, `method`, `tid`, `bid`, `need_reply`, `ts`, `ts_fmt` ✓
- `HandleOsd` (line 296-301): `device_sn`, `tid`, `bid`, `ts`, `ts_fmt` ✓
- `HandleState` (line 325-330): `device_sn`, `tid`, `bid`, `ts`, `ts_fmt` ✓
- `HandleStatus` (line 346-352): `gateway_sn`, `method`, `tid`, `bid`, `ts`, `ts_fmt` ✓
- `HandleRequests` (line 418-424): `gateway_sn`, `method`, `tid`, `bid`, `ts`, `ts_fmt` ✓
- `HandleDrcUp` (line 475-481): `gateway_sn`, `method`, `tid`, `bid`, `ts`, `ts_fmt` ✓

**mqttx 基础层** (client.go:312-317): `client`, `topic`, `topic_template`, `payload_bytes`, `payload_size` ✓

All entries in the spec table are accurate. No issue.

### 5. djicloud-models.md: "所有模型嵌入 LegacyBaseModel" claim

**File**: `.trellis/spec/backend/djicloud-models.md`, line 21

**What's wrong**: Spec says "所有模型嵌入 `gormx.LegacyBaseModel`". This is architecturally plausible but needs verification against actual `app/djicloud/model/gormmodel/*.go` files. The `LegacyBaseModel` (model_legacy.go:43-48) embeds `LegacyIDMixin` (int64 id), `LegacyTimeMixin` (create_time/update_time), `LegacySoftDeleteMixin` (delete_time/del_state), and `VersionMixin`. If any new models use `TenantModel` or `BaseModel` (uint id, standard timestamps) instead, the spec would be wrong.

Cannot verify without reading actual model files, but the spec should be checked.
