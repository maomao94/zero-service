# 数据库规范

> 涉及 MySQL、PostgreSQL、SQLite、TDengine、Redis、Kafka 或缓存时，先复用现有 model、client、cache、config 和 `common/` 封装。
> gormx 包专属约定（文件组织、调用签名、配置默认值、陷阱）见 [gormx-guidelines.md](./gormx-guidelines.md)。

## 基本原则

- 不在 Logic 中直接拼接连接串、账号、密码或环境参数。
- 数据库、Redis、消息队列和第三方客户端通过配置结构和 `ServiceContext` 注入。
- 先查现有 model、DAO、cache、client 和相邻服务实现，再新增持久化代码。
- 复杂数据流先画清：接口契约 → Logic → Model/SDK/Client → 存储或消息系统。
- 所有跨层调用传递 `context.Context`，便于超时、取消和链路追踪。

## Model 和生成脚本

项目在 `model/` 目录提供生成脚本（`sh genModel.sh`、`sh genPgModel.sh`、`sh genModelSql.sh`）。生成代码非必要不手改；需要调整字段、索引或表结构时优先改 SQL/schema 或生成配置。

## SQL 变更

- 独立 SQL 放入项目约定 SQL 目录，文件名建议 `yyyyMMdd-{需求号}-{简短说明}.sql`。
- SQL 内容要和 Trellis task 或 Backlog 条目关联，方便追踪上线影响。
- 不在 SQL、配置或日志中提交真实账号、密码、连接串、内网地址或对象存储配置。

## 查询和事务

- 简单 CRUD 优先复用生成 model 方法。
- 批量、事务和聚合查询先找同库同服务的既有写法。
- 需要事务时显式说明事务边界、提交条件和回滚条件。
- Redis/cache 更新要明确缓存 key、TTL、失效策略和数据一致性边界。

## Scenario: GaussDB PG 空串即 NULL 字段

### 1. Scope / Trigger

GaussDB PG 兼容模式可能把 `''` 按 `NULL` 处理。外部协议字段可缺省时不能用 `not null default:''` 表示"无值"。

### 2. Signatures

- DB column: `track_id varchar(64) NULL`，不要加 `NOT NULL`，不要依赖 `DEFAULT ''`。
- GORM field: `TrackId sql.NullString \`gorm:"column:track_id;type:varchar(64);index"\``。
- Write mapping: `sql.NullString{String: ext.TrackID, Valid: ext.TrackID != ""}`。
- Response mapping: protobuf `string track_id` 返回 `item.TrackId.String`，`NULL` 对外表现为 `""`。

### 3. Contracts

- `ext.track_id == ""` 或缺省 → DB 保存 `NULL`。
- `ext.track_id != ""` → DB 保存原字符串。
- API 不改契约，DB `NULL` 映射为 `""`。
- 既有 `NOT NULL` 列需执行 schema 变更允许 NULL，不能只改 GORM tag。

### 4. Validation & Error Matrix

- `NOT NULL` + 上游空/缺省 → GaussDB 写入 `NULL`，触发 `SQLSTATE 23502`。
- 同一事务首次写入失败后继续执行 → 触发 `SQLSTATE 25P02`（级联错误，非根因）。
- GORM string 扫描 DB `NULL` → 应使用 `sql.NullString`，防止扫描失败。
- `sql.NullString` 不要泄漏到 protobuf 响应，在 Logic/mapper 层显式取 `.String`。

### 5. Good/Base/Bad Cases

- Good: 可缺省字段 + GaussDB PG → `sql.NullString` + DB 允许 NULL + 响应层空字符串。
- Base: 业务必填字段保留 `string` + `not null`，handler 先校验空值并拒绝写入。
- Bad: 可缺省字段用 `string` + `not null;default:''`，期望空串绕过非空约束。

### 6. Tests Required

- 断言 `field.NotNull == false` 且 `field.HasDefaultValue == false`。
- 断言上游缺省字段不导致 upsert 报错，有值时能保存原字符串。
- 断言 DB `NULL` 映射到响应 `""`，非空值原样返回。
- 看到 `23502` 后的 `25P02` 时先查事务内第一条 SQL 错误。

## Scenario: GORM ErrRecordNotFound

### 1. Scope / Trigger

使用 GORM `First` / `Last` / `Take` 查询单条记录，或补充关联数据时，必须判断记录缺失是业务错误还是正常缺省。

### 2. Signatures

- Error sentinel: `gorm.ErrRecordNotFound`。
- Check pattern: `errors.Is(err, gorm.ErrRecordNotFound)`。
- `IgnoreRecordNotFoundError` 只控制 SQL trace 日志，不改变 `db.First(&v).Error` 返回值。

### 3. Contracts

- 必需主记录缺失 → 返回 `extproto.Code__1_02_RECORD_NOT_EXIST`。
- 可选关联数据缺失 → 返回空字段并继续组装，不打 error 日志。
- 真实数据库错误 → 保留原始错误，不能吞掉。

### 4. Good/Base/Bad

- Good: 关联快照查不到时跳过该字段，避免噪声日志。
- Base: 详情接口查不到主记录时返回结构化不存在错误。
- Bad: 全局忽略 `ErrRecordNotFound`，或完全不处理导致 error 日志刷屏。

### 5. Wrong vs Correct

```go
// Wrong — 可选关联缺失也触发业务 error
if err := db.Where("device_sn = ?", sn).First(&state).Error; err != nil {
    return err
}

// Correct — 区分可选缺失和真实错误
if err := db.Where("device_sn = ?", sn).First(&state).Error; err != nil {
    if errors.Is(err, gorm.ErrRecordNotFound) { return nil }
    return err
}
```

