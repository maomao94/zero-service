# Design

## Boundary

This task changes the `common/gormx` Legacy GORM path only. Generated go-zero DAOs and trigger SQL remain unchanged until the later trigger GORM migration.

## Legacy Soft Delete Contract

`LegacySoftDeleteMixin` should expose:

- `DeleteTime sql.NullTime` mapped to `delete_time` as audit time.
- `IsDeleted soft_delete.DeletedAt` mapped to `is_deleted` as the delete-state flag.

The legacy compatibility state contract is:

- `is_deleted = 0`: not deleted.
- `is_deleted = 1`: deleted.
- `delete_time`: delete audit time only.

The soft-delete plugin remains responsible for converting delete to update, applying default query filtering, setting `is_deleted`, and filling `delete_time` through `DeletedAtField`.

## Mixin Capabilities

Mixins should provide capability methods without owning GORM lifecycle hooks:

- `LegacyStringIDMixin.BeforeCreateID() error`: generate UUID when `Id` is empty.
- `AuditMixin` / `StringAuditMixin`: create/update/delete audit field setters from `UserContext` for atomic mixin users.
- `AuditWithoutDeleteMixin` / `StringAuditWithoutDeleteMixin`: create/update audit setters for atomic mixin users.
- `TenantMixin`: create tenant setter from `UserContext` for atomic mixin users.

The existing `Id` field spelling stays unchanged to match current project conventions and existing model code.

## Legacy BaseModel Lifecycle

`LegacyBaseModel` and `LegacyStringBaseModel` own GORM hooks for Legacy models:

- `BeforeCreate`: ID generation if needed, audit create fields, tenant create field.
- `BeforeUpdate`: update audit fields.
- `BeforeDelete`: delete audit fields.

The BaseModel hooks should orchestrate common fields through the shared gormx field-setting helpers. This keeps audit and tenant fields optional for Legacy models that do not physically contain every old-system common column, and avoids adding duplicate `tenant_id` fields to models that already define custom tenant columns.

## Callback Registration Boundary

`RegisterCallbacks()` remains registered from `Open` and test helpers.

Callbacks are currently no-op extension placeholders. They do not inject audit, tenant, or delete fields for any model.

Common field lifecycle is model-owned. `LegacyBaseModel` and `LegacyStringBaseModel` own the Legacy path through GORM model hooks. Non-Legacy models that need automatic audit/tenant behavior should implement their own hooks and call the mixin capability methods.

Delete audit remains business-owned. The `soft_delete` plugin owns the actual `is_deleted/delete_time` update; business-specific delete audit requirements should be handled explicitly in business code rather than making gormx callbacks complex.

`setSchemaColumn()` stays as a helper for model hooks that fill optional fields. `HasTenantField()` remains for existing tenant-query helpers, but callbacks no longer use it.

## Restore Contract

`Restore` remains conservative:

- For models with `delete_time`, set it to NULL.
- For models with `is_deleted`, set zero value.
- Transitional `del_state` support may remain to avoid breaking existing helper tests and old known model shapes.
- Unknown complex recovery should be performed by business code using `Unscoped()` and explicit `Updates(...)`.

`RestoreWithTenant` keeps tenant filtering behavior through existing `withTenantQueryFromDB`.

## Compatibility Notes

- Non-Legacy callback-based field injection is intentionally disabled; tests should not expect automatic audit/tenant fill unless the model has its own hook.
- `SkipHooksCreate` and `SkipHooksUpdate` skip model hooks and callbacks are no-op, so they do not fill audit/tenant fields implicitly.
- Legacy models using `SkipHooks: true` will skip BaseModel hooks by GORM design.
- Existing comments or README text that mention Legacy `del_state` must be updated for the gormx Legacy path.

## Rollback Shape

The change is scoped to gormx model/callback code and tests. Rollback is reverting `common/gormx` code and documentation changes. No database migration or generated DAO rewrite is included in this task.
