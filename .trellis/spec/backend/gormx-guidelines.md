# gormx 包规范

> gormx 是项目内 GORM 封装包（`common/gormx/`），负责数据库打开、回调注册、租户查询、软删恢复、批量操作、分页、日志和追踪配置。
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
| `tenant_scope.go` | `TenantScope` 等 scope 函数、`WithTenantContext` / `WithUserAndTenantContext` |
| `pagination.go` | `QueryPage`、`QueryPageData`、`CursorPage`、`NormalizePage` |
| `callbacks.go` | 全局 GORM 回调注册入口；当前回调函数是 no-op 扩展占位，不写审计/租户/删除字段 |
| `model.go` | 原子 mixin：`IDModel`/`StringIDModel`（uint/string 主键）、`TimeMixin`（created_at/updated_at）、`SoftDeleteMixin`（deleted_at）、`VersionMixin`（乐观锁，按需嵌入）、`TenantMixin`（tenant_id） |
| `model_audit.go` | 审计 mixin：`AuditMixin`（uint 用户）、`StringAuditMixin`（string 用户）、`AuditWithoutDeleteMixin`、`StringAuditWithoutDeleteMixin` |
| `model_legacy.go` | Legacy mixin + `LegacyBaseModel`/`LegacyStringBaseModel`（int64/string 主键 + create_time/update_time + 旧系统兼容 `is_deleted/delete_time`，**不含 VersionMixin**） |
| `driver.go` | 数据库驱动：DSN 前缀识别、Dialector 创建、类型推断 |
| `upsert.go` | `Upsert` / `UpsertInBatches` 批量合并写入 |
| `trace.go` | GORM 链路追踪回调 |
| `logger.go` | `gormLogger` 自定义日志器，支持 `WithoutSQLTrace` context 标记 |
| `user_context.go` | `AuditUserValue`/`AuditUserID`，租户上下文提取 |
| `util.go` | `setSchemaColumn`、`mapKeys`、`zeroValue` |

## 数据库驱动

### DSN 前缀识别

`ParseDatabaseType` 按 DSN scheme 前缀识别数据库类型，不使用端口号、关键字等启发式：

| 前缀 | DatabaseType |
|------|-------------|
| `mysql://` | `DatabaseMySQL` |
| `postgres://` / `postgresql://` | `DatabasePostgres` |
| `sqlite://` / `sqlite3://` / `file:` / `:memory:` | `DatabaseSQLite` |

空 DSN 默认 `DatabaseMySQL`。`gaussdb://` 暂时不支持，GaussDB PG 兼容模式必须使用 `postgres://` DSN 走 PostgreSQL driver，避免 `gorm.io/driver/gaussdb` 的 timestamp 扫描时区兼容问题。其他格式（如 `user:pass@tcp(...)`、`host=localhost sslmode=...`）不会被自动识别，需通过 `WithDialector` 或 `OpenWithRawDB` 显式指定类型。

```go
// Good: URL 前缀可自动识别
db, _ := gormx.Open("postgres://user:pass@host/db?sslmode=disable")
db, _ := gormx.Open("postgres://user:pass@gaussdb-host:8000/db?sslmode=disable&TimeZone=Asia/Shanghai")

// Good: 无法自动识别时显式指定
db, _ := gormx.OpenWithRawDB(sqlDB, gormx.DatabaseMySQL, opts...)
dialector := postgres.Open("postgres://user:pass@host/db?sslmode=disable")
db, _ := gormx.OpenWithDialector(&dialector)
```

### 新增驱动

新增数据库类型需要修改 3 处：
1. `DatabaseType` 常量 + `ParseDatabaseType` 前缀识别
2. `GetDialector` switch case
3. `GetDatabaseTypeFromDialector` type switch

源文件：`driver.go`。

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

## 模型

### Mixin 组合方式

新表模型由原子 mixin 按需组合；Legacy 表保留 `LegacyBaseModel` / `LegacyStringBaseModel` 作为旧系统兼容字段组合：