## Scenario: GORM 跨数据库 Upsert（禁止 clause.OnConflict）

### 1. Scope / Trigger

`clause.OnConflict` 在不同数据库驱动下生成不同 SQL（MySQL `ON DUPLICATE KEY UPDATE`、PostgreSQL `ON CONFLICT ... DO UPDATE`），GaussDB 驱动生成 MySQL 语法但 GORM 在 `Create` 时追加 `RETURNING "id"` 取自增主键，GaussDB 不支持两者组合，报 `SQLSTATE 0A000`。

### 2. Contracts

- 永远不用 `clause.OnConflict` 做 upsert。
- 统一使用 `FirstOrCreate` + `Assign` 模式，零方言 SQL，全数据库兼容。
- djicloud 有专门测试 `TestHookHandlersDoNotGenerateDialectUpsertSQL` 确保不生成 `ON CONFLICT`。
- 需要按列选择性更新时，用 `map[string]any` 构造 assign map，从 `updateColumns` 过滤。

### 3. Signatures

```go
// 标准模式：全部字段覆盖
c.Where("task_patrolled_id = ?", id).Assign(assign).FirstOrCreate(&task)

// 按列选择性更新
all := map[string]any{"task_state": task.TaskState, "task_progress": task.TaskProgress, ...}
assign := make(map[string]any, len(updateColumns))
for _, col := range updateColumns {
    if v, ok := all[col]; ok { assign[col] = v }
}
c.Where("task_patrolled_id = ?", id).Assign(assign).FirstOrCreate(&task)
```

### 4. Attrs vs Assign

| | `Attrs` | `Assign` |
|---|---|---|
| 记录已存在 | 不更新 | 更新为 Assign 值 |
| 记录不存在 | INSERT 带这些值 | INSERT 带这些值 |

首次写入用 `Attrs`（如默认值），每次覆盖用 `Assign`。

### 5. Good/Base/Bad Cases

- Good: `FirstOrCreate` + `Assign` → 零方言，全兼容。
- Base: 两步法（First → Create/Updates）→ 可工作但冗余。
- Bad: `clause.OnConflict` + `Create` → GaussDB 报 `SQLSTATE 0A000`。

### 6. 源文件

- `app/djicloud/internal/hooks/telemetry_up.go:56-72` — FirstOrCreate + Assign 示例
- `app/djicloud/internal/hooks/register_test.go:662-714` — 防 ON CONFLICT 测试
- `app/ispagent/internal/handler/task.go:359` — 按列选择性 Assign

---

## 时间格式化

proto/API 层时间字段使用 `string`，格式 `YYYY-MM-DD HH:mm:ss.SSSSSS`，UTC+8 时区。Go 层使用 `carbon.CreateFromStdTime(m.ReportedAt).ToDateTimeMicroString()` 转换。

需要落库或对外生成秒级业务时间时，先清掉亚秒精度，再格式化或写入 `time.Time`：

```go
now := tool.NowStartOfSecond()
text := now.ToDateTimeString()        // 2006-01-02 15:04:05
idSuffix := now.ToShortDateTimeString() // 20060102150405
stored := now.StdTime()
```

从已有 `time.Time` 派生业务时间时使用 `tool.CarbonFromTimeStartOfSecond(t)`；不要手写 `Format("YmdHis")`，carbon 已提供 `ToShortDateTimeString()`。

盒子端 SQLite 读取 `time.Time` 的 GORM tag 约定见 [`gormx-guidelines.md`](./gormx-guidelines.md#时间字段类型)：使用 `type:timestamp`，不要使用 `timestamp(6)` / `timestamp(0)`；秒级业务约束由写入前的 `tool.NowStartOfSecond()` 保证。

`timestamp without time zone` 列必须在驱动层明确扫描时区。数据库驱动可能把 `2026-07-15 13:44:21` 扫描成 `time.Date(..., time.UTC)`；如果业务按上海墙上时间调度，后续绝对时间比较会变成 `21:44:21 +0800`，小时任务可能错误跳到 `22:44:21`。

规则：PostgreSQL 使用 `gorm.io/driver/postgres` 内置的 `TimeZone` -> `pgtype.TimestampCodec.ScanLocation` 处理。GaussDB PG 兼容模式暂时禁用 `gaussdb://` DSN 和 `gorm.io/driver/gaussdb`，必须使用 `postgres://...&TimeZone=Asia/Shanghai` 连接，避免重新引入 timestamp 时区偏移。MySQL DSN 必须保留 `parseTime=true&loc=Asia%2FShanghai`；SQLite 测试和本地库按 `_loc=auto` 或应用层显式解析处理。

测试要求：`gaussdb://` 必须被拒绝；GaussDB 连接示例必须使用 `postgres://` 并返回 PostgreSQL Dialector。

源文件：
- `common/carbonx/carbonx.go` — 全局时区默认 `carbon.Shanghai`
- `common/gormx/driver.go` — 数据库 Dialector 和 timestamp 扫描时区配置
- `app/djicloud/djicloud.proto` — 所有 `reported_at` 字段
- `app/djicloud/internal/logic/helper.go` — 转换示例
- `common/tool/timeutil.go` — 秒级业务时间 helper

## 常见错误

- 新增功能前未搜索已有 model/client/cache，导致重复封装。
- 为单个 Logic 私有逻辑创建过度通用的公共 DAO。
- 手写生成模型文件，后续生成时被覆盖。
- 忽略 `context.Context`，导致超时、取消和链路追踪失效。
- 将真实数据库连接、远程地址或账号写入示例、日志、文档或提交信息。
