# gormx 快速使用指南

`common/gormx` 是项目内 GORM 封装包，负责数据库打开、回调注册、租户查询、软删恢复、批量操作、分页、日志和追踪配置。

## 快速开始

```go
db, err := gormx.OpenWithConf(gormx.Config{
	DataSource: "user:pass@tcp(127.0.0.1:3306)/app?charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai",
})
if err != nil {
	return err
}
```

`OpenWithConf` 依赖 go-zero 配置加载时的 tag 默认值（`default=100`、`default=true` 等），因此不要在代码里直接构造零值 `Config{}`。需要编程式构建连接选项时使用 `Open(dsn, With...)`。

## 模型选择

gormx 提供原子级 mixin，按需组合；Legacy 表保留 `LegacyBaseModel` / `LegacyStringBaseModel` 作为旧系统兼容字段组合。

- `IDModel` / `StringIDModel` — uint / string 主键。
- `TimeMixin` — `created_at` / `updated_at`（自动填充）。
- `SoftDeleteMixin` — 标准 `deleted_at` 软删除。
- `VersionMixin` — `optimisticlock.Version` 乐观锁，**默认不要加**。每次 UPDATE 会附加 `WHERE version = ?` 并自增 version，影响写入性能。仅用于高并发编辑场景（如多人同时修改同一配置），IoT 数据流、日志表、事件表不要加。
- `TenantMixin` — `tenant_id` 租户隔离字段。
- `AuditMixin` / `StringAuditMixin` — 创建/更新/删除审计（uint / string 类型用户 ID）。
- `AuditWithoutDeleteMixin` / `StringAuditWithoutDeleteMixin` — 仅创建/更新审计，无删除审计。
- `LegacyIDMixin` / `LegacyStringIDMixin` — 旧表 int64 / string 主键。
- `LegacyTimeMixin` — 旧表 `create_time` / `update_time`。
- `LegacySoftDeleteMixin` — 旧系统兼容表 `is_deleted` / `delete_time` 软删除；`is_deleted` 为状态字段，`delete_time` 仅作删除审计时间。
- `LegacyBaseModel` — 旧表默认组合（int64 id + legacy 时间 + legacy 软删）。
- `LegacyStringBaseModel` — 同上，string 主键。

```go
// 典型组合方式
type Device struct {
	gormx.IDModel
	gormx.TimeMixin
	gormx.SoftDeleteMixin
	gormx.AuditMixin
	Name string `gorm:"column:name"`
}

// 需要乐观锁时显式嵌入
type Config struct {
	gormx.LegacyBaseModel
	gormx.VersionMixin
	Key string `gorm:"column:key"`
}
```

## 用户与租户上下文

创建、更新和删除审计依赖 context 中的 `UserContext`。

```go
ctx := gormx.WithUserAndTenantContext(context.Background(), uint(7), "alice", "tenant-a")
err := db.WithContext(ctx).Create(&device).Error
```

通用字段生命周期由模型自己的 GORM hook 负责；全局 callbacks 目前只是注册/扩展占位，不会自动写入审计、租户或删除字段。Legacy 模型使用 `LegacyBaseModel` / `LegacyStringBaseModel` 的 hook 填充存在的 `create_user`、`create_name`、`update_user`、`update_name`、`tenant_id` 字段；非 Legacy 模型如需自动审计，应在模型上显式实现 hook 并调用对应 mixin capability 方法。

租户查询优先使用 `DB` 方法或 scope：

```go
err := db.WithTenant(ctx).Find(&devices).Error
err := db.Scopes(gormx.TenantScopeStrict(ctx)).Find(&devices).Error
```

`TenantScope` 没有租户上下文时不过滤；`TenantScopeStrict` 没有租户上下文时返回空结果。

## 常用操作

事务：

```go
err := db.Transact(func(tx *gormx.DB) error {
	return tx.WithContext(ctx).Create(&device).Error
})
```

批量：

```go
err := gormx.BatchInsert(db.DB, devices, 100)
err := gormx.BatchUpdateByIds(db.DB, &Device{}, []gormx.Ups{{"id": 1, "name": "new"}})
```

Upsert（WHERE 匹配 → Assign 强更新 → 未命中则 Create）：

```go
err := db.WithContext(ctx).Where(map[string]any{"device_sn": sn}).
	Assign(map[string]any{"name": name}).
	FirstOrCreate(&Device{DeviceSn: sn, Name: name}).Error
```

分页：

```go
var list []Device
page, err := gormx.QueryPage(db.WithTenant(ctx), 1, 20, &list)
```

## 软删与恢复

普通删除走 GORM 软删；Legacy 旧系统兼容表会设置 `is_deleted=1` 并填充 `delete_time`。

```go
err := gormx.SoftDelete(db.DB, &Device{}, "id = ?", id)
err := gormx.Restore(db.DB, &Device{}, "id = ?", id)
```

`Restore` 只处理 gormx 已知软删字段：标准 `deleted_at`、Legacy `is_deleted/delete_time`，以及过渡兼容的 `del_state`。字段形态复杂或恢复条件有业务含义时，业务侧应使用 `Unscoped()` 加显式 `Updates(...)` 自行恢复。

租户表使用带 tenant 的版本，防止跨租户恢复或硬删：

```go
err := gormx.RestoreWithTenant(db.DB.WithContext(ctx), &Device{}, "id = ?", id)
err := gormx.UnscopedDeleteWithTenant(db.DB.WithContext(ctx), &Device{}, "id = ?", id)
```

## 日志与追踪

默认日志参数脱敏，避免在 SQL 日志中打印手机号、Token 等敏感值。调试完整 SQL 时显式使用：

```go
ctx = gormx.WithFullSQL(ctx)
```

OpenTelemetry 默认启用，但不采集 metrics 和 query variables。需要关闭：

```go
db, err := gormx.Open(dsn, gormx.WithoutOpenTelemetry())
```

## 测试建议

- 包内单测使用 SQLite 内存库，并通过 `openTestDB` 自动注册回调和清理连接。
- 改动公共行为后至少运行 `go test ./common/gormx`。
- 新增调用方依赖软删恢复、租户隔离或 upsert 时，补调用方包的目标单测。

## 注意事项

- `BatchInsertWithTenant` 使用批量 create，GORM hook 对批量审计字段存在限制；需要逐条完整审计时不要使用批量插入。
- `Restore` 会执行模型 update hook；如果恢复后又更新业务字段，可能触发两次 update/save hook。
- `OpenWithRawDB` 不接管外部 `*sql.DB` 生命周期，调用方负责关闭。
