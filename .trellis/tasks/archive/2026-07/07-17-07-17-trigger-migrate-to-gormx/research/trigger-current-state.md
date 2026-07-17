# Research: Trigger Current State

- **Query**: Current trigger repository types, usage counts, remaining sqlx dependencies
- **Scope**: internal
- **Date**: 2026-07-17

## Findings

### ServiceContext DB Fields

**File**: `app/trigger/internal/svc/servicecontext.go`

| Line | Field | Type |
|------|-------|------|
| 45 | `DB` | `*gormx.DB` |
| 46 | `PlanModel` | `gormmodel.PlanRepository` |
| 47 | `PlanBatchModel` | `gormmodel.PlanBatchRepository` |
| 48 | `PlanExecItemModel` | `gormmodel.PlanExecItemRepository` |
| 49 | `PlanExecLogModel` | `gormmodel.PlanExecLogRepository` |
| 50 | `Database` | `*goqu.Database` |

Initialization at lines 63–65:
```go
dbConn := dbx.New(c.DB.DataSource)
gormDB := mustOpenGormDB(c.DB.DataSource, dbConn)
repos := gormmodel.NewRepositories(gormDB)
```

### Gormmodel Files (4 files)

**Directory**: `app/trigger/model/gormmodel/`

| File | Public Types/Interfaces |
|------|------------------------|
| `plan.go` | `Plan`, `PlanBatch`, `PlanExecItem`, `PlanExecLog` (gorm structs with table names) |
| `repository.go` | `PlanRepository` (12 methods), `PlanBatchRepository` (13 methods), `PlanExecItemRepository` (15 methods), `PlanExecLogRepository` (5 methods), `Repositories` struct, `repoBase`, `Tx = *gorm.DB`, helper functions (`findOne`, `createModel`, `updateModel`, `updateModelWithVersion`, `pageRows`, etc.) |
| `plan_repositories.go` | `planRepo` (PlanRepository impl), `planBatchRepo` (PlanBatchRepository impl), column name constants, to/from converter functions |
| `exec_repositories.go` | `planExecItemRepo` (PlanExecItemRepository impl), `planExecLogRepo` (PlanExecLogRepository impl), to/from converter functions |

### Repository Usage Across Logic Files

Total logic files: **44** (in `app/trigger/internal/logic/`)

Files using repositories (19 unique files):

**PlanModel** (15 files): `getplanlogic`, `pauseplanexecitemlogic`, `resumeplanlogic`, `pauseplanlogic`, `terminateplanlogic`, `runplanexecitemlogic`, `resumeplanexecitemlogic`, `listplanexecitemslogic`, `resumeplanbatchlogic`, `terminateplanexecitemlogic`, `callbackplanexecitemlogic`, `createplantasklogic`, `pauseplanbatchlogic`, `terminateplanbatchlogic`, `listplanslogic`

**PlanBatchModel** (16 files): Above + `getplanbatchlogic`, `listplanbatcheslogic`

**PlanExecItemModel** (10 files): `getplanbatchlogic`, `pauseplanexecitemlogic`, `runplanexecitemlogic`, `resumeplanexecitemlogic`, `listplanexecitemslogic`, `terminateplanexecitemlogic`, `callbackplanexecitemlogic`, `createplantasklogic`, `getplanexecitemlogic`, `listplanbatcheslogic`

**PlanExecLogModel** (3 files): `listplanexeclogslogic`, `getplanexecloglogic`, `callbackplanexecitemlogic`

### Remaining sqlx Import Locations (10 logic files)

All 10 files import `sqlx` **solely** for the `sqlx.ErrNotFound` sentinel error check:
- `model.ErrNotFound = sqlx.ErrNotFound` (defined at `model/vars.go:19`)
- These files already use gormmodel repositories — the sqlx import is a legacy artifact

| File | Line | sqlx Usage |
|------|------|-----------|
| `getplanlogic.go` | 16, 50 | `import sqlx`, `if err == sqlx.ErrNotFound` |
| `resumeplanlogic.go` | 17, 56 | `import sqlx`, `if err == sqlx.ErrNotFound` |
| `getplanbatchlogic.go` | 16, 51 | `import sqlx`, `if err == sqlx.ErrNotFound` |
| `getplanexecitemlogic.go` | 15, 49 | `import sqlx`, `if err == sqlx.ErrNotFound` |
| `createplantasklogic.go` | 20, 46 | `import sqlx`, `if err != sqlx.ErrNotFound` |
| `callbackplanexecitemlogic.go` | 23, 61 | `import sqlx`, `if err == sqlx.ErrNotFound` |
| `getplanexecloglogic.go` | 13, 41 | `import sqlx`, `if err == sqlx.ErrNotFound` |
| `resumeplanbatchlogic.go` | 15, 54 | `import sqlx`, `if err == sqlx.ErrNotFound` |
| `runplanexecitemlogic.go` | 17, 52 | `import sqlx`, `if err == sqlx.ErrNotFound` |
| `resumeplanexecitemlogic.go` | 17, 55 | `import sqlx`, `if err == sqlx.ErrNotFound` |

### goqu Database Usage (2 files)

Files using `l.svcCtx.Database` for raw goqu queries (not via repositories):
- `getexecitemdashboardlogic.go` — complex JOIN + aggregation query
- `listplanbatcheslogic.go` — paginated list with JOIN and count

### Transaction Pattern

Many logic files use the repository's `Trans` method with `*gorm.DB` tx:
```go
err = l.svcCtx.PlanModel.Trans(l.ctx, func(ctx context.Context, tx *gorm.DB) error {
    _, transErr = l.svcCtx.PlanModel.Insert(ctx, tx, insertPlan)
    _, err := l.svcCtx.PlanBatchModel.Insert(ctx, tx, &batch)
    _, err = l.svcCtx.PlanExecItemModel.Insert(ctx, tx, &planItem)
    return nil
})
```

Files with transactions: `createplantasklogic`, `pauseplanlogic`, `resumeplanlogic`, `terminateplanlogic`, `pauseplanbatchlogic`, `resumeplanbatchlogic`, `terminateplanbatchlogic`, `pauseplanexecitemlogic`, `resumeplanexecitemlogic`, `terminateplanexecitemlogic`, `callbackplanexecitemlogic`

## Caveats

- The `findOne` helper in `repository.go:156-165` converts `gorm.ErrRecordNotFound` → `model.ErrNotFound` (= `sqlx.ErrNotFound`). Replacing this with raw GORM would change the error to `gorm.ErrRecordNotFound`. Logic files must switch from `err == sqlx.ErrNotFound` to `err == gorm.ErrRecordNotFound` (or `errors.Is(err, gorm.ErrRecordNotFound)`).
- The `Trans` helper in `repoBase.Trans` wraps `r.db.WithContext(ctx).DB.Transaction(...)`. Replacing this with direct `l.svcCtx.DB.WithContext(ctx).DB.Transaction(...)` is straightforward but would need to be repeated in every file that uses transactions.
- The `Tx = *gorm.DB` type alias in `repository.go:20` is used in all repository method signatures — if removing repositories, tx type becomes raw `*gorm.DB`.
- Some logic files import `database/sql` for `sql.NullString`/`sql.NullTime` in model construction — these would remain.
