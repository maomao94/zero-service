# Migrate ieccaller device table to GORM

## Goal

Migrate `app/ieccaller` device point mapping persistence from the root generated `model.DevicePointMappingModel` / `sqlx` path to a GORM-based implementation modeled after `app/ispagent`.

The service should keep the existing `device_point_mapping` table, proto contract, cache behavior, query filters, and ASDU push enrichment semantics while using `common/gormx` and service-local GORM model/store code.

## Confirmed Facts

- `app/ieccaller/internal/svc/servicecontext.go` currently imports root `zero-service/model`, opens DB via `common/dbx`, disables SQL statement logging through `sqlx.DisableStmtLog`, and initializes `DevicePointMappingModel` only when `c.DB.DataSource` is non-empty.
- Current `DevicePointMappingModel` supports `FindOne`, `FindOneByTagStationCoaIoa`, paginated list with total, cache get/remove/generate-key, and cache-backed lookup with 24h TTL.
- `PushASDU` uses cache-backed lookup to enrich `types.MsgBody.Pm` and drops push when `enable_push != 1`.
- `ClearPointMappingCache` and MQTT broadcast cache clearing use only cache primitives and generated key shape `pm:{tagStation}:{coa}:{ioa}`.
- The existing table name is `device_point_mapping`; current fields include legacy `id`, `create_time`, `update_time`, `delete_time`, `is_deleted`, `create_user`, `update_user`, `dept_code`, key fields `tag_station/coa/ioa`, device fields, push flags, description, and `ext_1` through `ext_5` nullable strings. User explicitly requested not to carry `version` in the GORM model and to use `gormx.LegacyStringBaseModel` as the default migration base model.
- `app/ispagent` uses `DB gormx.Config`, `gormx.MustOpenWithConf(c.DB)`, service-local `model/gormmodel`, and dev/test `MustAutoMigrate`.
- Project specs require using `ServiceContext` for dependencies, `common/gormx` for GORM, `gormx.QueryPage` for pagination, and preserving config defaults through go-zero tags.

## Requirements

- Add service-local GORM model/store code under `app/ieccaller` for `device_point_mapping` instead of using root generated `model.DevicePointMappingModel` for this table. Reusing generic helpers from `model/vars.go` is allowed when explicitly requested.
- Update `ieccaller` config and `ServiceContext` to use `common/gormx.Config` / `gormx.MustOpenWithConf` in the same style as `ispagent`.
- Keep DB optional: when `DB.DataSource` is empty, device point mapping functionality remains uninitialized and existing nil-model behavior is preserved.
- Preserve table name, column names, soft-delete filtering, and externally visible proto response fields.
- Preserve point mapping cache behavior, including 24h TTL, key format, negative-cache behavior for missing rows, and manual/broadcast cache invalidation.
- Preserve current read behaviors for query-by-id, query-by-key, page-list filters (`tag_station`, `coa`, `device_id`) and `id desc` ordering.
- Do not change `.proto` contract or generated RPC files unless inspection reveals an unavoidable compile break.
- Avoid introducing shared/common abstractions unless the migration requires them; keep implementation service-local like `ispagent`.
- Do not keep a compatibility layer or fallback to the old root `model.DevicePointMappingModel`; migrate the device point mapping path directly to GORM.

## Acceptance Criteria

- [ ] `app/ieccaller` builds without using root `model.DevicePointMappingModel` / generated SQLx DAO for device point mapping.
- [ ] `ServiceContext` initializes a `*gormx.DB` and GORM-backed point mapping store when `DB.DataSource` is configured.
- [ ] Query-by-id and query-by-key exclude soft-deleted rows and map nullable DB fields to proto/string fields as before.
- [ ] Page list supports existing filters, returns total, honors page/pageSize normalization behavior provided by `gormx.QueryPage`, and orders by `id desc`.
- [ ] `PushASDU` still enriches point mapping metadata and suppresses push when `enable_push != 1`.
- [ ] Cache clearing through RPC and MQTT broadcast keeps the same key format and clears local cache entries.
- [ ] Related `ieccaller` tests/build commands pass.
- [ ] Old `sqlx`/`dbx` device point mapping initialization is removed from `ieccaller` rather than retained as a fallback.

## Out Of Scope

- Database data migration/backfill scripts are out of scope unless the existing schema cannot be represented by GORM tags.
- Changing RPC request/response semantics is out of scope.
- Reworking unrelated IEC104 client, MQTT, Kafka, or stream-event behavior is out of scope.

## Open Questions

- None blocking after repository inspection.