```go
// 典型——含审计+软删
type Device struct {
    gormx.IDModel
    gormx.TimeMixin
    gormx.SoftDeleteMixin
    gormx.AuditMixin
    Name string `gorm:"column:name"`
}

// 租户表——嵌入 TenantMixin
type TenantDevice struct {
    gormx.IDModel
    gormx.TenantMixin
    gormx.TimeMixin
    Name string `gorm:"column:name"`
}

// 机巢 IoT 数据——不要加 VersionMixin
type DjiDeviceOsdSnapshot struct {
    gormx.LegacyBaseModel
    DeviceSn string `gorm:"column:device_sn;uniqueIndex"`
    RawJSON  string `gorm:"column:raw_json;type:text"`
}
```

Legacy 表继续使用项目内既有字段名风格：Go 主键字段保留 `Id`，对应列名仍是 `id`；`TenantMixin` 使用 `TenantID` / `tenant_id`；`LegacySoftDeleteMixin` 的 Go 字段是 `IsDeleted`，列名是 `is_deleted`，`DeleteTime` 只保存删除审计时间。

### `VersionMixin` 性能影响

`VersionMixin` 使用 `optimisticlock.Version`，每次 UPDATE 自动附加 `WHERE version = ?` 并自增 version。高频写入场景下每个 UPDATE 多一次 version 比较+自增，影响吞吐。**仅用于高并发编辑场景**（多人同时改同一配置），IoT 数据流、日志表、事件表不要加。

```go
// Good: 配置表（多人并发编辑）→ 显式嵌入
type ModbusSlaveConfig struct {
    gormx.LegacyBaseModel
    gormx.VersionMixin
    ...
}

// Good: IoT 快照表（单写高频更新）→ 不加 version
type DjiDeviceOsdSnapshot struct {
    gormx.LegacyBaseModel  // 不含 VersionMixin
    ...
}
```

### ID 字段约定

- 旧表（`LegacyIDMixin` / `LegacyStringIDMixin`）：`Id` + `gorm:"column:id;primaryKey"`
- 新表（`IDModel` / `StringIDModel`）：`Id` + `gorm:"primarykey"`
- `TenantMixin`：`TenantID`（Go 复合缩写惯例）
- GORM 仅对全大写 `ID` 自动识别为主键，`Id` 需要显式 `primarykey` tag

### UUID 字段约定

- 入库 UUID 必须去杠：使用 `tool.SimpleUUID()`（生成 `550e8400e29b41d4a716446655440000`），不要用 `tool.UUID()`（生成 `550e8400-e29b-41d4-a716-446655440000`）。
- `LegacyStringBaseModel.BeforeCreate` 钩子通过 `BeforeCreateID()` 自动为空的 `Id` 生成去杠 UUID，业务侧不需要手动预生成主键。
- 业务唯一标识字段（如 `plan_id`、`batch_id`、`exec_id`）在手动生成时同样使用 `tool.SimpleUUID()`，保持存储格式一致。
- 对外接口透出的 UUID 可以根据协议需要保留杠格式，但落库值必须去杠。

### 数据库迁移 (AutoMigrate)

GORM 服务启动时通过 `AutoMigrate` 自动同步表结构，避免手动维护 SQL 迁移脚本与 GORM 模型定义不一致：

```go
// app/{service}/internal/svc/servicecontext.go
func NewServiceContext(c config.Config) *ServiceContext {
    gormDB := gormx.MustOpenWithConf(c.DB)

    if err := gormDB.AutoMigrate(
        &gormmodel.Plan{},
        &gormmodel.PlanBatch{},
        &gormmodel.PlanExecItem{},
        &gormmodel.PlanExecLog{},
    ); err != nil {
        panic(err)
    }
    // ...
}
```

**What**: `AutoMigrate` 根据 GORM struct tag（`comment`、`size`、`index`、`uniqueIndex`、`type`）自动创建或更新表结构，不会删除已有列。

