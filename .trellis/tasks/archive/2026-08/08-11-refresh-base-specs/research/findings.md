# Base Specs Refresh Findings

## File: .trellis/spec/backend/index.md
Status: OK
Issues found:
- None. All 20 referenced spec files under `.trellis/spec/backend/` exist on disk. The reference to `../guides/index.md` is valid (`.trellis/spec/guides/index.md` exists with 4 entries). All domain spec files (trigger-guidelines.md, isp-guidelines.md, iec104-guidelines.md, dji-guidelines.md, gis-guidelines.md, realtime-guidelines.md, ai-guidelines.md) are present.

## File: .trellis/spec/backend/directory-structure.md
Status: OK
Issues found:
- None. Verified all referenced paths: `README.md` (exists), `docs/architecture.md` (exists), `go.mod` (exists, module `zero-service`), `app/`, `aiapp/`, `socketapp/`, `gtw/`, `facade/`, `common/`, `deploy/`, `docs/`, `third_party/` (all directories exist). `app/trigger/gen.sh`, `app/bridgegtw/gen.sh`, `aiapp/aisolo/gen.sh` (all exist). `docs/README.md` (exists). `common/gnetx` (exists). `app/xfusionmock` (exists). `1.7.1/` and `1.9.x/` (exist).

## File: .trellis/spec/backend/coding-standards.md
Status: OK
Issues found:
- None. All referenced files exist: `common/antsx/invoke.go`, `common/tool/errorutil.go`, `common/djisdk/drc.go`, `common/bytex/` (package with bytex.go), `common/isp/client.go`, `common/netx/client.go`, `common/djisdk/option.go`, `common/isp/errors.go`, `common/gtwx/` (package exists).

## File: .trellis/spec/backend/quality-guidelines.md
Status: OK
Issues found:
- None. All referenced test files/directories exist: `common/crontask/*_test.go` (crontask_test.go, describe_test.go), `common/gormx/*_test.go` (21 test files), `common/djisdk/protocol_drc_test.go`, `app/ispagent/internal/handler/*_test.go` (task_test.go).

## File: .trellis/spec/backend/go-zero-conventions.md
Status: OK
Issues found:
- None. All referenced directories and files exist: `app/trigger/internal/logic/`, `app/trigger/internal/svc/servicecontext.go`, `app/trigger/internal/server/`, `app/bridgegtw/internal/handler/`, `aiapp/aisolo/internal/logic/`. Note: trigger uses `internal/svc/` (not `internal/svc/servicecontext.go` with a different spelling — the file is valid as referenced).

## File: .trellis/spec/backend/contract-generation.md
Status: OK
Issues found:
- None. All referenced files exist: `app/trigger/trigger.proto`, `app/trigger/gen.sh`, `app/bridgegtw/bridgegtw.api`, `app/bridgegtw/gen.sh`, `aiapp/aisolo/aisolo.proto`, `aiapp/aisolo/gen.sh`, `third_party/` (directory exists).

## File: .trellis/spec/backend/service-lifecycle.md
Status: OK
Issues found:
- None. All referenced files exist: `app/trigger/internal/svc/servicecontext.go`, `app/ispagent/internal/svc/servicecontext.go`, `app/djicloud/internal/svc/servicecontext.go`, `app/trigger/trigger.go`, `common/crontask/crontask.go`, `common/antsx/replypool.go`, `common/wsx/client.go`.

## File: .trellis/spec/backend/error-handling.md
Status: OK
Issues found:
- None. All referenced files exist: `common/Interceptor/rpcclient/metadataInterceptor.go`, `common/Interceptor/rpcserver/loggerInterceptor.go`, `common/ctxprop/grpc.go`, `common/tool/errorutil.go`, `third_party/extproto.proto`, `common/gtwx/errorhandler.go`, `common/gtwx/openai_error.go`. Code patterns verified: `DJIError` type (error.go:11), `NewDJIError(code)` (error.go:22), `IsDJIError(err)` (error.go:39), `PlatformError` (error.go:50), `ErrSkipRequestReply` (error.go:72), `commandError(err)` helper (app/djicloud/internal/logic/helper.go:29).

## File: .trellis/spec/backend/common-package-design.md
Status: OK
Issues found:
- None. All referenced files and directories exist: `common/netx`, `common/mqttx`, `common/bytex`, `common/gisx`, `common/netx/client.go`, `common/djisdk/option.go`, `common/antsx/replypool.go`, `common/wsx/config.go`, `common/isp/serializer.go`, `common/flowx/`, `common/antsx/invoke.go`.

