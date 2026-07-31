# Trigger Plan 表 openGauss 停服迁移指南

本文说明如何在停服维护窗口内，将 Trigger 的 Plan 相关表从旧版自增数值主键迁移到当前字符串主键结构。迁移采用“外部备份、旧表改名、新建目标表、`INSERT ... SELECT` 全量复制、校验、切换验证”的方式。

## 适用范围

本次只迁移以下四张表：

- `plan`
- `plan_batch`
- `plan_exec_item`
- `plan_exec_log`

四张表全量保留，包括软删除、已完成、已终止记录和全部执行日志。不迁移 `device_point_mapping`、`modbus_slave_config` 或其他表。

## 目标变化

| 位置 | 旧结构 | 新结构 | 迁移规则 |
| --- | --- | --- | --- |
| 四张表的 `id` | `BIGSERIAL` / `BIGINT` | `VARCHAR(64)` | 数值转十进制字符串，例如 `42` 转为 `'42'` |
| `plan_pk` | `BIGINT` | `VARCHAR(64)` | 与对应 `plan.id` 使用相同字符串值 |
| `batch_pk` | `BIGINT` | `VARCHAR(64)` | 与对应 `plan_batch.id` 使用相同字符串值 |
| `item_pk` | `BIGINT` | `VARCHAR(64)` | 与对应 `plan_exec_item.id` 使用相同字符串值 |
| 软删除列 | `del_state` | `is_deleted` | 原值复制，只允许 `0` 或 `1` |
| 新增规则列 | 旧表不存在 | 可空文本列 | 历史数据写 `NULL`，不回填，不影响迁移 |
| 部分结果字段 | `VARCHAR(256)` / `TEXT` | 最长 2000 | 原值复制 |

> 当前仓库 `app/trigger/model/gormmodel/plan.go` 中的新增规则快照列名是 `rrule_str`，没有名为 `rule` 的 Plan 列。执行前必须以待发布版本的真实 GORM 模型和目标 DDL 确认最终列名。如果最终列名是 `rule`，下文使用 `rule`；如果最终列名是 `rrule_str`，将 SQL 中的 `rule` 替换为 `rrule_str`。无论采用哪个列名，历史值均写 `NULL`。

> 数据库列允许 `NULL` 时，对应 Go 字段必须是 `sql.NullString`、`*string` 或其他可空类型。当前 `Plan.RRuleStr` 是普通 `string`，不能直接把数据库 `NULL` 作为已验证契约。如果本次新增的实际字段是 `rrule_str`，发布前应将模型改成可空类型；如果模型仍保持普通 `string`，历史数据必须改填空字符串而不是 `NULL`。本文后续 SQL 按用户确认的“新增 `rule` 字段可空”方案编写。

## 总体流程

1. 确认 openGauss 版本、兼容模式和线上真实表结构。
2. 停止 Trigger 以及所有可能写入四张表的程序。
3. 使用 `gs_dump` 创建数据库外部备份。
4. 执行迁移前数据检查并保存结果。
5. 将旧表改名为带日期后缀的 legacy 表。
6. 按当前 GORM 模型创建四张新表。
7. 使用显式字段清单执行 `INSERT ... SELECT`。
8. 校验行数、主键、父子引用、业务唯一键和软删除状态。
9. 启动新版本，执行受控冒烟验证。
10. 保留 legacy 表和旧序列，稳定观察后再单独清理。

## 重要约束

- 必须停服后再改名和复制，避免迁移期间继续产生写入。
- 原表改名只能用于快速回滚，不能替代 `gs_dump`。
- 不使用生产环境 GORM `AutoMigrate`。
- 不直接在原表上执行主键类型转换。
- `INSERT ... SELECT` 必须显式列出字段，禁止使用 `SELECT *`。
- 新增规则列写 `NULL`，不得根据旧数据猜测或伪造规则。
- 下列 SQL 是迁移模板，必须先在与生产相同版本的 openGauss 演练库验证。

## 1. 记录数据库信息

执行并保存输出：

```sql
SELECT version();
SHOW sql_compatibility;
SELECT current_schema();
```