**Why**: 确保开发、测试、生产环境的表结构与模型定义一致，减少手动 SQL 管理成本。`size` tag 控制 `VARCHAR(n)` 宽度，`comment` tag 生成列注释，`index`/`uniqueIndex` tag 自动建索引。

**When**: 每次服务启动时调用。只在 `AutoMigrate` 内列出本服务拥有的表，不要跨服务迁移。

### 时间值精度

GORM `timestamp` 列使用 `autoCreateTime:milli` / `autoUpdateTime:milli`（毫秒精度），与旧 `carbon` 字符串 `ToDateTimeMicroString()` 不兼容。当 GORM 模型通过 Scan 填充后构造 protobuf 响应时，不再依赖 `carbon`，直接用字段的 `time.Time` 值。

### Legacy 旧系统兼容软删除

Legacy GORM 模型使用旧系统兼容字段：

| 字段 | 含义 |
| --- | --- |
| `is_deleted` | 删除状态，`0` 未删除，`1` 已删除 |
| `delete_time` | 删除审计时间，不作为删除状态判断 |

`LegacySoftDeleteMixin` 使用 `gorm.io/plugin/soft_delete` 的 flag 模式，删除时由插件负责把 DELETE 转成 UPDATE、设置 `is_deleted` 并填充 `delete_time`。业务代码判断删除状态只能看 `is_deleted`，不要用 `delete_time.Valid` 推断。

### LegacyStringBaseModel 在业务服务内的使用

**What**: 业务服务迁移到 GORM 时，本地模型放在 `app/{service}/model/gormmodel/`，优先使用 `LegacyStringBaseModel` 表达 string/UUID 主键语义。

**Why**: 这样可以把 GORM 迁移限制在服务内，不污染根目录 go-zero 生成 DAO；同时 string/UUID 主键和业务唯一标识、外部协议字段更容易保持一致。

**Example**:
```go
type Plan struct {
    gormx.LegacyStringBaseModel
    gormx.VersionMixin

    PlanId string `gorm:"column:plan_id"`
}

type PlanBatch struct {
    gormx.LegacyStringBaseModel
    gormx.VersionMixin

    PlanPk string `gorm:"column:plan_pk"` // migrated-side pk sync field
}
```

**Notes**:
- 现有 SQLx / go-zero 生成 DAO 仍可保留 `int64` 自增主键，直到服务侧完成后续 schema 迁移。
- 如果迁移侧字段表示“上游/未来 UUID 主键同步”，Go 类型就应按 `string` 建模，不要在本地 GORM 模型里继续保留 `int64` 语义。
- `VersionMixin` 只在确实需要乐观锁的表上嵌入，日志类高频写表不要默认加。

### JSON 原文载荷字段

跨 MySQL、PostgreSQL、GaussDB 和 SQLite 运行的模型不要使用 `type:json` 或 `type:jsonb`。项目内 JSON 原文载荷按字符串保存，GORM tag 使用 `type:text`；长度未知时也使用 `text/TEXT`，不要为了 PostgreSQL 查询能力牺牲 MySQL 兼容性。

```go
// Good: 跨库兼容，保存原始 JSON 文本
RawJSON string `gorm:"column:raw_json;type:text;comment:完整事件原始JSON"`

// Bad: jsonb 只适合 PostgreSQL，会破坏 MySQL AutoMigrate/DDL 兼容性
RawJSON string `gorm:"column:raw_json;type:jsonb;default:'{}'"`
```

MySQL `TEXT` 默认值在不同版本和 SQL mode 下兼容性差。JSON 原文 text 字段不要写 `default:'{}'` / `default:'[]'`，由业务写入时提供空对象、空数组或空字符串。当前示例见 `app/djicloud/model/gormmodel/dji_event.go` 和 `app/djicloud/model/gormmodel/dji_osd_state.go`；trigger 计划表的 `recurrence_rule` / `payload` 也使用 `type:text`，对应 SQL 为 `TEXT`。

