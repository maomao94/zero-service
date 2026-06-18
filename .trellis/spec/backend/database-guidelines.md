# 数据库规范

> 涉及 MySQL、PostgreSQL、SQLite、TDengine、Redis、Kafka 或缓存时，先复用现有 model、client、cache、config 和 `common/` 封装。

## 基本原则

- 不在 Logic 中直接拼接连接串、账号、密码或环境参数。
- 数据库、Redis、消息队列和第三方客户端通过配置结构和 `ServiceContext` 注入。
- 先查现有 model、DAO、cache、client 和相邻服务实现，再新增持久化代码。
- 复杂数据流先画清：接口契约 → Logic → Model/SDK/Client → 存储或消息系统。
- 所有跨层调用传递 `context.Context`，便于超时、取消和链路追踪。

## Model 和生成脚本

项目提供模型生成脚本：

```bash
cd model
sh genModel.sh
sh genPgModel.sh postgres <table_name>
sh genModelSql.sh
```

- 使用脚本生成模型后，必须检查生成代码 diff。
- 生成代码非必要不手改；需要调整字段、索引或表结构时，优先改 SQL/schema 或生成配置。
- 业务逻辑不要绕过 model/client 直接访问底层连接，除非相邻模块已有同类模式。

## SQL 变更

- 表结构、初始化数据、修复数据等独立 SQL，应放入项目约定 SQL 目录；如果目标模块已有 SQL 目录，优先跟随模块现有位置。
- SQL 文件名建议：`yyyyMMdd-{需求号或Trellis任务号}-{简短说明}.sql`。
- SQL 内容要能和 Trellis task、Backlog 条目或变更说明关联，方便追踪上线影响。
- 不在 SQL、配置或日志中提交真实账号、密码、连接串、内网地址或对象存储配置。

## Scenario: GaussDB PG 空串即 NULL 字段

### 1. Scope / Trigger

- Trigger: 修改 PostgreSQL/GaussDB 兼容表的可空字符串字段、GORM model、DB schema 或跨层响应映射时必须检查本契约。
- GaussDB PG 兼容模式可能把 `''` 按 `NULL` 处理；外部协议字段可缺省时，不能用 `not null default:''` 表示"无值"。

### 2. Signatures

- DB column: `track_id varchar(64) NULL`，不要加 `NOT NULL`，不要依赖 `DEFAULT ''`。
- GORM field: `TrackId sql.NullString \`gorm:"column:track_id;type:varchar(64);index;comment:..."\``。
- Write mapping: `sql.NullString{String: ext.TrackID, Valid: ext.TrackID != ""}`。
- Response mapping: protobuf/API `string track_id` 继续返回 `item.TrackId.String`，`NULL` 对外表现为 `""`。

### 3. Contracts

- Request/event field: 外部 `ext.track_id` 类型为 string，可缺省或为空。
- Storage contract: `ext.track_id == ""` 或字段缺省时，数据库保存 `NULL`。
- Storage contract: `ext.track_id != ""` 时，数据库保存原字符串。
- Response contract: 现有 API string 字段不改契约，DB `NULL` 映射为空字符串。
- Migration contract: 既有 `NOT NULL` 列需要执行 schema 变更允许 NULL，不能只改 GORM tag。

### 4. Validation & Error Matrix

- `track_id NOT NULL` + 上游空/缺省 -> GaussDB 写入 `NULL`，触发 `SQLSTATE 23502`。
- 同一事务首次写入失败后继续执行 SQL -> 触发 `SQLSTATE 25P02`，这是级联错误，不是根因。
- GORM string 字段扫描 DB `NULL` -> 可能扫描失败或丢失 NULL 语义；应使用 `sql.NullString`。
- API 仍要求 string -> 在 Logic/mapper 层显式取 `.String`，不要把 `sql.NullString` 泄漏到 protobuf 响应。

### 5. Good/Base/Bad Cases

- Good: 外部字段可缺省且目标库是 GaussDB PG，model 用 `sql.NullString`，DB 列允许 NULL，响应层转换为空字符串。
- Base: 外部字段业务上必填，保留 `string` + `not null`，但 handler 必须先校验空值并拒绝写入。
- Bad: 对可缺省字段使用 `string` + `not null;default:''`，期望空串能绕过非空约束。

