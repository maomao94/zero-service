# gormx 包规范

> gormx 是项目内 GORM 封装包（`common/gormx/`），负责数据库打开、审计回调、租户查询、软删恢复、批量操作、分页、日志和追踪配置。
> 通用数据库原则和 GORM 场景（ErrRecordNotFound、GaussDB NULL 处理等）见 [database-guidelines.md](./database-guidelines.md)。

## 文件组织

| 文件 | 职责 |
| --- | --- |
| `config.go` | `Config` 结构体，`parseLogLevel` |
| `db.go` | `DB` 包装类型，事务/租户/迁移方法 |
| `options.go` | `Option` 和 `dbOptions` |
| `open.go` | `Open`、`OpenWithConf`、`OpenWithDialector`、`OpenWithRawDB` |
| `batch.go` | 批量 CRUD（`BatchInsert`、`BatchUpdateByIds`、`BatchDeleteByIds`） |
| `batch_tenant.go` | 租户感知批量 CRUD（`BatchInsertWithTenant` 等） |
| `delete.go` | `SoftDelete`、`UnscopedDelete`、`UnscopedDeleteWithTenant` |
| `restore.go` | `Restore`、`RestoreWithTenant`、`hasLegacyDeleteFields` |
| `hook_helpers.go` | `SkipHooksUpdate`、`SkipHooksCreate` |
| `tenant_query.go` | `withTenantQueryFromDB`（未导出） |
| `tenant_scope.go` | `TenantScope` 等 scope 函数 |
| `pagination.go` | `QueryPage`、`QueryPageData`、`CursorPage`、`NormalizePage` |

## 调用约定

### DB 包装类型

`gormx.DB` 是 `*gorm.DB` 的包装，提供事务、租户和迁移辅助方法。

```go
func (db *DB) Transact(fn func(tx *DB) error) error
func (db *DB) WithTenant(ctx context.Context) *gorm.DB
func (db *DB) AutoMigrate(dst ...any) error
```

### 工具函数不再接受 ctx 参数

工具函数（`Upsert`、`BatchInsertWithTenant` 等）不接受 `ctx context.Context` 参数。ctx 通过调用方在传 db 前调用 `.WithContext(ctx)` 传递，函数内部从 `db.Statement.Context` 获取。`gorm.DB.Statement.Context` 是 GORM 存放上下文的规范位置。

```go
ctx := gormx.WithUserAndTenantContext(context.Background(), uid, "alice", "tenant-a")

// Before: gormx.Upsert(ctx, db, data, columns, updateColumns)
// After:  gormx.Upsert(db.WithContext(ctx), data, columns, updateColumns)
```

**10 个改动函数及新旧签名**（源文件：`batch_tenant.go`、`delete.go`、`restore.go`）：

| 函数 | 旧签名 | 新签名 |
| --- | --- | --- |
| `Upsert` | `(ctx, db, data, columns, updateColumns)` | `(db, data, columns, updateColumns)` |
| `BatchInsertWithTenant` | `(ctx, db, values)` | `(db, values)` |
| `BatchUpdateByIdsWithTenant` | `(ctx, db, model, updates)` | `(db, model, updates)` |
| `BatchDeleteByIdsWithTenant` | `(ctx, db, model, ids)` | `(db, model, ids)` |
| `BatchDeleteByConditionWithTenant` | `(ctx, db, model, queryFn)` | `(db, model, queryFn)` |
| `UnscopedDeleteWithTenant` | `(ctx, db, model, conds...)` | `(db, model, conds...)` |
| `RestoreWithTenant` | `(ctx, db, model, conds...)` | `(db, model, conds...)` |

内部 ctx 提取使用 `withTenantQueryFromDB(db)` 代替已删除的 `withTenantQuery(ctx, db)`。`withTenantQueryFromDB` 从 `db.Statement.Context` 提取租户 ID。

**以下功能保持 `ctx` 参数不变**：上下文生产/消费函数 (`WithUserContext`、`GetTenantID`、`WithFullSQL`)、`*DB` 链式方法 (`WithContext`、`WithTenant`)、Scope 工厂、Logger 接口方法。

### 分页查询

统一使用 `gormx.QueryPage` 或 `gormx.QueryPageData`，不手动写 Count + Find。

