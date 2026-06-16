# gormx 快速使用指南

`common/gormx` 是项目内 GORM 封装包，负责数据库打开、审计回调、租户查询、软删恢复、批量操作、分页、日志和追踪配置。

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

- 新表默认用 `BaseModel`：uint 主键、审计字段、版本、标准 `deleted_at` 软删和时间字段。
- string 主键用 `StringBaseModel` 或 `StringIDModel`。
- 租户表用 `TenantModel`、`TenantStringIDModel` 或手动嵌入 `TenantMixin`。
- 旧表用 `LegacyBaseModel`：`id/create_time/update_time/delete_time/del_state/version`。

```go
type Device struct {
	gormx.TenantModel
	Name string `gorm:"column:name"`
}
```

## 用户与租户上下文

创建、更新和删除审计依赖 context 中的 `UserContext`。

```go
ctx := gormx.WithUserAndTenantContext(context.Background(), uint(7), "alice", "tenant-a")
err := db.WithContext(ctx).Create(&device).Error
```

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

Upsert：

```go
err := gormx.UpdateOrCreate(ctx, db, &Device{},
	map[string]any{"device_sn": sn},
	&Device{DeviceSn: sn, Name: name},
	map[string]any{"name": name},
)
```

分页：

```go
var list []Device
page, err := gormx.QueryPage(db.WithTenant(ctx), 1, 20, &list)
```

## 软删与恢复

普通删除走 GORM 软删；旧表会设置 `delete_time/del_state`。

```go
err := gormx.SoftDelete(db.DB, &Device{}, "id = ?", id)
err := gormx.Restore(db.DB, &Device{}, "id = ?", id)
```

租户表使用带 tenant 的版本，防止跨租户恢复或硬删：

```go
err := gormx.RestoreWithTenant(ctx, db.DB, &Device{}, "id = ?", id)
err := gormx.UnscopedDeleteWithTenant(ctx, db.DB, &Device{}, "id = ?", id)
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
- `Restore` 会执行 update callbacks；如果恢复后又更新业务字段，可能触发两次 update/save hook。
- `OpenWithRawDB` 不接管外部 `*sql.DB` 生命周期，调用方负责关闭。
