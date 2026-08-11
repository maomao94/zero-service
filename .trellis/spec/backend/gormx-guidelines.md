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