```go
func QueryPage[T any](db *gorm.DB, page, pageSize int, dest *[]T) (*PageResult[T], error)
func QueryPageData[T any](db *gorm.DB, page, pageSize int) ([]T, error)
```

- `QueryPage`：完整分页，返回 total / page / pageSize / totalPages
- `QueryPageData`：仅取数据，不计算 total，返回 `[]T`
- Good: `gormx.QueryPage(db.Order("id DESC"), page, pageSize, &devices)`
- Bad: 手动 `db.Count().Offset().Limit().Find()`，不处理 count=0

**JOIN 分页**（源文件 `common/gormx/pagination.go`、`app/djicloud/internal/logic/listhmsalertslogic.go`）：

```go
db := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.DjiHmsAlert{}).
    Table("dji_hms_alert dha").
    Joins("LEFT JOIN dji_device dd ON dha.gateway_sn = dd.gateway_sn")
queryDB := db.WithContext(l.ctx).Select("dha.*, dd.is_online").Order("dha.reported_at DESC,dha.id DESC")
pageResult, err := gormx.QueryPage(queryDB, int(in.GetPage()), int(in.GetPageSize()), &alerts)
```

用 `db.WithContext(ctx)` 克隆新 Statement（clone=1）隔离 ORDER BY/SELECT，避免污染原始 db 的 `Statement.Clauses`。LEFT JOIN 在唯一关联键上不产生重复行，count 无需 GROUP BY。

## 配置与打开

### `OpenWithConf` 信任 go-zero tag 默认值

`OpenWithConf` 不做程序化默认值兜底。所有默认值由 go-zero config tag（`default=100`、`default=true` 等）在配置加载阶段填入。

```go
// Wrong — 零值 Config（MaxIdleConns=0, SkipDefaultTransaction=false）
db, _ := gormx.OpenWithConf(gormx.Config{DataSource: dsn})

// Correct — 通过 go-zero 加载配置，tag 默认值已生效
var c config.Config
db := gormx.MustOpenWithConf(c.DB)

// Correct — 编程式构造使用 Open + Option
db, _ := gormx.Open(dsn, gormx.WithMaxIdleConns(50), gormx.WithMaxOpenConns(50))
```

`Config.LogLevel` tag 包含 `options=[silent,error,warn,info]`，go-zero 加载配置时校验，不在列表内的值被拒绝。

### Open* 入口函数必须对指针入参做 nil 校验

```go
func OpenWithDialector(dialector *gorm.Dialector, opts ...Option) (*DB, error) {
    if dialector == nil { return nil, errors.New("dialector is required") }
}
```

### 生产配置默认值

| 配置项 | 默认值 | 原因 |
|--------|--------|------|
| `MaxIdleConns` | 100 | = MaxOpenConns，避免连接抖动（5-20ms/次） |
| `ConnMaxIdleTime` | 5min | 低流量时自动清理闲置连接 |
| `ParameterizedQueries` | true | 安全：日志不暴露敏感参数（手机号、密码等） |
| `SkipDefaultTransaction` | true | 性能：单条写操作省掉 BEGIN/COMMIT |
| `PrepareStmt` | false | 保守：连接池切换/DB 重启后缓存失效 |

源文件：`config.go`。

### OpenTelemetry 注册失败时资源泄露

`Open()` 在 `registerOpenTelemetry` 失败后必须关闭已打开的底层 `sql.DB`：

```go
RegisterCallbacks(gormDB)
if err := registerOpenTelemetry(gormDB, options.openTelemetry); err != nil {
    closeOpenedDB(gormDB, options.rawDB)
    return nil, err
}
```

源文件：`open.go`。

### API 兼容性: `tracing.WithDBName` → `tracing.WithDBSystem`

`gorm.io/plugin/opentelemetry v0.1.16` 移除了 `tracing.WithDBName`，升级时需改为 `tracing.WithDBSystem`（设置 `db.system` 而非数据库名称）。

## 模型与审计

### `TimeMixin` 必须包含自动时间标签

```go
type TimeMixin struct {
    CreatedAt time.Time `gorm:"type:timestamp(6);autoCreateTime:milli" json:"created_at"`
    UpdatedAt time.Time `gorm:"type:timestamp(6);autoUpdateTime:milli" json:"updated_at"`
}
```

新表 `BaseModel` 已内嵌 `TimeMixin` 且标签正确，旧表 `LegacyBaseModel` 使用 `create_time/update_time` 非 GORM 自动标签，由 `RegisterCallbacks` 在 `BeforeCreate`/`BeforeUpdate` 中手动写入。

