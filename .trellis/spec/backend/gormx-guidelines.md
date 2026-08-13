# GORM 与数据访问规范

## 适用范围

修改 `common/gormx`、GORM model/store、分页、租户、乐观锁、Upsert、事务或时间空值时读取。

## 连接与配置

- 代码内明确提供 `driver.Config` 时使用 `gormx.Open`；从 go-zero 配置加载时使用 `gormx.OpenWithConf`，让 tag 默认值生效。
- 不用未经过配置加载的零值 `gormx.Config` 调 `OpenWithConf`，因为 tag 默认值不会自动填充普通 Go struct。
- 连接失败、方言不支持和必需插件安装失败必须返回错误；不要静默切换数据库或继续使用 nil DB。
- 测试优先使用包内现有 SQLite helper，但涉及方言 SQL、锁和 `RETURNING` 时必须补对应数据库验证或明确风险。

依据：`common/gormx/open.go`、`common/gormx/README.md`、`common/gormx/*_test.go`。

## 模型组合

- 新模型按需组合原子 mixin：`IDModel`（或 `StringIDModel`）提供主键，`TimeMixin` 提供时间戳，`SoftDeleteMixin` 提供软删除，`VersionMixin` 提供乐观锁版本号，`TenantMixin` 提供租户隔离；不要嵌入多个重复定义主键/时间/软删字段的 mixin。
- `Version` 是显式乐观锁能力，不是所有模型默认字段；只有更新路径实际使用版本条件时才引入。
- `LegacyBaseModel` / `LegacyStringBaseModel` 只服务现有旧表兼容，新表不复制历史字段布局；旧表独立 mixin（`LegacyIDMixin`、`LegacyStringIDMixin`、`LegacyTimeMixin`、`LegacySoftDeleteMixin`）也可按需组合。
- `LegacyBaseModel` / `LegacyStringBaseModel` 已通过 `gorm.io/plugin/soft_delete` 为 `is_deleted` 注册默认查询 scope；普通 `Model`、`First`、`Find`、`Count` 和 `Updates` 不重复手写 `is_deleted = 0`。只有使用 `Unscoped()`、原生 SQL、`Table(...)` 或无法触发 model scope 的查询时，才由调用方显式处理软删除条件。
- 生命周期字段由 model hook 或明确 store 写入。`common/gormx/callbacks.go` 的全局 callback 是扩展入口，不假设它已替所有模型维护字段。

依据：`common/gormx/model.go`、`common/gormx/callbacks.go`、现有服务模型。

## 查询与写入

- Store/Model 拥有 SQL/GORM 表达式、事务和字段更新范围；Logic 传递领域参数，不拼接列名或 SQL。
- 动态排序、游标列和字段名必须经白名单或 identifier 校验；值使用参数绑定。
- 使用 `gormx.NewPageParams` 和访问器创建分页参数；不要绕过私有字段或复制分页规范化逻辑。
- `gormx.QueryPage` 负责页码/页大小边界；调用方仍要提供稳定排序，避免翻页重复或遗漏。
- 冲突写入优先使用 `gormx.Upsert` 的 `clause.OnConflict` 封装，并明确 conflict columns 与 update columns；禁止各服务随意拼一套方言 SQL。

依据：`common/gormx/pagination.go`、`common/gormx/upsert.go`、相邻 store 测试。

## 租户、用户与并发所有权

- 用户/租户通过 `common/gormx` context helper 传递；普通 `TenantScope` 在缺失租户时不加过滤，`TenantScopeStrict` / `WithTenantStrict` 则追加恒假条件并返回空结果，两种语义不可混用。context 中存在租户 ID 时，两类 scope 都会查询 `tenant_id`；目标 model 无该列会返回数据库错误。
- 条件更新必须检查 error 和 `RowsAffected`。幂等成功、目标不存在和竞争失败是不同契约，应由 store 明确返回。
- 查询无记录使用 `model.ErrNotFound`（`model` 包定义通用数据库哨兵错误）；条件更新 `RowsAffected == 0` 使用 `model.ErrNoRowsUpdate` 或领域专用竞争错误（如 `common/crontask` 包的 `ErrNotFound` / `ErrUpdate`），不能把 CAS 未命中伪装成记录不存在。`common/gormx` 本身不定义错误哨兵，直接使用 `gorm.ErrRecordNotFound`。
- 并发状态更新使用事务、唯一约束、版本或 CAS 条件；完成路径只能更新自己拥有的字段，不能用整行 `Save` 覆盖调度/配置字段。
- 数据库可空时间使用 `sql.NullTime` 或指针；转换层与领域 `time.Time{}` 语义一一对应，禁止用远期时间伪装“无下次执行”。

依据：`common/gormx/user_context.go`、`common/gormx/tenant_scope.go`、`common/gormx/tenant_scope_test.go`、`common/gormx/db_test.go`、`app/trigger/internal/cronjob/db_store.go`、`app/ispagent/internal/crontask/db_store.go`。

## 反模式

