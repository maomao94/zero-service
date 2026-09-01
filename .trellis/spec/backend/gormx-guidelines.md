# GORM 与数据访问规范

> 修改 `common/gormx`、GORM model/store、分页、租户、乐观锁、Upsert、事务或时间空值时读取。

## 连接与配置

| 场景 | 正确做法 | 错误做法 |
|------|---------|---------|
| 代码内提供 driver.Config | `gormx.Open(cfg)` | — |
| 从 go-zero 配置加载 | `gormx.OpenWithConf(conf)` | 用零值 Config 调 OpenWithConf |
| 连接失败/方言不支持 | 返回错误 | 静默切换或使用 nil DB |
| 测试 | 用包内 SQLite helper | 方言 SQL 只用 SQLite 测试 |

依据：`common/gormx/open.go`、`common/gormx/README.md`

## 模型组合

| mixin | 提供字段 | 使用场景 |
|-------|---------|---------|
| `IDModel` / `StringIDModel` | 主键 | 所有新模型 |
| `TimeMixin` | created_at, updated_at | 需要时间戳 |
| `SoftDeleteMixin` | is_deleted | 软删除 |
| `VersionMixin` | version | 乐观锁（非默认） |
| `TenantMixin` | tenant_id | 多租户 |

**关键规则：**
- 不嵌入多个重复定义主键/时间/软删字段的 mixin
- `Version` 是显式乐观锁，只有更新路径实际使用版本条件时才引入
- `LegacyBaseModel` 只服务旧表兼容，新表不复制历史字段布局
- 已组合 `SoftDeleteMixin` 的 model，普通查询不重复手写 `is_deleted = 0`
- 使用 `Unscoped()`、原生 SQL、`Table(...)` 时才显式处理软删除

依据：`common/gormx/model.go`、`common/gormx/callbacks.go`

## 索引命名

| 类型 | 格式 | 示例 |
|------|------|------|
| 普通索引 | `idx_{table}_{suffix}` | `idx_oryx_record_created_at` |
| 唯一索引 | `uq_{table}_{suffix}` | `uq_dji_device_topo_gateway_sn` |
| 裸索引 | GORM 自动生成 | `idx_{table}_{column}` |

- 复合唯一索引的两个字段声明必须使用**同一个**索引名
- **AutoMigrate 不会删除旧索引**：改索引名后需手动 `DROP INDEX` 清理旧名
- 业务时间字段使用 `time.Time` + `type:timestamp`，不混用 `autoCreateTime` 和显式赋值

依据：`app/oryxserver/model/gormmodel/record.go`、`common/gormx/model_legacy.go`

## 查询与写入

| 职责 | 归属 | 说明 |
|------|------|------|
| SQL/GORM 表达式 | Store/Model | Logic 不拼接列名或 SQL |
| 动态排序/游标 | Store | 必须经白名单或 identifier 校验 |
| 分页参数 | `gormx.NewPageParams` | 不绕过私有字段 |
| 分页边界 | `gormx.QueryPage` | 调用方提供稳定排序 |
| 冲突写入 | `gormx.Upsert` | 明确 conflict/update columns |

依据：`common/gormx/pagination.go`、`common/gormx/upsert.go`

## 租户与并发所有权

| 概念 | 说明 |
|------|------|
| `TenantScope` | 缺失租户时不加过滤 |
| `TenantScopeStrict` | 缺失租户时追加恒假条件，返回空结果 |
| 查询无记录 | 使用 `model.ErrNotFound` |
| 条件更新零行 | 使用 `model.ErrNoRowsUpdate` 或领域竞争错误 |
| 并发状态更新 | 用事务/唯一约束/版本/CAS |
| 可空时间 | 用 `sql.NullTime` 或指针，禁止用远期时间伪装"无下次执行" |

- 条件更新必须检查 error 和 `RowsAffected`；幂等成功/目标不存在/竞争失败是不同契约
- 完成路径只能更新自己拥有的字段，不能用整行 `Save` 覆盖调度/配置字段

依据：`common/gormx/user_context.go`、`common/gormx/tenant_scope.go`、`app/trigger/internal/cronjob/db_store.go`

## 反模式