### GORM Model Hook 与 gormx Callback 的区别

| 机制 | 定义方式 | `SkipHooks: true` 影响 |
|------|----------|------------------------|
| **Model Hook** | 模型实现 `BeforeUpdate(tx *gorm.DB) error` | 跳过 |
| **Callback** | `RegisterCallbacks` 注册的全局回调 | 不跳过 |

gormx 的审计 callback 是基础设施级逻辑，不受 `SkipHooks` 控制：

```go
// SkipHooksUpdate 跳过 Model Hook，不跳过 gormx 审计 callback
gormx.SkipHooksUpdate(db.WithContext(ctx), &model, map[string]any{"name": "new"})
```

源文件：`callbacks.go`、`hook_helpers.go`。

## 陷阱与注意事项

### Don't: 在 GORM Scope 函数中使用 `HasTenantField`

`HasTenantField(db)` 依赖 `db.Statement.Schema`，但 GORM scope 函数在 Schema 解析**之前**执行。在 scope 内调用始终返回 `false`。

```go
// Wrong — HasTenantField always returns false in scope functions
func TenantScope(ctx context.Context) func(db *gorm.DB) *gorm.DB {
    return func(db *gorm.DB) *gorm.DB {
        if !HasTenantField(db) { return db }  // BUG
        return db.Where("tenant_id = ?", tenantID)
    }
}
```

源文件：`tenant_scope.go`。

### Don't: 依赖 Hooks 为 `SoftDeleteMixin` 模型设置删除审计字段

`SoftDeleteMixin` 使用 `gorm.DeletedAt`，GORM 内置软删除走 `db.Session(NewDB: true)` → 新 Session 丢弃 `beforeDeleteHook` 的 `SetColumn` 值。删除审计字段需要在 service 层显式设置。`LegacySoftDeleteMixin`（`del_state`/`delete_time`）不受影响，因 `soft_delete` 插件机制不同。

源文件：`callbacks.go`、`model_audit.go`。

### Gotcha: 软删后再写回需先 Restore

旧表模型（`LegacyBaseModel`，含 `delete_time/del_state`）伪删除后，`FirstOrCreate` 查不到该记录，可能撞唯一索引。先调用 `gormx.Restore` 恢复：

```go
gormx.Restore(tx.DB, &DjiDeviceTopo{}, "gateway_sn = ? AND sub_device_sn = ?", gatewaySn, sub.SN)
tx.WithContext(ctx).Where(where).Assign(updateData).FirstOrCreate(&topoRecord)
```

`Restore` 内部根据模型判断是旧表（清空 `delete_time/del_state`）还是新表（清空 `deleted_at`），走正常 update callbacks。

源文件：`restore.go`、`app/djicloud/internal/hooks/sys_status_up.go`。

### Gotcha: 批量 Create 中 Hook 仅对首条记录生效

GORM `Create([]T)` 的 `beforeCreateHook` 仅对第一个元素设置审计字段：

```sql
INSERT INTO model (tenant_id, create_user, ...) VALUES
  ("tenant-1", "uid-1", ...),  -- ✅ 第1条正确
  ("",        "",      ...)     -- ❌ 第2条为空
```

需要完整审计时逐条 create 或使用 `BatchInsertWithTenant`。

### Gotcha: 测试 DB 必须关闭连接

`openTestDB` 返回的 `sql.DB` 不会随进程退出自动清理（SQLite 内存库除外）。必须调用 `t.Cleanup(func() { _ = sqlDB.Close() })`。

源文件：`test_helpers_test.go`。

## 测试

`test_helpers_test.go` 中 `openTestDB` 默认启用 `logger.Info`，每次 `go test` 输出完整 SQL。AI 审查代码时必须分析 SQL 日志验证：

- CREATE: 审计字段 + 租户 + 版本写入正确
- UPDATE: 仅有 `update_user`/`update_name`/`version+1`，无 `delete_*` 侧漏
- DELETE: 硬删除为 `DELETE FROM`（无 SET），软删除 SQL 符合预期
- 租户过滤: 跨租户操作返回 `rows:0`

改动公共行为后至少运行 `go test ./common/gormx/...`。单测使用 SQLite 内存库，通过 `openTestDB` 自动注册回调和清理连接。
