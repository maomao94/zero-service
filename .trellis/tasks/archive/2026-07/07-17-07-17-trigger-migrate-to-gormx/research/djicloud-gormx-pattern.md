# Research: djicloud gormx.DB Pattern

- **Query**: Understand how djicloud ServiceContext exposes `*gormx.DB` and how logic files use raw GORM
- **Scope**: internal
- **Date**: 2026-07-17

## Findings

### ServiceContext Wiring

**File**: `app/djicloud/internal/svc/servicecontext.go`

- Line 31: `DB *gormx.DB` field — directly exposed, no repository wrapper
- Lines 36–58: `initDB(c config.Config) *gormx.DB` — opens gormx, auto-migrates gorm models in dev mode
- Line 63: `db := initDB(c)` in `NewServiceContext`
- Line 163: `DB: db` assigned to `ServiceContext`

Key difference from trigger: djicloud does **not** create gormmodel repositories. It passes `*gormx.DB` directly to the service context. Logic files consume `svcCtx.DB` directly via GORM methods.

### Logic Usage Pattern

Logic files use `l.svcCtx.DB.WithContext(l.ctx)` as the entry point, then chain GORM methods:

**Find one (First)**:
```
app/djicloud/internal/logic/getdevicedetaillogic.go:31-34
  l.svcCtx.DB.WithContext(l.ctx).
      Where("device_sn = ?", in.DeviceSn).
      First(&device).Error
```

**Find many (Find)**:
```
app/djicloud/internal/logic/deletecustomflyregionlogic.go:34
  l.svcCtx.DB.WithContext(l.ctx).Where("gateway_sn = ?", gatewaySn).Find(&regions).Error
```

**Create**:
```
app/djicloud/internal/logic/submitcustomflyregionlogic.go:99
  l.svcCtx.DB.WithContext(l.ctx).Create(region).Error
```

**Updates**:
```
app/djicloud/internal/logic/ackhmsalertlogic.go:31-36
  l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.DjiHmsAlert{}).Where("id = ?", in.Id).Updates(map[string]any{...})
```

**Delete**:
```
app/djicloud/internal/logic/deletecustomflyregionlogic.go:41
  l.svcCtx.DB.WithContext(l.ctx).Where("gateway_sn = ?", gatewaySn).Delete(&gormmodel.DjiFlyRegion{}).Error
```

**Model-based query with list + pagination**:
```
app/djicloud/internal/logic/listflighttaskprogresslogic.go:33
  db := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.DjiDockFlightTask{})
```

### Error Handling

djicloud does **NOT** import `sqlx` or use `sqlx.ErrNotFound`. It checks `err != nil` directly from GORM:

- `getdevicedetaillogic.go:33`: `if err != nil { return nil, tool.NewErrorByPbCodeWrap(...) }`
- `getdevicestatesnapshotlogic.go:32`: Same pattern
- `ackhmsalertlogic.go:37-38`: `if result.Error != nil { return ... }` — also checks `RowsAffected == 0`

### Key Difference Summary

| Aspect | djicloud | trigger (current) |
|--------|----------|-------------------|
| DB field type | `*gormx.DB` | `*gormx.DB` |
| Repository layer | **None** | 4 repository interfaces |
| Logic accesses | `svcCtx.DB.WithContext(ctx).Model().Where().First()` | `svcCtx.PlanModel.FindOne(ctx, id)` |
| Not-found error | GORM `err != nil` | `sqlx.ErrNotFound` (via `model.ErrNotFound`) |
| Imports in logic | Import gormmodel structs for model types | Import `sqlx` for ErrNotFound, import `gorm.io/gorm` for `*gorm.DB` tx param |

## Caveats

- djicloud has no transaction use in logic files — only single-table operations via GORM
- djicloud does NOT use squirrel/query builders — only GORM chain methods
- For trigger's migration, the transaction pattern (`l.svcCtx.PlanModel.Trans(ctx, func(ctx, tx *gorm.DB) error {...})`) is a key concern — trigger logic files currently pass `*gorm.DB` (tx) into repository methods
