# Research: Files That Would Need Modification

- **Query**: If we remove repositories and use raw gormx.DB, which files need modification?
- **Scope**: internal
- **Date**: 2026-07-17

## Findings

### Category 1: ServiceContext (core wiring)

**1 file**:
| File | Changes |
|------|---------|
| `app/trigger/internal/svc/servicecontext.go` | Remove 4 repository fields (lines 46-49), remove `repos := gormmodel.NewRepositories(gormDB)` (line 65), remove repo assignments (lines 86-89) |

### Category 2: Gormmodel Package (can be simplified or removed)

**4 files**:
| File | Changes |
|------|---------|
| `app/trigger/model/gormmodel/repository.go` | Remove or simplify — contains 4 repository interfaces, `Repositories` struct, `repoBase`, helper functions (`findOne`, `createModel`, `updateModel`, `updateModelWithVersion`, `pageRows`, `selectRows`, etc.) |
| `app/trigger/model/gormmodel/plan_repositories.go` | Remove entirely — contains `planRepo`, `planBatchRepo` implementations |
| `app/trigger/model/gormmodel/exec_repositories.go` | Remove entirely — contains `planExecItemRepo`, `planExecLogRepo` implementations |
| `app/trigger/model/gormmodel/plan.go` | **Keep** — gorm struct definitions (`Plan`, `PlanBatch`, `PlanExecItem`, `PlanExecLog`) are still needed for `Model(&gormmodel.Plan{})` calls |

### Category 3: Logic Files Using Repositories (19 files)

All of these would need to change from `l.svcCtx.PlanModel.FindOne(...)` pattern to `l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.Plan{}).Where("id = ?", id).First(&result).Error`:

| File | Repository Methods Used |
|------|------------------------|
| `app/trigger/internal/logic/getplanlogic.go` | PlanModel.FindOne, PlanModel.FindOneByPlanId, PlanBatchModel.CalculatePlanProgress |
| `app/trigger/internal/logic/resumeplanlogic.go` | PlanModel.FindOne, PlanModel.FindOneByPlanId, PlanModel.Trans, PlanModel.UpdateWithVersion, PlanBatchModel.UpdateBuilder, PlanBatchModel.UpdateWithBuilder |
| `app/trigger/internal/logic/getplanbatchlogic.go` | PlanBatchModel.FindOne, PlanBatchModel.FindOneByBatchId, PlanExecItemModel.GetBatchStatusCounts, PlanExecItemModel.GetBatchTotalExecItems |
| `app/trigger/internal/logic/getplanexecitemlogic.go` | PlanExecItemModel.FindOne, PlanExecItemModel.FindOneByExecId |
| `app/trigger/internal/logic/createplantasklogic.go` | PlanModel.FindOneByPlanId, PlanModel.Trans, PlanModel.Insert, PlanBatchModel.Insert, PlanExecItemModel.Insert |
| `app/trigger/internal/logic/callbackplanexecitemlogic.go` | PlanExecItemModel.FindOne, PlanExecItemModel.FindOneByExecId, PlanModel.FindOne, PlanBatchModel.FindOne, PlanModel.Trans, PlanExecItemModel.UpdateStatusToCompleted, UpdateStatusToFail, UpdateStatusToDelayed, UpdateStatusToOngoing, UpdateStatusToTerminated, PlanExecLogModel.Insert, PlanBatchModel.UpdateBatchFinishedTime, PlanModel.UpdateBatchFinishedTime |
| `app/trigger/internal/logic/getplanexecloglogic.go` | PlanExecLogModel.FindOne |
| `app/trigger/internal/logic/listplanexeclogslogic.go` | PlanExecLogModel.SelectBuilder, PlanExecLogModel.FindPageListByPageWithTotal |
| `app/trigger/internal/logic/listplanslogic.go` | PlanModel.SelectBuilder, PlanModel.FindPageListByPageWithTotal, PlanBatchModel.CalculatePlanProgress |
| `app/trigger/internal/logic/listplanexecitemslogic.go` | PlanExecItemModel.SelectBuilder, PlanModel.FindOne, PlanExecItemModel.FindPageListByPageWithTotal |
| `app/trigger/internal/logic/listplanbatcheslogic.go` | PlanExecItemModel.GetBatchStatusCounts, PlanExecItemModel.GetBatchTotalExecItems |
| `app/trigger/internal/logic/pauseplanlogic.go` | PlanModel.FindOne, PlanModel.FindOneByPlanId, PlanModel.Trans, PlanModel.UpdateWithVersion, PlanBatchModel.UpdateBuilder, PlanBatchModel.UpdateWithBuilder |
| `app/trigger/internal/logic/terminateplanlogic.go` | PlanModel.FindOne, PlanModel.FindOneByPlanId, PlanModel.Trans, PlanModel.UpdateWithVersion, PlanBatchModel.UpdateBuilder, PlanBatchModel.UpdateWithBuilder |
| `app/trigger/internal/logic/pauseplanbatchlogic.go` | PlanBatchModel.FindOne, PlanBatchModel.FindOneByBatchId, PlanModel.FindOneByPlanId, PlanBatchModel.Trans, PlanBatchModel.UpdateWithVersion |
| `app/trigger/internal/logic/resumeplanbatchlogic.go` | PlanBatchModel.FindOne, PlanBatchModel.FindOneByBatchId, PlanModel.FindOneByPlanId, PlanBatchModel.Trans, PlanBatchModel.UpdateWithVersion, PlanModel.UpdateBuilder, PlanModel.UpdateWithBuilder |
| `app/trigger/internal/logic/terminateplanbatchlogic.go` | PlanBatchModel.FindOne, PlanBatchModel.FindOneByBatchId, PlanModel.FindOneByPlanId, PlanBatchModel.Trans, PlanBatchModel.UpdateWithVersion, PlanModel.UpdateBatchFinishedTime |
| `app/trigger/internal/logic/pauseplanexecitemlogic.go` | PlanExecItemModel.FindOne, PlanExecItemModel.FindOneByExecId, PlanBatchModel.FindOne, PlanModel.FindOneByPlanId, PlanModel.Trans, PlanExecItemModel.UpdateWithVersion |
| `app/trigger/internal/logic/resumeplanexecitemlogic.go` | PlanExecItemModel.FindOne, PlanExecItemModel.FindOneByExecId, PlanModel.FindOne, PlanModel.Trans, PlanExecItemModel.UpdateWithVersion |
| `app/trigger/internal/logic/terminateplanexecitemlogic.go` | PlanExecItemModel.FindOne, PlanExecItemModel.FindOneByExecId, PlanBatchModel.FindOne, PlanModel.FindOneByPlanId, PlanModel.Trans, PlanExecItemModel.UpdateWithVersion, PlanBatchModel.UpdateBatchFinishedTime, PlanModel.UpdateBatchFinishedTime |
| `app/trigger/internal/logic/runplanexecitemlogic.go` | PlanExecItemModel.FindOne, PlanExecItemModel.FindOneByExecId, PlanBatchModel.FindOne, PlanModel.FindOneByPlanId, PlanExecItemModel.UpdateWithVersion |