### 6. Tests Required

- Model schema test: 断言该字段 `field.NotNull == false` 且 `field.HasDefaultValue == false`。
- Handler/write test: 断言上游缺省字段不会导致 upsert 报错，且有值时能保存原字符串。
- Logic/API test: 断言 DB `NULL` 映射到响应 `""`，非空值原样返回。
- Regression log check: 看到 `23502` 后的 `25P02` 时先查事务内第一条 SQL 错误。

### 7. Wrong vs Correct

#### Wrong

```go
TrackId string `gorm:"column:track_id;type:varchar(64);index;not null;default:''"`

record.TrackId = ext.TrackID
```

#### Correct

```go
TrackId sql.NullString `gorm:"column:track_id;type:varchar(64);index"`

record.TrackId = sql.NullString{String: ext.TrackID, Valid: ext.TrackID != ""}
response.TrackId = record.TrackId.String
```

## 查询和事务

- 简单 CRUD 优先复用生成 model 方法。
- 批量、事务和聚合查询先找同库同服务的既有写法。
- 需要事务时显式说明事务边界、提交条件和回滚条件，不把多个外部系统操作伪装成单数据库事务。
- Redis/cache 更新要明确缓存 key、TTL、失效策略和数据一致性边界。

### Scenario: GORM ErrRecordNotFound 按业务场景处理

#### 1. Scope / Trigger

- Trigger: 使用 GORM `First` / `Last` / `Take` 查询单条记录，或列表接口补充 OSD、State、Topo、Task 等关联数据时，必须判断记录缺失是业务错误还是正常缺省。

#### 2. Signatures

- Error sentinel: `gorm.ErrRecordNotFound`。
- Check pattern: `errors.Is(err, gorm.ErrRecordNotFound)`，不要直接依赖字符串或只用 `err == gorm.ErrRecordNotFound`。
- GORM logger config: `IgnoreRecordNotFoundError bool` 只控制 SQL trace 是否打印 record-not-found，不改变 `db.First(&v).Error` 的返回值。

#### 3. Contracts

- 必需主记录缺失：例如详情页按 ID/SN 查询主资源，返回 `extproto.Code__1_02_RECORD_NOT_EXIST` 或相邻 Logic 约定的结构化不存在错误。
- 可选关联数据缺失：例如列表项补充遥测快照、状态快照、任务状态，返回空字段并继续组装主列表，不打 error 日志。
- 真实数据库错误：连接、SQL、扫描、权限等非 `ErrRecordNotFound` 错误继续返回或包装，不能吞掉。

#### 4. Validation & Error Matrix

- 主资源 `First` 返回 `ErrRecordNotFound` -> 返回记录不存在业务错误。
- 可选关联 `First` 返回 `ErrRecordNotFound` -> 返回 `nil`，对应响应字段保持空。
- 任意查询返回非 `ErrRecordNotFound` -> 保留原始错误，按调用边界包装或记录。
- 仅设置 `IgnoreRecordNotFoundError=true` -> 只减少 GORM SQL 日志，业务层 `err` 仍需显式处理。

#### 5. Good/Base/Bad Cases

- Good: 列表接口主查询成功后，关联快照查不到时跳过该字段，避免 `[list-devices] query associated data failed: err=record not found` 这类噪声日志。
- Base: 必需详情接口查不到主记录时返回结构化不存在错误，让调用方明确资源不存在。
- Bad: 全局忽略所有 `ErrRecordNotFound`，导致真正缺失的主资源被误当成功；或完全不处理，导致可选关联缺失刷 error 日志。

#### 6. Tests Required

- Logic test: 主记录缺失时断言返回记录不存在错误码。
- Logic test: 可选关联缺失时断言接口仍成功，关联响应字段为空，且不产生业务 error。
- Logger/config test: 如变更 GORM logger，断言 `IgnoreRecordNotFoundError` 只影响日志输出，不改变查询错误返回。

#### 7. Wrong vs Correct

##### Wrong

```go
if err := db.Where("device_sn = ?", sn).First(&state).Error; err != nil {
    return err // 可选关联缺失也会触发业务 error 日志
}
```