同时导出四张表的真实 DDL、索引、唯一约束、触发器和序列。不能只根据仓库中的历史建表脚本判断线上结构。

## 2. 停止写入

停止以下组件：

- Trigger 服务所有实例。
- 直接写入四张表的管理脚本、定时任务或其他服务。
- 可能在发布过程中自动执行迁移的任务。

确认没有活跃写事务后再继续。停服时间必须覆盖备份后的最终检查、改名、复制、建索引、校验和应用冒烟测试。

## 3. 使用 gs_dump 备份

推荐生成自定义格式备份：

```bash
gs_dump \
  -h <host> \
  -p <port> \
  -U <user> \
  -F c \
  -f trigger_plan_before_string_id.dump \
  -t public.plan \
  -t public.plan_batch \
  -t public.plan_exec_item \
  -t public.plan_exec_log \
  <database>
```

可额外生成便于人工检查的纯 SQL 备份：

```bash
gs_dump \
  -h <host> \
  -p <port> \
  -U <user> \
  -F p \
  -f trigger_plan_before_string_id.sql \
  -t public.plan \
  -t public.plan_batch \
  -t public.plan_exec_item \
  -t public.plan_exec_log \
  <database>
```

不要把密码写进命令或文档。备份完成后，应确认文件非空，并在演练环境验证能够恢复。

## 4. 迁移前检查

### 4.1 保存行数

```sql
SELECT 'plan' AS table_name, COUNT(*) AS row_count FROM public.plan
UNION ALL
SELECT 'plan_batch', COUNT(*) FROM public.plan_batch
UNION ALL
SELECT 'plan_exec_item', COUNT(*) FROM public.plan_exec_item
UNION ALL
SELECT 'plan_exec_log', COUNT(*) FROM public.plan_exec_log;
```

### 4.2 检查软删除值

```sql
SELECT 'plan' AS table_name, del_state, COUNT(*) FROM public.plan GROUP BY del_state
UNION ALL
SELECT 'plan_batch', del_state, COUNT(*) FROM public.plan_batch GROUP BY del_state
UNION ALL
SELECT 'plan_exec_item', del_state, COUNT(*) FROM public.plan_exec_item GROUP BY del_state
UNION ALL
SELECT 'plan_exec_log', del_state, COUNT(*) FROM public.plan_exec_log GROUP BY del_state;
```

以下查询必须返回 `0`：

```sql
SELECT
    (SELECT COUNT(*) FROM public.plan WHERE del_state NOT IN (0, 1) OR del_state IS NULL)
  + (SELECT COUNT(*) FROM public.plan_batch WHERE del_state NOT IN (0, 1) OR del_state IS NULL)
  + (SELECT COUNT(*) FROM public.plan_exec_item WHERE del_state NOT IN (0, 1) OR del_state IS NULL)
  + (SELECT COUNT(*) FROM public.plan_exec_log WHERE del_state NOT IN (0, 1) OR del_state IS NULL)
    AS invalid_delete_state_count;
```

### 4.3 检查父子引用

```sql
SELECT COUNT(*) AS orphan_batch_plan_count
FROM public.plan_batch b
LEFT JOIN public.plan p ON p.id = b.plan_pk
WHERE b.plan_pk <> 0 AND p.id IS NULL;

SELECT COUNT(*) AS orphan_item_plan_count
FROM public.plan_exec_item i
LEFT JOIN public.plan p ON p.id = i.plan_pk
WHERE i.plan_pk <> 0 AND p.id IS NULL;

SELECT COUNT(*) AS orphan_item_batch_count
FROM public.plan_exec_item i
LEFT JOIN public.plan_batch b ON b.id = i.batch_pk
WHERE i.batch_pk <> 0 AND b.id IS NULL;

SELECT COUNT(*) AS orphan_log_plan_count
FROM public.plan_exec_log l
LEFT JOIN public.plan p ON p.id = l.plan_pk
WHERE l.plan_pk <> 0 AND p.id IS NULL;

SELECT COUNT(*) AS orphan_log_batch_count
FROM public.plan_exec_log l
LEFT JOIN public.plan_batch b ON b.id = l.batch_pk
WHERE l.batch_pk <> 0 AND b.id IS NULL;

SELECT COUNT(*) AS orphan_log_item_count
FROM public.plan_exec_log l
LEFT JOIN public.plan_exec_item i ON i.id = l.item_pk
WHERE l.item_pk <> 0 AND i.id IS NULL;
```