### Category 4: Files With sqlx Import That Need Error Pattern Change (10 files)

These already use gormmodel repositories, but change `sqlx.ErrNotFound` → `gorm.ErrRecordNotFound` (or `errors.Is`):

`getplanlogic.go`, `resumeplanlogic.go`, `getplanbatchlogic.go`, `getplanexecitemlogic.go`, `createplantasklogic.go`, `callbackplanexecitemlogic.go`, `getplanexecloglogic.go`, `resumeplanbatchlogic.go`, `runplanexecitemlogic.go`, `resumeplanexecitemlogic.go`

### Category 5: Logic Files NOT Using Repositories (no changes needed)

Approximately 25 files in `app/trigger/internal/logic/` do **not** use gormmodel repositories and would be unaffected:
- `sendprototriggerlogic.go`, `sendtriggerlogic.go`, `calcplantaskdatelogic.go`, `runtasklogic.go`, `invokelogic.go`, `queueslogic.go`, `getqueueinfologic.go`, `gettaskinfologic.go`, `listscheduledtaskslogic.go`, `listcompletedtaskslogic.go`, `listpendingtaskslogic.go`, `listactivetaskslogic.go`, `listarchivedtaskslogic.go`, `listaggregatingtaskslogic.go`, `listretrytaskslogic.go`, `deleteallcompletedtaskslogic.go`, `deleteallarchivedtaskslogic.go`, `deletetasklogic.go`, `archivetasklogic.go`, `historicalstatslogic.go`, `nextidlogic.go`, `batchnextidlogic.go`, `common.go`, `getexecitemdashboardlogic.go`, etc.

NOTE: `getexecitemdashboardlogic.go` and `listplanbatcheslogic.go` use `l.svcCtx.Database` (goqu) and do NOT use repositories — they would NOT need changes for a repository removal, but `listplanbatcheslogic.go` also calls `PlanExecItemModel.GetBatchStatusCounts` and `GetBatchTotalExecItems`.

### Summary of Impact

| Category | File Count |
|----------|-----------|
| ServiceContext | 1 |
| Gormmodel package (remove/simplify) | 3 of 4 |
| Logic files (rewrite to raw GORM) | 19 |
| Logic files (error pattern change only) | 10 (subset of above) |
| Logic files (no change) | ~25 |
| **Total files needing modification** | **~23** |

## Caveats

- The `getexecitemdashboardlogic.go` and `listplanbatcheslogic.go` use raw goqu queries via `l.svcCtx.Database` in addition to repository calls — these raw goqu paths are independent of the repository layer.
- `createplantasklogic.go` line 46 has a **different** error pattern: `if err != sqlx.ErrNotFound` (negated) vs other files' `if err == sqlx.ErrNotFound`.
- The squirrel-based SelectBuilder/UpdateBuilder patterns in `listplanslogic.go`, `resumeplanlogic.go`, etc. would need to be converted to GORM chain methods or raw SQL.