Legacy 通用字段生命周期由 `LegacyBaseModel` / `LegacyStringBaseModel` 的 GORM Model Hook 编排：create 填充存在的创建/更新审计字段和 `tenant_id`，update 只填充更新审计字段，delete 不额外写删除审计字段，避免干扰 `soft_delete` 插件生成的 UPDATE。`LegacyStringBaseModel` 在 create 时为空 ID 自动生成 UUID，预设 ID 保持不变。

`RegisterCallbacks()` 仍会在 `Open()` 和测试辅助函数里注册，但当前 `beforeCreateHook` / `beforeUpdateHook` / `beforeDeleteHook` 都是 no-op 占位，不注入审计、租户或删除字段。Legacy 路径完全由模型 hook 负责，非 Legacy 模型如果要自动填充通用字段，必须自己实现 GORM hook 并调用 mixin capability 方法。

`Restore` 只适合 gormx 已知软删字段：标准 `deleted_at`、Legacy `is_deleted/delete_time`，以及过渡兼容的 `del_state`。字段形态复杂或恢复规则包含业务条件时，业务侧使用 `Unscoped()` 和显式 `Updates(...)`。

### 迁移审计检查清单

从 sqlx/squirrel 迁移到 gormx 后，必须逐项检查以下内容确保业务一致：

1. **soft_delete 条件** — `gorm.io/plugin/soft_delete` 自动为 `Model().Updates()` 注入 `is_deleted = 0`，但 Raw SQL、子查询、JOIN 需显式写。
2. **状态守卫** — `status IN ?` / `status NOT IN ?` 条件是否与旧代码一致。
3. **字段映射** — 尤其是 `last_message` vs `last_reason`，不同 exec_result 分支下字段来源不同，容易写错。
4. **乐观锁** — `VersionMixin` 使用 `optimisticlock.Version`，`Save()` 自动 CAS；`LockTriggerItem` 等原子操作需显式 `WHERE version = ?`。
5. **预生成主键** — `LegacyStringBaseModel.BeforeCreate` 钩子自动生成去杠 UUID，业务侧不应再手动调用 `tool.UUID()` / `tool.SimpleUUID()` 为主键赋值。
6. **事务边界** — GORM `Transaction` 闭包内返回 error 即回滚，与旧 `sqlx.Transact` 行为一致，但事务后通知逻辑应与旧代码相同（事务外执行）。
7. **Raw SQL** — 改为 `db.Raw().Scan()`，占位符 `?` 与旧 squirrel 一致，不需要额外 `Dollar` 转换。
8. **`Save()` 全字段更新** — GORM `Save()` 更新所有非零字段，旧代码可能用 `Updates(map)` 只更新指定字段；需要确保不会把旧值（如 `paused_time`）意外覆盖为空。

### 数据库类型判断

需要根据数据库类型决定 SQL 方言时（如随机函数 `RAND()` vs `RANDOM()`），统一使用 `gormx.DatabaseType` 而非临时的 `isPostgres bool`：

```go
// 调用方：从 *gorm.DB 获取 dbType
dbType := gormx.GetDatabaseTypeFromDialector(db)

// 被调用方：使用 DatabaseType 常量判断
func clauseOrderBy(dbType gormx.DatabaseType) string {
    switch dbType {
    case gormx.DatabasePostgres, gormx.DatabaseSQLite:
        return "RANDOM()"
    default:
        return "RAND()"
    }
}
```

**Why**: `gormx.DatabaseType` 是 golangx 统一的数据库类型常量，新加入 SQLite 等类型时只需在 switch 中补充 case，调用方不需要变更签名。避免用 `Dialector.Name() == "postgres"` 等字符串比较。GaussDB PG 兼容模式当前按 PostgreSQL 处理。

**How**: `gormx.GetDatabaseTypeFromDialector(*gorm.DB)` 从 dialector 实例类型推断，覆盖 MySQL / Postgres / SQLite。

### 时间字段类型

盒子端会使用 SQLite 作为一等运行环境。`mattn/go-sqlite3` 读取 `time.Time` 时只把 declared type 精确等于 `timestamp` / `datetime` / `date` 的列解析为时间；`timestamp(6)`、`timestamp(0)` 会按 `string` 返回，扫描到 `time.Time` 时失败。