如果存在孤儿引用，应暂停迁移并记录具体行。迁移脚本不能静默删除或修复历史数据。

### 4.4 检查业务唯一键

```sql
SELECT plan_id, COUNT(*)
FROM public.plan
GROUP BY plan_id
HAVING COUNT(*) > 1;

SELECT batch_id, COUNT(*)
FROM public.plan_batch
GROUP BY batch_id
HAVING COUNT(*) > 1;

SELECT batch_num, COUNT(*)
FROM public.plan_batch
GROUP BY batch_num
HAVING COUNT(*) > 1;

SELECT exec_id, COUNT(*)
FROM public.plan_exec_item
GROUP BY exec_id
HAVING COUNT(*) > 1;
```

所有查询都应返回零行，否则新表唯一约束可能创建失败。

## 5. 将旧表改名

以下示例使用 `legacy_20260728` 后缀，实际执行时替换为真实发布日期：

```sql
ALTER TABLE public.plan RENAME TO plan_legacy_20260728;
ALTER TABLE public.plan_batch RENAME TO plan_batch_legacy_20260728;
ALTER TABLE public.plan_exec_item RENAME TO plan_exec_item_legacy_20260728;
ALTER TABLE public.plan_exec_log RENAME TO plan_exec_log_legacy_20260728;
```

改名后立即确认四张旧表都存在：

```sql
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name IN (
      'plan_legacy_20260728',
      'plan_batch_legacy_20260728',
      'plan_exec_item_legacy_20260728',
      'plan_exec_log_legacy_20260728'
  );
```

### 对象名称冲突

表改名通常不会同步修改索引、约束、序列和触发器的对象名。新建正式表时，原表遗留的同名对象可能造成名称冲突。

执行前必须列出相关对象：

```sql
SELECT schemaname, tablename, indexname
FROM pg_indexes
WHERE schemaname = 'public'
  AND tablename IN (
      'plan_legacy_20260728',
      'plan_batch_legacy_20260728',
      'plan_exec_item_legacy_20260728',
      'plan_exec_log_legacy_20260728'
  );
```

推荐把旧索引和约束改成带 legacy 后缀，或者给新对象使用 `_v2` 后缀。不要直接重复执行旧建表脚本并忽略“对象已存在”错误。

## 6. 创建新表

新表结构必须根据待发布版本的 GORM 模型生成并由人工复核。关键要求：

- `id VARCHAR(64) PRIMARY KEY`，不绑定自增序列。
- `plan_pk`、`batch_pk`、`item_pk` 使用 `VARCHAR(64)`。
- 使用 `is_deleted SMALLINT NOT NULL DEFAULT 0`。
- 新增规则列允许 `NULL`，不设置 `NOT NULL`，不要求默认值。
- 对应 GORM 字段必须使用 `sql.NullString` 或 `*string` 等可空类型；普通 `string` 模型应改填空字符串。
- `create_time`、`update_time` 由 GORM 维护，不应使用会覆盖历史时间的 INSERT 触发器。
- 唯一键和普通索引按当前查询契约创建。

> 创建新表前，应再次核对新增规则列究竟是 `rule` 还是当前代码中的 `rrule_str`。列名必须与待发布 GORM tag 完全一致。

## 7. 全量复制数据

四张表按以下顺序复制：

```text
plan -> plan_batch -> plan_exec_item -> plan_exec_log
```

下列 SQL 假设目标新增列名是 `rule`。如果发布版本使用 `rrule_str`，只替换目标列名，不改变 `NULL` 回填规则。

### 7.1 复制 plan