##### Correct

```go
if err := db.Where("device_sn = ?", sn).First(&state).Error; err != nil {
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil
    }
    return err
}
```

## gormx 分页查询

### Convention: 使用 gormx.QueryPage

**What**: 分页查询统一使用 `gormx.QueryPage`，不手动写 Count + Find。

**Why**: `QueryPage` 封装了分页参数归一化、count=0 提前返回、offset/limit 计算，避免重复代码和遗漏边界处理。

**Signature**:
```go
func QueryPage[T any](db *gorm.DB, page, pageSize int, dest *[]T) (*PageResult[T], error)
func QueryPageData[T any](db *gorm.DB, page, pageSize int) ([]T, error)
```

- `QueryPage` — 完整分页，返回 PageResult（含 total, page, pageSize, totalPages）
- `QueryPageData` — 仅取数据分页，不计算 total，返回 `[]T`

**Good/Base/Bad Cases**:

- Good: `gormx.QueryPage(db.Order("id DESC"), page, pageSize, &devices)`
- Base: 手动 Count + Find，但使用 `gormx.NormalizePage` 归一化参数
- Bad: 手动写 `db.Count().Offset().Limit().Find()`，不处理 count=0

### Convention: 自定义 SQL + JOIN 分页

**What**: 需要 JOIN、表别名、自定义 WHERE 时的分页标准写法。

**Pattern**（以 `dji_hms_alert LEFT JOIN dji_device` 为例）：

```go
// 1. 建 base：Model 保持 schema 推断，Table 设定别名
db := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.DjiHmsAlert{}).
    Table("dji_hms_alert dha").
    Joins("LEFT JOIN dji_device dd ON dha.gateway_sn = dd.gateway_sn")

// 2. 条件全部用别名手写 SQL
if in.GatewaySn != "" {
    db = db.Where("dd.gateway_sn = ?", in.GatewaySn)
}

// 3. 分页：WithContext 克隆新 Statement，追加 Order + Select（不影响 countDB）
queryDB := db.WithContext(l.ctx).
    Select("dha.*, dd.is_online").
    Order("dha.reported_at DESC,dha.id DESC")
countDB := db   // 同一 base，Count 内部自动替换 Select 为 count(*)

pageResult, err := gormx.QueryPage(queryDB, int(in.GetPage()), int(in.GetPageSize()), &alerts)
```

**Why**: `db.WithContext(l.ctx)` 内部走 `Session({Context: ctx})` → `clone=1` → 创建新的独立 `Statement`。ORDER BY 和自定义 Select 写在新 Statement 上，不影响原 `db`。`Count()` 内部自动替换 Select 为 `count(*)`，无需单独提 countDB。

**注意**:
- LEFT JOIN 在唯一关联键上不产生重复行，count 无需 GROUP BY
- 必须显式 `Select("主表.*")`，否则 GORM 会把 JOIN 表的所有列也 SELECT 出来，导致 Scan 异常
- 别名统一：`Table("dji_hms_alert dha")`（带空格，GORM 输出 `FROM dji_hms_alert dha`，无引号，PG/MySQL/高斯均兼容）

### Gotcha: GORM Statement 共享陷阱

**Symptom**：`db.Order(...)` 后 `countDB := db` 的 count SQL 也带上了 ORDER BY。

**Cause**：GORM 链式调用中，当 `db.clone == 0` 时 `getInstance()` 返回同一 `DB` 对象，共享底层 `Statement`。`Order()` 将 ORDER BY 写入 `Statement.Clauses`，`countDB` 指向的同一 Statement 因此被污染。

```go
// Wrong — countDB 也被 Order 污染
db := l.svcCtx.DB.WithContext(l.ctx).Model(&T{})
queryDB := db.Order("id DESC")  // Statement 共享，Order 写入了 db
countDB := db                   // 拿到同一个 Statement，含 ORDER BY

// Correct — 用 WithContext 克隆新 Statement
queryDB := db.WithContext(l.ctx).Order("id DESC")  // 新 Statement，Order 不污染 db
countDB := db  // 干净的 Statement
```