因此 gormx 基础时间 mixin 使用无精度 `timestamp`：

```go
type TimeMixin struct {
    CreatedAt time.Time `gorm:"type:timestamp;autoCreateTime:milli"`
    UpdatedAt time.Time `gorm:"type:timestamp;autoUpdateTime:milli"`
}

type LegacyTimeMixin struct {
    CreateTime time.Time `gorm:"column:create_time;type:timestamp;autoCreateTime:milli"`
    UpdateTime time.Time `gorm:"column:update_time;type:timestamp;autoUpdateTime:milli"`
}
```

需要秒级业务时间时，不通过 `timestamp(0)` 表达，而是在写入前清精度，例如 `tool.NowStartOfSecond()`。已有生产表需要改精度时另写 SQL migration，不通过改 mixin tag 假设 AutoMigrate 会安全改列。

源文件：`common/gormx/model.go`、`common/gormx/model_legacy.go`；SQLite driver 行为见 `github.com/mattn/go-sqlite3` 的 `SQLiteTimestampFormats` 和 declared type 判断。

### GORM Model Hook 与 gormx Callback 的区别

| 机制 | 定义方式 | `SkipHooks: true` 影响 |
|------|----------|------------------------|
| **Model Hook** | 模型实现 `BeforeUpdate(tx *gorm.DB) error` | 跳过 |
| **Callback** | `RegisterCallbacks` 注册的全局回调 | 不跳过 |

gormx 全局 callback 当前是注册/扩展占位，不写通用字段。审计、租户等通用字段由模型自己的 GORM hook 负责；`SkipHooks: true` 会跳过模型 hook，因此也会跳过这类字段填充：

```go
// SkipHooksUpdate 跳过 Model Hook；当前 gormx callback 不会补写审计字段
gormx.SkipHooksUpdate(db.WithContext(ctx), &model, map[string]any{"name": "new"})
```

源文件：`callbacks.go`、`hook_helpers.go`。

非 Legacy 模型如需自动审计/租户写入，应在模型上显式实现 GORM hook，并调用 `AuditMixin`、`StringAuditMixin`、`AuditWithoutDeleteMixin`、`StringAuditWithoutDeleteMixin`、`TenantMixin` 提供的 capability 方法。不要依赖 `RegisterCallbacks` 自动注入字段。

`BeforeDelete` 不适合在 Legacy 路径里手写删除审计字段。当前实现让 `soft_delete` 插件负责生成更新语句；如果在 `BeforeDelete` 里额外 `SetColumn`，会破坏插件生成的软删变量和 SQL 形态。

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

### Don't: 依赖 Model Hook 为 `SoftDeleteMixin` 模型设置删除审计字段

`SoftDeleteMixin` 使用 `gorm.DeletedAt`，GORM 内置软删除走 `db.Session(NewDB: true)`，Model Hook 中的 `SetColumn` 不可靠。`LegacySoftDeleteMixin` 使用 `soft_delete` 插件时，也不要在 `BeforeDelete` 中额外写复杂 SQL；删除机制由插件负责，业务有特殊删除审计规则时应在业务侧显式更新。

### Gotcha: 软删后再写回需先 Restore

旧表模型（`LegacyBaseModel`，含 `is_deleted/delete_time`）伪删除后，`FirstOrCreate` 查不到该记录，可能撞唯一索引。先调用 `gormx.Restore` 恢复：

```go
gormx.Restore(tx.DB, &DjiDeviceTopo{}, "gateway_sn = ? AND sub_device_sn = ?", gatewaySn, sub.SN)
tx.WithContext(ctx).Where(where).Assign(updateData).FirstOrCreate(&topoRecord)
```

`Restore` 内部根据模型判断是 Legacy 表（清空 `delete_time`，重置 `is_deleted`）还是新表（清空 `deleted_at`），走正常模型 update hook。过渡期仍兼容 `del_state` 字段。

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