```sql
INSERT INTO public.plan (
    id,
    create_time,
    update_time,
    delete_time,
    is_deleted,
    version,
    create_user,
    update_user,
    dept_code,
    plan_id,
    plan_name,
    type,
    group_id,
    recurrence_rule,
    start_time,
    end_time,
    status,
    scan_flg,
    terminated_reason,
    paused_time,
    paused_reason,
    finished_time,
    description,
    ext_1,
    ext_2,
    ext_3,
    ext_4,
    ext_5
)
SELECT
    CAST(id AS VARCHAR(64)),
    create_time,
    update_time,
    delete_time,
    del_state,
    version,
    create_user,
    update_user,
    dept_code,
    plan_id,
    plan_name,
    type,
    group_id,
    CAST(recurrence_rule AS TEXT),
    start_time,
    end_time,
    status,
    scan_flg,
    terminated_reason,
    paused_time,
    paused_reason,
    finished_time,
    description,
    ext_1,
    ext_2,
    ext_3,
    ext_4,
    ext_5
FROM public.plan_legacy_20260728;
```

### 7.2 复制 plan_batch

```sql
INSERT INTO public.plan_batch (
    id,
    create_time,
    update_time,
    delete_time,
    is_deleted,
    version,
    create_user,
    update_user,
    dept_code,
    plan_pk,
    plan_id,
    batch_id,
    batch_name,
    batch_num,
    status,
    scan_flg,
    plan_trigger_time,
    terminated_reason,
    paused_time,
    paused_reason,
    finished_time,
    ext_1,
    ext_2,
    ext_3,
    ext_4,
    ext_5
)
SELECT
    CAST(id AS VARCHAR(64)),
    create_time,
    update_time,
    delete_time,
    del_state,
    version,
    create_user,
    update_user,
    dept_code,
    CASE WHEN plan_pk = 0 THEN '' ELSE CAST(plan_pk AS VARCHAR(64)) END,
    plan_id,
    batch_id,
    batch_name,
    batch_num,
    status,
    scan_flg,
    plan_trigger_time,
    terminated_reason,
    paused_time,
    paused_reason,
    finished_time,
    ext_1,
    ext_2,
    ext_3,
    ext_4,
    ext_5
FROM public.plan_batch_legacy_20260728;
```

### 7.3 复制 plan_exec_item

```sql
INSERT INTO public.plan_exec_item (
    id,
    create_time,
    update_time,
    delete_time,
    is_deleted,
    version,
    create_user,
    update_user,
    dept_code,
    plan_pk,
    plan_id,
    batch_pk,
    batch_id,
    exec_id,
    item_id,
    item_type,
    item_name,
    item_row_id,
    point_id,
    payload,
    request_timeout,
    plan_trigger_time,
    next_trigger_time,
    last_trigger_time,
    trigger_count,
    status,
    last_result,
    last_message,
    last_reason,
    terminated_reason,
    paused_time,
    paused_reason,
    ext_1,
    ext_2,
    ext_3,
    ext_4,
    ext_5
)
SELECT
    CAST(id AS VARCHAR(64)),
    create_time,
    update_time,
    delete_time,
    del_state,
    version,
    create_user,
    update_user,
    dept_code,
    CASE WHEN plan_pk = 0 THEN '' ELSE CAST(plan_pk AS VARCHAR(64)) END,
    plan_id,
    CASE WHEN batch_pk = 0 THEN '' ELSE CAST(batch_pk AS VARCHAR(64)) END,
    batch_id,
    exec_id,
    item_id,
    item_type,
    item_name,
    item_row_id,
    point_id,
    payload,
    request_timeout,
    plan_trigger_time,
    next_trigger_time,
    last_trigger_time,
    trigger_count,
    status,
    last_result,
    last_message,
    last_reason,
    terminated_reason,
    paused_time,
    paused_reason,
    ext_1,
    ext_2,
    ext_3,
    ext_4,
    ext_5
FROM public.plan_exec_item_legacy_20260728;
```

### 7.4 复制 plan_exec_log