**Fix**：用 `db.WithContext(l.ctx)` 替代 `db.Session(&gorm.Session{})` 作为克隆手段。`WithContext` 是 GORM 官方推荐的 Statement 隔离方式，语义清晰。

**Prevention**：凡是对 `db` 链式调用了会修改 Statement 的方法（`Order`、`Select`、`Limit`、`Offset` 等），后续需要 "原始 db" 时，必须先用 `WithContext` 克隆再修改。

**Reference**:
- `common/gormx/pagination.go`
- `app/djicloud/internal/logic/listhmsalertslogic.go`

## 时间格式化

### Convention: 使用 carbon 格式化时间

**What**: proto/API 层时间字段统一使用 `string` 类型，格式 `YYYY-MM-DD HH:mm:ss.SSSSSS`，UTC+8 时区。Go 层使用 `carbon.CreateFromStdTime().ToDateTimeMicroString()` 转换。

**Why**: 项目全局默认时区已在 `common/carbonx/carbonx.go` 设置为 `carbon.Shanghai`，统一时间格式避免前端解析歧义。

**Contract**:
- proto 字段: `string reported_at = N;`
- 注释: `// reported_at 设备上报时间，UTC+8 格式：YYYY-MM-DD HH:mm:ss.SSSSSS。`
- Go 转换: `carbon.CreateFromStdTime(m.ReportedAt).ToDateTimeMicroString()`

**Reference**:
- `common/carbonx/carbonx.go` - 全局时区配置
- `app/djicloud/djicloud.proto` - 所有 `reported_at` 字段
- `app/djicloud/internal/logic/helper.go` - 转换函数示例

## 常见错误

- 新增功能前未搜索已有 model/client/cache，导致重复封装。
- 为单个 Logic 私有逻辑创建过度通用的公共 DAO。
- 手写生成模型文件，后续生成时被覆盖。
- 忽略 `context.Context`，导致超时、取消和链路追踪失效。
- 将真实数据库连接、远程地址或账号写入示例、日志、文档或提交信息。

## gormx 包约定与陷阱

### Convention: gormx 文件组织

`common/gormx` 按职责拆分源文件，不再集中在一个大文件：

| 文件 | 职责 |
| --- | --- |
| `config.go` | `Config` 结构体，`parseLogLevel` |
| `db.go` | `DB` 包装类型，事务/租户/迁移方法 |
| `options.go` | `Option` 和 `dbOptions`，`defaultDBOptions` |
| `open.go` | `Open`、`OpenWithConf`、`OpenWithDialector`、`OpenWithRawDB` |
| `batch.go` | 批量 CRUD（`BatchInsert`、`BatchUpdateByIds`、`BatchDeleteByIds`） |
| `batch_tenant.go` | 租户感知批量 CRUD |
| `delete.go` | `SoftDelete`、`UnscopedDelete`、`UnscopedDeleteWithTenant` |
| `restore.go` | `Restore`、`RestoreWithTenant`、`hasLegacyDeleteFields` |
| `hook_helpers.go` | `SkipHooksUpdate`、`SkipHooksCreate`（跳过 Model Hook，不跳过 gormx 审计 callback） |
| `tenant_query.go` | `withTenantQuery`（未导出） |
| `tenant_scope.go` | `TenantScope` 等 scope 函数、`WithTenantContext` |

新增功能前确认职责落到对应文件，不往不相关的文件里塞函数。

### Convention: OpenWithConf 信任 go-zero tag 默认值

`OpenWithConf` 不做程序化默认值兜底。所有默认值由 go-zero config tag（`default=100`、`default=true` 等）在配置加载阶段填入。

```go
// Wrong — 零值 Config 直接传给 OpenWithConf
db, _ := gormx.OpenWithConf(gormx.Config{
    DataSource: dsn,
}) // MaxIdleConns=0, SkipDefaultTransaction=false, ParameterizedQueries=false

// Correct — 通过 go-zero 加载配置，tag 默认值已生效
var c config.Config // go-zero 根据 default= 标签自动填充
db := gormx.MustOpenWithConf(c.DB)

// Correct — 编程式构造使用 Open + Option
db, _ := gormx.Open(dsn,
    gormx.WithMaxIdleConns(100),
    gormx.WithMaxOpenConns(100),
)
```