- 用 `db.Save(&wholeModel)` 更新高并发状态表。
- 不检查 `RowsAffected` 就把竞争失败当成功。
- 把租户过滤留给每个 Logic 手写。
- 对已组合 GORM soft-delete mixin 的 model 普通查询重复手写 `is_deleted = 0`，造成 scope 所有权不清；`Table(...)`、原生 SQL和 `Unscoped()` 除外。
- 用字符串拼接动态列名、排序或未转义值。
- 复制旧 Spec 的通用禁令，绕过当前 `gormx.Upsert` 封装。

## 验证

- 测试创建、更新、未命中、重复、软删、租户缺失、版本冲突和事务回滚。
- 对 claim/complete 或状态机更新断言 SQL 条件与 `RowsAffected`，必要时并发执行。
- 运行目标 store/model 包测试；方言敏感逻辑不能只依赖 SQLite 通过。

## Scenario: GaussDB 字符串空值与非空列设计

### 1. Scope / Trigger

- 新增或修改字符串列的 `NOT NULL`、`DEFAULT`、GORM `default` tag、Go `string` / `sql.NullString` 类型，且服务支持 GaussDB/openGauss 时适用。
- GaussDB A/ORA 兼容模式将空字符串 `''` 视为 SQL `NULL`；B/MYSQL、C、PG 兼容模式不可直接套用该行为，先查询目标库兼容模式。

### 2. Signatures

```go
// 必填或使用非空领域哨兵：数据库 NOT NULL，Go 使用 string。
Status string `gorm:"column:status;type:varchar(32);not null;default:'unknown'"`

// 业务允许未知值：数据库列允许 NULL，Go 使用 sql.NullString。
Description sql.NullString `gorm:"column:description;type:varchar(255)"`
```

环境核验 SQL：

```sql
SHOW sql_compatibility;
SELECT '' IS NULL AS empty_string_is_null;
```

### 3. Contracts

- `DEFAULT` 只在 INSERT 省略该列或显式使用 `DEFAULT` 时生效；显式传入 SQL `NULL` 不会回退到列默认值，`NOT NULL` 列会报约束错误。
- 在 GaussDB A/ORA 兼容模式下，显式 `''` 按 `NULL` 处理，因此 `NOT NULL DEFAULT 'unknown'` 不能兜底应用显式写入的空字符串。
- `NOT NULL` 不等于“必须声明 DEFAULT”。若所有写入路径都显式提供合法非空值，可以没有默认值；若允许调用方省略列，则提供非空默认值作为数据库侧兜底。
- `sql.NullString{Valid:false}` 表示写入 SQL `NULL`，只能配合允许 NULL 的列；`sql.NullString{String:"", Valid:true}` 在 A/ORA 模式仍可能成为 SQL `NULL`。`sql.NullString` 不能解决 `NOT NULL` 列的空字符串问题。
- 业务必填字符串使用普通 `string` 并在写入前拒绝空值；业务允许“未知但记录仍需存在”且列必须非空时，使用有文档和测试的非空领域哨兵；业务语义本来就是未知/缺失时，列允许 NULL 并使用 `sql.NullString` 或指针。
- GORM model tag 中的默认值属于 schema 和 ORM create 行为的一部分，但不得假设所有 `Create`、map insert、`Select`、原生 SQL 或批量路径都会省略零值列；关键列应检查实际生成 SQL，必要时在 model/store 显式赋非空值。