```sql
INSERT INTO public.plan_exec_log (
    id,
    create_time,
    update_time,
    delete_time,
    is_deleted,
    version,
    create_user,
    update_user,
    dept_code,
    plan_pk,
    plan_id,
    plan_name,
    batch_pk,
    batch_id,
    item_pk,
    exec_id,
    item_id,
    item_type,
    item_name,
    point_id,
    trigger_time,
    trace_id,
    exec_result,
    message,
    reason
)
SELECT
    CAST(id AS VARCHAR(64)),
    create_time,
    update_time,
    delete_time,
    del_state,
    version,
    create_user,
    update_user,
    dept_code,
    CASE WHEN plan_pk = 0 THEN '' ELSE CAST(plan_pk AS VARCHAR(64)) END,
    plan_id,
    plan_name,
    CASE WHEN batch_pk = 0 THEN '' ELSE CAST(batch_pk AS VARCHAR(64)) END,
    batch_id,
    CASE WHEN item_pk = 0 THEN '' ELSE CAST(item_pk AS VARCHAR(64)) END,
    exec_id,
    item_id,
    item_type,
    item_name,
    point_id,
    trigger_time,
    trace_id,
    exec_result,
    message,
    reason
FROM public.plan_exec_log_legacy_20260728;
```

## 8. 迁移后校验

### 8.1 行数一致

以下四条查询的 `old_count` 和 `new_count` 必须分别相等：

```sql
SELECT
    (SELECT COUNT(*) FROM public.plan_legacy_20260728) AS old_count,
    (SELECT COUNT(*) FROM public.plan) AS new_count;

SELECT
    (SELECT COUNT(*) FROM public.plan_batch_legacy_20260728) AS old_count,
    (SELECT COUNT(*) FROM public.plan_batch) AS new_count;

SELECT
    (SELECT COUNT(*) FROM public.plan_exec_item_legacy_20260728) AS old_count,
    (SELECT COUNT(*) FROM public.plan_exec_item) AS new_count;

SELECT
    (SELECT COUNT(*) FROM public.plan_exec_log_legacy_20260728) AS old_count,
    (SELECT COUNT(*) FROM public.plan_exec_log) AS new_count;
```

### 8.2 主键有效

```sql
SELECT 'plan' AS table_name, COUNT(*) AS invalid_count
FROM public.plan WHERE id IS NULL OR id = ''
UNION ALL
SELECT 'plan_batch', COUNT(*) FROM public.plan_batch WHERE id IS NULL OR id = ''
UNION ALL
SELECT 'plan_exec_item', COUNT(*) FROM public.plan_exec_item WHERE id IS NULL OR id = ''
UNION ALL
SELECT 'plan_exec_log', COUNT(*) FROM public.plan_exec_log WHERE id IS NULL OR id = '';
```

每张表的 `invalid_count` 必须为 `0`。主键约束会同时保证没有重复 ID。

### 8.3 新增规则列为空

如果目标列名是 `rule`：

```sql
SELECT COUNT(*) AS unexpected_rule_count
FROM public.plan
WHERE rule IS NOT NULL;
```

如果目标列名是 `rrule_str`：

```sql
SELECT COUNT(*) AS unexpected_rrule_count
FROM public.plan
WHERE rrule_str IS NOT NULL;
```

迁移后应返回 `0`。新版本应用创建的记录可在此后正常写入该列。

### 8.4 父子引用完整

```sql
SELECT COUNT(*) AS orphan_batch_plan_count
FROM public.plan_batch b
LEFT JOIN public.plan p ON p.id = b.plan_pk
WHERE b.plan_pk <> '' AND p.id IS NULL;

SELECT COUNT(*) AS orphan_item_plan_count
FROM public.plan_exec_item i
LEFT JOIN public.plan p ON p.id = i.plan_pk
WHERE i.plan_pk <> '' AND p.id IS NULL;

SELECT COUNT(*) AS orphan_item_batch_count
FROM public.plan_exec_item i
LEFT JOIN public.plan_batch b ON b.id = i.batch_pk
WHERE i.batch_pk <> '' AND b.id IS NULL;

SELECT COUNT(*) AS orphan_log_plan_count
FROM public.plan_exec_log l
LEFT JOIN public.plan p ON p.id = l.plan_pk
WHERE l.plan_pk <> '' AND p.id IS NULL;

SELECT COUNT(*) AS orphan_log_batch_count
FROM public.plan_exec_log l
LEFT JOIN public.plan_batch b ON b.id = l.batch_pk
WHERE l.batch_pk <> '' AND b.id IS NULL;

SELECT COUNT(*) AS orphan_log_item_count
FROM public.plan_exec_log l
LEFT JOIN public.plan_exec_item i ON i.id = l.item_pk
WHERE l.item_pk <> '' AND i.id IS NULL;
```