| 错误做法 | 正确做法 | 原因 |
|---------|---------|------|
| `db.Save(&wholeModel)` 更新高并发状态表 | 只更新自己拥有的字段 | 会覆盖其他字段 |
| 不检查 `RowsAffected` 就当成功 | 检查 error 和 RowsAffected | 竞争失败≠成功 |
| 租户过滤留给每个 Logic 手写 | 用 `TenantScope` | 统一过滤逻辑 |
| 普通查询重复手写 `is_deleted = 0` | 依赖 mixin scope | scope 所有权不清 |
| 字符串拼接动态列名/排序 | 用白名单或 identifier 校验 | SQL 注入风险 |
| 绕过 `gormx.Upsert` 封装 | 使用统一封装 | 方言 SQL 不一致 |

## 验证

- 测试创建、更新、未命中、重复、软删、租户缺失、版本冲突和事务回滚
- 对 claim/complete 或状态机更新断言 SQL 条件与 `RowsAffected`，必要时并发执行
- 运行目标 store/model 包测试；方言敏感逻辑不能只依赖 SQLite 通过

---

## Scenario: GaussDB 字符串空值

> 新增或修改字符串列的 `NOT NULL`、`DEFAULT`、GORM `default` tag，且服务支持 GaussDB 时适用。

**GaussDB A/ORA 兼容模式**：空字符串 `''` 视为 SQL `NULL`。

### 设计决策

| 列定义 | INSERT 输入 | GaussDB A/ORA 结果 | 设计结论 |
|--------|------------|-------------------|---------|
| `NOT NULL DEFAULT 'unknown'` | 省略列 | 保存 `unknown` | 默认值生效 |
| `NOT NULL DEFAULT 'unknown'` | 显式 `NULL` | 约束错误 | 默认值不生效 |
| `NOT NULL DEFAULT 'unknown'` | 显式 `''` | 约束错误 | 写入前拒绝或改非空哨兵 |
| nullable | `NullString{Valid:false}` | 保存 `NULL` | 适合业务可空字段 |
| nullable | `NullString{String:"", Valid:true}` | A/ORA 下保存 `NULL` | 不能区分空串与 NULL |

### Good/Bad 对比

```go
// ✗ 错误：NOT NULL + DEFAULT 保护不了显式空串
type Device struct {
    Type sql.NullString `gorm:"not null;default:'unknown'"`
}
device.Type = sql.NullString{String: "", Valid: true}
db.Create(&device) // 约束错误

// ✓ 正确：非空列使用普通 string，应用层显式提供非空值
type Device struct {
    Type string `gorm:"not null;default:'unknown'"`
}
device.Type = UnknownDeviceType
db.Create(&device)

// ✓ 正确：业务允许 NULL 时才用 NullString
type OptionalMetadata struct {
    Description sql.NullString `gorm:"column:description"`
}
```

依据：`app/djicloud/model/gormmodel/dji_device.go` 及 hook 测试

---

## Scenario: 完整配置更新的字段所有权

> Store 接收完整配置对象更新已有记录，同时必须保留身份/状态/lease 字段时适用。

### 核心规则

- 使用显式 `Select(...)` 白名单声明更新列；不用 `Select("*").Omit(...)`
- 白名单更新要写入字符串零值、`NullTime{Valid:false}`、`NullString{Valid:false}`
- 身份、状态、审计、软删除、执行历史和 lease 字段不得进入白名单
- `next_run` 有独立所有权条件时单独更新
- 配置白名单 UPDATE 与 `next_run` 条件 UPDATE 放同一事务，但不合并所有权

### Good/Bad 对比

```go
// ✗ 错误：Select("*").Omit(...) 随模型扩展意外清空新字段
result := db.Model(&Model{}).Select("*").Omit("id", "status").Updates(record)
if result.RowsAffected == 0 {
    return ErrNotFound // 错误：应该用 ErrUpdate
}

// ✓ 正确：显式白名单，零行返回 ErrUpdate
result := db.Model(&Model{}).
    Select("name", "description", "start_time", "end_time").
    Updates(record)
if result.RowsAffected == 0 {
    return ErrUpdate
}
// next_run 单独更新，零行保留 in-flight lease
err := db.Model(&Model{}).
    Where("id = ? AND scheduled_time IS NULL", id).
    Update("next_run", nextRun).Error
```

依据：`app/trigger/internal/cronjob/db_store.go`