依据：GaussDB FAQ [What Is the Relationship Between an Empty String and NULL?](https://support.huaweicloud.com/intl/en-us/distributed-devg-v8-gaussdb/gaussdb-12-1803.html)；openGauss [CREATE TABLE](https://docs.opengauss.org/en/docs/latest/sql_reference/create_table.html) 的 `DEFAULT` 契约；`app/djicloud/model/gormmodel/dji_device.go` 及 hook 测试。

### 4. Validation & Error Matrix

| 列定义 | INSERT 输入 | GaussDB A/ORA 结果 | 设计结论 |
| --- | --- | --- | --- |
| `NOT NULL DEFAULT 'unknown'` | 省略列或 `DEFAULT` | 保存 `unknown` | 默认值生效 |
| `NOT NULL DEFAULT 'unknown'` | 显式 `NULL` | 非空约束错误 | 默认值不生效 |
| `NOT NULL DEFAULT 'unknown'` | 显式 `''` | 转为 `NULL` 后报非空约束错误 | 写入前拒绝或改非空哨兵 |
| nullable，无默认值 | `sql.NullString{Valid:false}` | 保存 SQL `NULL` | 适合业务可空字段 |
| nullable | `sql.NullString{String:"", Valid:true}` | A/ORA 下保存 SQL `NULL` | 不能区分空串与 NULL |
| `NOT NULL`，无默认值 | 显式合法非空字符串 | 保存该值 | 合法，不强制要求 DEFAULT |

### 5. Good/Base/Bad Cases

- Good: 设备身份暂时未知但主记录必须存在，model 和 store 显式写入非空 `unknown`，后续真实拓扑覆盖哨兵。
- Base: 可选描述允许 SQL `NULL`，model 使用 `sql.NullString`，转换层明确映射 RPC 缺省值。
- Bad: 给 `NOT NULL` 列添加默认值后仍显式写 `""`，误以为数据库会使用默认值；或把字段改成 `sql.NullString` 却保留 `NOT NULL`。

### 6. Tests Required

- Model schema 测试断言列是否 `NOT NULL`、是否有非空默认值，以及 Go 字段类型符合领域空值语义。
- Store/hook 测试断言首次创建的关键字符串字段为真实非空值或领域哨兵，并断言后续真实值可覆盖哨兵。
- 方言敏感行为至少在目标 GaussDB 兼容模式执行：省略列、显式 `DEFAULT`、显式 `NULL`、显式 `''` 四组 INSERT；SQLite 测试不能替代该验证。
- 审查 GORM SQL，确认关键写入路径是省略列、写 `DEFAULT`，还是显式绑定值；不能只根据 struct tag 推断。

### 7. Wrong vs Correct

#### Wrong

```go
// NOT NULL + DEFAULT 保护不了显式空串；NullString(false) 也会写 NULL。
type Device struct {
	Type sql.NullString `gorm:"not null;default:'unknown'"`
}
device.Type = sql.NullString{String: "", Valid: true}
db.Create(&device)
```

#### Correct

```go
// 非空列使用普通 string，并在应用写入路径显式提供非空值。
type Device struct {
	Type string `gorm:"not null;default:'unknown'"`
}
device.Type = UnknownDeviceType
db.Create(&device)

// 只有业务与数据库都允许 NULL 时才使用 NullString。
type OptionalMetadata struct {
	Description sql.NullString `gorm:"column:description"`
}
```

## Scenario: 完整配置更新的字段所有权与零行语义

### 1. Scope / Trigger

- Store 接收完整配置对象更新已有记录，同时必须保留身份、控制状态、软删除状态、执行历史或 lease 字段时适用。

### 2. Signatures

```go
func (s *DBStore) Update(ctx context.Context, cfg *crontask.TaskConfig) error
```

### 3. Contracts

- 使用显式 `Select(...)` 白名单声明配置更新拥有的列；不要使用 `Select("*").Omit(...)`，否则模型新增字段会自动进入更新范围。
- 完整更新必须允许把可选配置清空，因此白名单更新要写入字符串零值、`sql.NullTime{Valid:false}` 和 `sql.NullString{Valid:false}`。
- 身份、状态、审计、软删除、执行历史和 lease 字段不得进入配置更新白名单。
- `next_run` 有独立所有权条件时单独更新；例如 CronJob 只有 `scheduled_time IS NULL` 时才允许配置更新覆盖 `next_run`。
- Trigger CronJob 的 `specified_times` / `excluded_times` 为可空 JSON 文本配置列：空列表应写 `NULL`，并且两列必须与完整 `rrule_str` 在同一显式白名单事务中替换或清空；在途 `scheduled_time` 任务更新失败时，这些列和 lease 均不得变化。
- 从数据库读取 SQL `NULL` 时，转换层将两个字段重建为空 slice，并通过同一 `CronJobExtra -> CronJobPb` 链路供 Get/List 回显；不得从 `rrule_str` 反推调用方原始列表。
- 配置白名单 UPDATE 与 `next_run` 条件 UPDATE 应放在同一事务，但不得合并所有权：前者确认配置写入，后者只在无在途 lease 时推进新计划。

### 4. Validation & Error Matrix

- 配置校验失败 -> 原校验错误，数据库不写入。
- 配置 UPDATE 报错 -> 返回数据库错误。
- `RowsAffected > 0` -> 配置更新成功。
- `RowsAffected == 0` -> 直接返回领域 `ErrUpdate`，不再发起 `Count` 查询扩大竞态窗口。
- lease 条件更新零行 -> 保留当前 lease，不得把整个配置更新判为失败。

### 5. Good/Base/Bad Cases

- Good: 更新任务名称并清空描述、排除日期；状态、`task_code`、执行历史和在途 lease 保持不变。
- Base: 配置 UPDATE 影响零行时返回 `ErrUpdate`，由调用边界决定映射为竞争或更新失败。
- Bad: `Select("*").Omit(...)` 随模型扩展意外清空新字段；零行后追加 `Count` 形成第二个竞态窗口；或把 `next_run` 放入普通配置 UPDATE 覆盖 lease。

### 6. Tests Required

- 断言白名单字段可更新，并且可选字段可以清空为 SQL `NULL` 或空字符串。
- 断言身份、状态、软删除字段、执行历史和 lease 字段不变。
- 断言目标不存在或配置更新零行返回 `ErrUpdate`。

### 7. Wrong vs Correct

#### Wrong

```go
result := db.Model(&Model{}).Select("*").Omit("id", "status").Updates(record)
if result.RowsAffected == 0 {
    return ErrNotFound
}
```

#### Correct

```go
result := db.Model(&Model{}).
    Select("name", "description", "start_time", "end_time").
    Updates(record)
if result.RowsAffected == 0 {
    return ErrUpdate
}
err := db.Model(&Model{}).
    Where("id = ? AND scheduled_time IS NULL", id).
    Update("next_run", nextRun).Error // zero rows means preserve in-flight lease
```