### Convention: LogLevel 必须使用 options 约束

`Config.LogLevel` tag 包含 `options=[silent,error,warn,info]`。go-zero 加载配置时会校验该字段，不在列表内的值会被拒绝，避免静默接受无效日志级别。

```go
LogLevel string `json:",optional,default=error,options=[silent,error,warn,info]"`
```

### Convention: TimeMixin 必须包含自动时间标签

新表 `TimeMixin` 和旧表 `LegacyTimeMixin` 都要有 GORM 自动时间标签，否则 `CreatedAt`/`UpdatedAt` 不会自动填充。

```go
// Correct
type TimeMixin struct {
    CreatedAt time.Time `gorm:"type:timestamp(6);autoCreateTime:milli" json:"created_at"`
    UpdatedAt time.Time `gorm:"type:timestamp(6);autoUpdateTime:milli" json:"updated_at"`
}
```

### Convention: Open* 入口函数必须对指针入参做 nil 校验

`OpenWithDialector(nil)` 和 `OpenWithRawDB(nil, ...)` 必须返回明确错误，不能 panic 或绕过校验进入 `Open`。

```go
// Correct
func OpenWithDialector(dialector *gorm.Dialector, opts ...Option) (*DB, error) {
    if dialector == nil {
        return nil, errors.New("dialector is required")
    }
    // ...
}
```

### Don't: 在 GORM Scope 函数中使用 `HasTenantField`

`HasTenantField(db)` 依赖 `db.Statement.Schema`，但 GORM 的 scope 函数在 Schema 解析**之前**执行。因此在 scope 内调用 `HasTenantField` 始终返回 `false`，导致租户过滤形同虚设。

```go
// Wrong — HasTenantField always returns false in scope functions
func TenantScope(ctx context.Context) func(db *gorm.DB) *gorm.DB {
    return func(db *gorm.DB) *gorm.DB {
        if !HasTenantField(db) { return db }  // BUG: schema not resolved yet
        return db.Where("tenant_id = ?", tenantID)
    }
}

// Correct — remove field check; SQL error if model has no tenant_id
func TenantScope(ctx context.Context) func(db *gorm.DB) *gorm.DB {
    return func(db *gorm.DB) *gorm.DB {
        return db.Where("tenant_id = ?", tenantID)
    }
}
```

### Don't: 依赖 Hooks 为 SoftDeleteMixin 模型设置删除审计字段

`SoftDeleteMixin` 使用 `gorm.DeletedAt`，GORM 内置软删除机制会：
1. 替换 Delete 回调为自定义实现
2. 通过 `db.Session(NewDB: true)` 创建新 Session
3. 新 Session 的 Statement 是克隆的，**丢弃**了 `beforeDeleteHook` 的 `SetColumn` 值
4. 最终 SQL 仅包含 `deleted_at`，不包含 `delete_user`/`delete_name`

对于使用 `SoftDeleteMixin` 的模型，删除审计字段需要在 service 层显式设置。

> 传统软删除模型（`LegacySoftDeleteMixin`——`del_state`/`delete_time`）不受影响，因为 `gorm.io/plugin/soft_delete` 的机制不同。

### Convention: GORM Model Hook 与 gormx Callback 的区别

GORM 有两种扩展机制，`SkipHooks` 只影响前者：

| 机制 | 定义方式 | `SkipHooks: true` 影响 |
|------|----------|------------------------|
| **Model Hook** | 模型实现 `BeforeUpdate(tx *gorm.DB) error` 等方法 | 跳过 |
| **Callback** | `db.Callback().Update().Before("gorm:update").Register(...)` | 不跳过 |

gormx 的 `beforeCreateHook`/`beforeUpdateHook`/`beforeDeleteHook` 是通过 `RegisterCallbacks` 注册的全局 callback，属于基础设施级审计逻辑，不受 `SkipHooks` 控制。

```go
// SkipHooksUpdate 跳过的是 Model Hook，不是 gormx 审计 callback
// 使用 SkipHooksUpdate 时，如果 ctx 中有 UserContext 且模型有 update_user 字段，
// 审计字段仍然会被写入
gormx.SkipHooksUpdate(db.WithContext(ctx), &model, map[string]any{"name": "new"})
```

