# Design

## Architecture

`app/ieccaller` will follow the `app/ispagent` pattern:

- `internal/config.Config.DB` becomes `gormx.Config`.
- `internal/svc.ServiceContext` owns `DB *gormx.DB` and a service-local point mapping store.
- Service-local model definitions live under `app/ieccaller/model/gormmodel`.
- Service-local model, store/cache, and conversion methods live together under `app/ieccaller/model/gormmodel` per user direction, rather than under a separate `internal/pointmapping` package. The cache wrapper reuses `model.CacheEntry[T]` from `model/vars.go`.

## Data Model

Define a GORM struct for `device_point_mapping` with explicit `gorm:"column:..."` tags matching existing columns:

- Legacy base columns: `id`, `create_time`, `update_time`, `delete_time`, `is_deleted`.
- Additional metadata columns: `create_user`, `update_user`, `dept_code`; `version` is intentionally omitted per user direction.
- Business key columns: `tag_station`, `coa`, `ioa`.
- Device/push columns: `device_id`, `device_name`, `td_table_type`, `enable_push`, `enable_raw_insert`.
- Nullable text columns: `description`, `ext_1` through `ext_5` as `sql.NullString` to preserve old NULL handling.
- Required business columns are `tag_station`, `coa`, `ioa`, `device_id`, and `device_name`; `tag_station/coa/ioa` enforce point uniqueness and `device_id/device_name` are the mapped device identity. `enable_push` and `enable_raw_insert` are also `not null`; `td_table_type`, description, and ext fields keep defaults without `not null`.
- Numeric columns avoid raw `type:` tags for cross-database compatibility across PostgreSQL/GaussDB/MySQL/SQLite. `coa/ioa/enable_push/enable_raw_insert` use Go `int`; protobuf conversion casts `coa/ioa` back to `int64` at the boundary.

Use `gormx.LegacyStringBaseModel` for legacy id/time/soft-delete behavior per user direction, plus explicit fields for columns not included by that mixin. Keep `TableName() string` returning `device_point_mapping`.

## Store Contract

The GORM store should provide the current operations needed by `ieccaller`:

- `FindOne(ctx, id)`
- `FindOneByTagStationCoaIoa(ctx, tagStation, coa, ioa)`
- `FindPage(ctx, filter, page, pageSize)`
- `GetCache(ctx, key)`
- `GenerateCacheKey(tagStation, coa, ioa)`
- `FindCacheOneByTagStationCoaIoa(ctx, tagStation, coa, ioa)`
- `RemoveCache(ctx, keys...)`

Return `gorm.ErrRecordNotFound` from direct missing-row lookups unless callers map it; cache-backed lookup treats missing rows as a valid negative cache entry, matching the existing generated-model behavior.

## Data Flow

- Query Logic calls the GORM store directly and maps GORM model values to protobuf values through small service-local mapper code.
- `PushASDU` calls cache-backed store lookup, then maps the returned model to `types.PointMapping`.
- Cache clearing RPC and MQTT broadcast call cache primitives only; no DB write occurs.
- Page-list builds a GORM query from request filters and calls `gormx.QueryPage`.

## Compatibility

- The database table name and columns remain unchanged.
- DB remains optional when `DB.DataSource` is empty.
- Cache key format remains `pm:%s:%d:%d`.
- Soft-delete filtering remains equivalent to `is_deleted = 0` by relying on GORM soft-delete field and/or explicit query predicate where needed.
- Dev/test may use `MustAutoMigrate` for the GORM model, matching `ispagent`; production should not rely on auto migration.
- There is no compatibility mode: configured DB access for this table always uses the GORM store.

## Trade-Offs

- A service-local store is preferable to adapting the old generated model interface because the goal is to remove this table's dependency on root sqlx model code.
- Keeping mapper functions explicit is safer than broad copier usage because GORM soft-delete uses `soft_delete.DeletedAt`, while proto fields are simple strings/integers.
- No shared common package is added because the migration is scoped to one `ieccaller` table.

## Rollback

The rollback path is a code revert of this migration. The implementation should not carry runtime fallback logic for the old root `model.DevicePointMappingModel`.
