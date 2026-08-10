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

- 新模型优先组合 `AtomicModel`、`AtomicTenantModel`、`AtomicVersionModel` 等原子 mixin；不要嵌入多个重复定义主键/时间/软删字段的 mixin。
- `Version` 是显式乐观锁能力，不是所有模型默认字段；只有更新路径实际使用版本条件时才引入。
- `LegacyModel` 只服务现有旧表兼容，新表不复制历史字段布局。
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
- 查询无记录使用 `ErrNotFound`；条件更新 `RowsAffected == 0` 使用 `ErrNoRowsUpdate` 或领域专用竞争错误，不能把 CAS 未命中伪装成记录不存在。
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