**实践含义**：
- 需要跳过模型自定义 `BeforeUpdate` 逻辑 → 用 `SkipHooksUpdate`。
- 需要跳过 gormx 审计 callback → 当前不支持，也不应该跳过（审计是基础设施）。
- `setSchemaColumn` 会检查模型 schema 是否有目标字段，没有则静默跳过，不会报错。因此 `LegacyBaseModel`（无 `update_user` 字段）的更新不会写审计字段，但 `version + 1` 仍会生效。

### Gotcha: 软删后再写回需先 Restore

旧表模型（`LegacyBaseModel`，含 `delete_time/del_state`）伪删除后，如果再执行 `UpdateOrCreate`，普通 scoped 查询看不到该记录，可能撞唯一索引。应先调用 `gormx.Restore` 恢复，再走更新逻辑。

```go
// Correct — restore before upsert
for _, sub := range topo.SubDevices {
    if err := gormx.Restore(tx.DB, &DjiDeviceTopo{},
        "gateway_sn = ? AND sub_device_sn = ?", gatewaySn, sub.SN); err != nil {
        return err
    }
    if err := gormx.UpdateOrCreate(ctx, tx, &DjiDeviceTopo{}, where, &topoRecord, updateData); err != nil {
        return err
    }
}
```

`Restore` 内部会根据模型判断是旧表（清空 `delete_time/del_state`）还是新表（清空 `deleted_at`），并走正常 update callbacks。

### Gotcha: 批量 `Create` 中 Hook 仅对首条记录生效

GORM 的 `beforeCreateHook` 在 `Create([]T)` 批量插入时只对第一个元素设置 `tenant_id` 和审计字段：

```sql
-- 批量 create 2 条记录的实际 SQL
INSERT INTO model (tenant_id, create_user, ...) VALUES
  ("tenant-1", "uid-1", ...),  -- ✅ 第1条正确
  ("",        "",      ...)     -- ❌ 第2条审计字段为空
```

在需要完整审计的批量操作场景，应逐条 create 或使用 `BatchInsertWithTenant` 等封装函数。

### Gotcha: OpenTelemetry 注册失败时资源泄露

`Open()` 在 `registerOpenTelemetry` 失败后必须关闭已打开的底层 `sql.DB`，否则连接泄露。

```go
// Correct — close opened DB on OpenTelemetry registration failure
RegisterCallbacks(gormDB)
if err := registerOpenTelemetry(gormDB, options.openTelemetry); err != nil {
    closeOpenedDB(gormDB, options.rawDB)
    return nil, err
}
return &DB{DB: gormDB}, nil
```

### Gotcha: 测试 DB 必须关闭连接

`openTestDB` 等测试 helper 返回的 `sql.DB` 不会随进程退出自动清理（SQLite 内存库除外）。必须调用 `t.Cleanup(func() { _ = sqlDB.Close() })`。

### API 兼容性: `tracing.WithDBName` → `tracing.WithDBSystem`

`gorm.io/plugin/opentelemetry v0.1.16` 移除了 `tracing.WithDBName`，改为 `tracing.WithDBSystem`。后者设置 OpenTelemetry 的 `db.system` 属性（期望值为 `mysql`、`postgresql` 等数据库类型，而非数据库名称）。升级该插件版本时需同步修改调用代码。

### Convention: 单测必须开启 SQL 日志

`test_helpers_test.go` 中的 `openTestDB` 默认启用 `logger.Default.LogMode(logger.Info)`，确保每次 `go test` 都输出完整 SQL。AI 审查代码时必须分析 SQL 日志，验证：
- CREATE: 审计字段 + 租户 + 版本写入正确
- UPDATE: 仅有 `update_user`/`update_name`/`version+1`，无 `delete_*` 侧漏
- DELETE: 硬删除为 `DELETE FROM`（无 SET），软删除 SQL 符合预期
- 租户过滤: 跨租户操作返回 `rows:0`

### Design Decision: gormx 生产配置默认值

**Context**: 用户只需提供 `DataSource`（地址+端口+密码），其他配置应为生产最佳实践。