如果迁移前没有已批准的历史例外，以上结果必须全部为 `0`。

### 8.5 软删除分布一致

```sql
SELECT del_state, COUNT(*)
FROM public.plan_legacy_20260728
GROUP BY del_state
ORDER BY del_state;

SELECT is_deleted, COUNT(*)
FROM public.plan
GROUP BY is_deleted
ORDER BY is_deleted;
```

对其他三张表执行相同检查。每个状态的数量必须一致。

## 9. 应用冒烟验证

数据库校验全部通过后启动新版本，依次验证：

1. 查询一条历史 Plan，并正常加载其 Batch、ExecItem 和 ExecLog。
2. 新建一个受控测试 Plan，确认新主键是非空字符串。
3. 确认新 Batch 和 ExecItem 的 `plan_pk`、`batch_pk` 与父表字符串主键一致。
4. 确认新增规则列可为 `NULL`，对应 GORM 字段也是可空类型，历史 Plan 查询不会报错。
5. 验证执行项扫描使用 `is_deleted`，没有查询旧 `del_state`。
6. 验证回调可以写入 `plan_exec_log` 并聚合更新 Batch 和 Plan 状态。

出现以下情况时立即停止新服务并回滚：

- GORM 报列不存在或类型不匹配。
- 历史数据无法读取。
- 父子查询或状态聚合异常。
- 执行项无法扫描、重复扫描或异常积压。
- 新数据无法写入或主键为空。

## 10. 回滚

回滚前先停止新服务，避免继续写入新表。将新表改名保留现场，再恢复旧表名：

```sql
ALTER TABLE public.plan RENAME TO plan_failed_20260728;
ALTER TABLE public.plan_batch RENAME TO plan_batch_failed_20260728;
ALTER TABLE public.plan_exec_item RENAME TO plan_exec_item_failed_20260728;
ALTER TABLE public.plan_exec_log RENAME TO plan_exec_log_failed_20260728;

ALTER TABLE public.plan_legacy_20260728 RENAME TO plan;
ALTER TABLE public.plan_batch_legacy_20260728 RENAME TO plan_batch;
ALTER TABLE public.plan_exec_item_legacy_20260728 RENAME TO plan_exec_item;
ALTER TABLE public.plan_exec_log_legacy_20260728 RENAME TO plan_exec_log;
```

恢复表名后检查旧索引、约束、触发器和序列仍绑定到恢复后的旧表，再启动旧版本服务。

新版本启动后产生的数据不会自动写回旧表。应尽量在受控冒烟阶段完成是否回滚的判断；如果已产生正式业务数据，回滚前必须另行制定补偿方案。

## 11. 稳定后清理

不要在首次迁移当天删除以下对象：

- 四张 `*_legacy_20260728` 表。
- 旧 `BIGSERIAL` 使用的序列。
- `gs_dump` 备份。

经过约定观察期并确认不再回滚后，再提交独立清理审批。清理前重新做一次归档备份。

## 执行检查表

- [ ] 已确认生产 openGauss 精确版本和兼容模式。
- [ ] 已导出线上真实 DDL、索引、约束、触发器和序列。
- [ ] 已确认新增规则列实际名称，并保证数据库列和 GORM 字段都允许 `NULL`。
- [ ] 已停止所有四表写入方。
- [ ] `gs_dump` 已完成并验证可恢复。
- [ ] 迁移前行数、唯一键、软删除和孤儿引用检查通过。
- [ ] legacy 表改名完成，相关对象名称无冲突。
- [ ] 新表结构与待发布 GORM 模型一致。
- [ ] 四张表按顺序全量复制完成。
- [ ] 新增规则列历史值全部为 `NULL`。
- [ ] 行数、主键、父子引用和软删除分布校验通过。
- [ ] 新版本冒烟验证通过。
- [ ] legacy 表、旧序列和外部备份仍保留。