## File: .trellis/spec/backend/gormx-guidelines.md
Status: NEEDS UPDATE
Issues found:
- **Line 18** (模型组合 section): Spec says `AtomicModel`, `AtomicTenantModel`, `AtomicVersionModel` — these types **do not exist** in `common/gormx`. The actual types in `common/gormx/model.go` are:
  - `IDModel` (uint primary key, line 11)
  - `StringIDModel` (string primary key, line 16)
  - `TimeMixin` (created_at/updated_at, line 21)
  - `SoftDeleteMixin` (gorm.DeletedAt, line 27)
  - `VersionMixin` (optimisticlock.Version, line 32)
  - `TenantMixin` (tenant_id, line 37)
  - `AuditMixin` / `StringAuditMixin` / `AuditWithoutDeleteMixin` / `StringAuditWithoutDeleteMixin` (model_audit.go)

  The spec describes "combined atomic mixins" but the code uses fine-grained individual mixins that callers compose. The spec wording "不要嵌入多个重复定义主键/时间/软删字段的 mixin" (don't embed multiple mixins that redundantly define PK/time/soft-delete) is what the separate mixins themselves avoid — but the type names are wrong.
- **Line 20**: Spec references `LegacyModel` — this type **does not exist**. The actual legacy types in `common/gormx/model_legacy.go` are:
  - `LegacyIDMixin` (int64 pk, line 19)
  - `LegacyStringIDMixin` (string pk, line 24)
  - `LegacyBaseModel` (line 55)
  - `LegacyStringBaseModel` (line 77)
- **Lines 40-41**: Spec says query-no-record uses `ErrNotFound` and CAS-miss uses `ErrNoRowsUpdate` as gormx errors. These **do not exist** in `common/gormx`:
  - `common/gormx` has **no sentinel error variables at all** (no `errors.go` file, no `var Err` declarations)
  - `ErrNotFound` is defined in `common/crontask/errors.go:6` (`[crontask] task not found`)
  - `ErrNoRowsUpdate` is defined in `model/vars.go:20` (`update db no rows change`)
  - The gormx package uses GORM's built-in `gorm.ErrRecordNotFound` for not-found cases (evidenced in `logger.go:97-98`)

## File: .trellis/spec/backend/concurrency-guidelines.md
Status: OK
Issues found:
- None. Verified: `antsx.Invoke` (invoke.go:28), `antsx.InvokeAllSettled` (invoke.go:177), `antsx.Promise` (promise.go:17), `antsx.ReplyPool` (replypool.go:52), `antsx.Reactor` (exists in package). `common/djisdk/drc.go` and `common/socketiox/container.go` both exist.

## File: .trellis/spec/backend/messaging-guidelines.md
Status: OK
Issues found:
- None. All referenced files exist: `common/netx/client.go`, `common/netx/response.go`, `common/netx/upload.go`, `common/netx/download.go`, `common/wsx/client.go`, `common/wsx/config.go`, `common/mqttx/reply_router.go`, `common/mqttx/request_replyer.go`.

## File: .trellis/spec/backend/crontask-guidelines.md
Status: OK
Issues found:
- None. All referenced files exist: `common/crontask/crontask.go`, `common/crontask/config.go`, `common/crontask/store.go`, `common/crontask/*_test.go`, `common/crontask/memory_store.go`, `app/trigger/internal/cronjob/db_store.go`, `app/ispagent/internal/crontask/db_store.go`. Code patterns verified: `TaskConfig` struct (config.go:34-51) with fields `RRuleStr`, `NextRun`, `ScheduledTime`, `LastRun`, `LastScheduledRun` — all match spec. `DescribeRRule(value string) (string, error)` (describe.go:14) matches spec. `ResolveLockTimeout` with MinLockTimeout=30s exists (config.go:55).

## Summary

| Status | Count | Files |
|--------|-------|-------|
| OK | 12 | index.md, directory-structure.md, coding-standards.md, quality-guidelines.md, go-zero-conventions.md, contract-generation.md, service-lifecycle.md, error-handling.md, common-package-design.md, concurrency-guidelines.md, messaging-guidelines.md, crontask-guidelines.md |
| NEEDS UPDATE | 1 | gormx-guidelines.md |

## Key Issues (gormx-guidelines.md only)

1. **Model type names outdated** — spec references `AtomicModel`, `AtomicTenantModel`, `AtomicVersionModel`, `LegacyModel`; actual code uses `IDModel`/`StringIDModel` + `TimeMixin`/`SoftDeleteMixin`/`VersionMixin`/`TenantMixin` for new tables, and `LegacyIDMixin`/`LegacyStringIDMixin`/`LegacyBaseModel`/`LegacyStringBaseModel` for legacy tables.

2. **Error sentinels misplaced** — spec implies `ErrNotFound` and `ErrNoRowsUpdate` are in `common/gormx`; they are actually in `common/crontask/errors.go` and `model/vars.go` respectively. `common/gormx` has no sentinel errors and relies on `gorm.ErrRecordNotFound`.