**Options Considered**:
1. 保守默认值（所有可选配置关闭）— 安全但性能差
2. 最佳实践默认值（按生产推荐开启）— 用户体验好但需文档说明
3. 必填所有配置 — 强制用户理解每个选项

**Decision**: 选择方案 2，默认值通过 go-zero config tag 的 `default=` 表达，不通过程序化兜底。

**Default Configuration**:

```go
type Config struct {
    // 必填：数据库连接地址，支持 MySQL/PostgreSQL/SQLite 自动识别
    DataSource string `json:",optional"`

    // 连接池：MaxIdleConns = MaxOpenConns，避免连接抖动（5-20ms/次）
    MaxIdleConns    int           `json:",optional,default=100"`
    MaxOpenConns    int           `json:",optional,default=100"`
    ConnMaxLifetime time.Duration `json:",optional,default=1h"`   // 有 LB 时缩短到 5-30min
    ConnMaxIdleTime time.Duration `json:",optional,default=5m"`   // 低流量时自动清理闲置连接

    // 日志：Error 级别 + 参数脱敏
    SlowThreshold             time.Duration `json:",optional,default=200ms"`
    LogLevel                  string        `json:",optional,default=error,options=[silent,error,warn,info]"`
    ParameterizedQueries      bool          `json:",optional,default=true"`   // 安全：日志不暴露参数值
    IgnoreRecordNotFoundError bool          `json:",optional,default=false"`

    // 性能：跳过单条操作的事务包裹
    QueryFields            bool `json:",optional,default=false"`
    SkipDefaultTransaction bool `json:",optional,default=true"`   // 单条写操作省掉 BEGIN/COMMIT，性能 +10-30%
    PrepareStmt            bool `json:",optional,default=false"`  // 保守：连接池切换/DB 重启后缓存失效

    // 链路追踪
    Trace TraceConfig `json:",optional"`
}
```

**Rationale**:

| 配置项 | 默认值 | 原因 |
|--------|--------|------|
| `MaxIdleConns` | 100 | = MaxOpenConns，避免连接抖动（每次创建/销毁 5-20ms 开销） |
| `ConnMaxIdleTime` | 5min | 低流量时自动清理闲置连接，防止被 DB 服务端单方面断开 |
| `ParameterizedQueries` | true | 安全：日志不暴露敏感参数（手机号、密码等），GDPR 合规 |
| `SkipDefaultTransaction` | true | 性能：单条写操作不再自动 BEGIN/COMMIT，提升约 10-30% |
| `PrepareStmt` | false | 保守：连接池切换或 DB 重启后预编译语句失效，某些驱动有兼容性问题 |

**Tests Required**:
- 验证 `DefaultGormLogger()` 默认 `ParameterizedQueries=true`（参数脱敏）
- 验证 `Open()` 函数传递所有配置到 `gorm.Config` 和 `sql.DB`
- 验证 `OpenWithConf()` 正确映射所有 Config 字段

**Wrong vs Correct**:

```go
// Wrong — 忽略默认值，手动设置所有配置
db := gormx.MustOpenWithConf(gormx.Config{
    DataSource:           dsn,
    MaxIdleConns:         10,   // 太低，连接抖动
    MaxOpenConns:         100,
    ParameterizedQueries: false, // 安全风险：日志泄露敏感数据
    SkipDefaultTransaction: false, // 性能损失：每条写操作多 2 次 DB 往返
})

// Correct — 只填 DataSource，其他用 go-zero tag 默认值
db := gormx.MustOpenWithConf(gormx.Config{
    DataSource: dsn,
})

// Correct — 编程式构造使用 Open + Option
db, _ := gormx.Open(dsn,
    gormx.WithMaxIdleConns(50),
    gormx.WithLogger(mylogger),
)
```

**Extensibility**:
- 需要调整连接池时，使用 `WithMaxIdleConns()`、`WithMaxOpenConns()` 等 Option 函数
- 需要事务时，使用 `db.Transact(func(tx *gormx.DB) error { ... })` 手动包裹
- 需要 PrepareStmt 时，显式设置 `PrepareStmt: true` 并注意连接池稳定性
