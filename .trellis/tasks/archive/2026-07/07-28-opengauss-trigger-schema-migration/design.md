# openGauss Trigger Schema Migration Design

## Scope And Target Contract

迁移一致性边界为 `plan`、`plan_batch`、`plan_exec_item`、`plan_exec_log`。四张表必须一起切换，不能只迁移前三张表，因为日志表保存 `plan_pk`、`batch_pk` 和 `item_pk`。

目标结构以 `app/trigger/model/gormmodel/plan.go` 为运行时契约，以 `model/sql/postgres.sql` 为参考而非可直接执行的生产迁移。关键变化：

- 四张表的 `id`：数值自增主键改为 `varchar(64)` 应用生成主键。
- 引用列 `plan_pk`、`batch_pk`、`item_pk`：`bigint` 改为 `varchar(64)`。
- `del_state` 改名为 `is_deleted`。
- `plan.recurrence_rule`：`jsonb` 改为文本，迁移时使用 JSON 文本表示。
- `plan.rrule_str`：新增文本列；历史记录没有可靠 RRULE Set 快照，默认回填空字符串，不能从旧 JSON 无依据伪造。
- `terminated_reason` 等结果文本按当前模型扩容到 2000。

历史数值 ID 使用十进制字符串表示，例如旧 `plan.id = 42` 迁移为新 `plan.id = '42'`。这能直接保持所有引用映射，无需生成映射表；新数据继续由 GORM `BeforeCreate` 生成 UUID。当前查询只将这些字段视为不透明字符串，没有要求历史 ID 必须符合 UUID 格式。

四张表全量迁移，包括软删除、已完成和已终止数据以及全部日志。复制 SQL 不按 `del_state` 或状态过滤；历史异常通过预检查报告，不在迁移中静默丢弃或修复。

## Recommended Migration Shape

已确认在停服维护窗口内执行影子表迁移：

1. 预检查和全量备份，不改业务表。
2. 创建带版本后缀的新表，例如 `plan_v2`、`plan_batch_v2`、`plan_exec_item_v2`、`plan_exec_log_v2`。
3. 按父子顺序复制数据，并在 `INSERT ... SELECT` 中显式转换主键和字段。
4. 对新旧表执行行数、唯一性、校验和/分组统计、孤儿引用和关键字段检查。
5. 停止 Trigger 及所有会写这些表的生产者，确认无活跃写事务。
6. 停服后执行最终全量复制；不引入双写、CDC 或基于 `update_time` 的增量追平。
7. 在短事务中将旧表改名为 `_legacy_<version>`，将 v2 表改为正式名，随后校验索引、约束和触发器。
8. 启动新服务，执行只读检查和受控冒烟测试。
9. 观察期内保留旧表和序列；稳定后另开清理变更。

影子表方案优于直接 `ALTER COLUMN TYPE`：它不会在原表上执行破坏性类型重写；结构和数据可在切换前检查；失败时旧表未被破坏；回滚只需切换表名。代价是需要约两倍表空间和足够覆盖全量复制、建索引、校验的维护窗口。

## Data Copy Contract

复制顺序：

```text
plan
  -> plan_batch
  -> plan_exec_item
  -> plan_exec_log
```

转换规则：

- `CAST(id AS varchar(64))` 写入新 `id`。
- 所有非零引用列同样按十进制文本转换。
- 旧脚本给引用列默认 `0`；值为 `0` 表示缺失引用时，目标应写空字符串并单独统计，而不是生成 `'0'` 假引用。是否允许存在这类行由预检查决定。
- `is_deleted = del_state`，只允许 0/1；异常值阻断迁移。
- `recurrence_rule` 使用数据库兼容的文本转换，并在演练库验证结果与应用 JSON 解析一致。
- `rrule_str = ''` 作为历史缺省值；新增记录由新版本正常写入。
- 可空字段保持 NULL，不用空字符串覆盖历史空值。
- `create_time/update_time/version/status/scan_flg` 原值保留。

## Constraints And Indexes

新表先创建主键和必要列，数据复制后再创建普通索引以减少复制成本。唯一约束可在复制前通过预检查确认，复制后创建并再次验证。

必须保留：

- `plan(plan_id)` 唯一。
- `plan_batch(batch_id)`、`plan_batch(batch_num)` 唯一。
- `plan_exec_item(exec_id)` 唯一。
- 所有现有查询索引，特别是 `plan_exec_item(is_deleted, next_trigger_time, status)`。

当前应用不声明数据库外键，迁移不擅自新增外键，以免改变删除和写入行为；引用完整性通过检查 SQL 验证。

## Trigger And Timestamp Handling

当前 GORM 自己维护 `create_time/update_time`，生产目标表不需要依赖旧的通用时间触发器。若保留数据库触发器，必须使用每表唯一、可重复部署的 DDL，并验证它不会覆盖迁移时保留的历史时间。

推荐新表迁移期间不挂 INSERT 时间触发器，避免复制时历史时间被改成当前时间。切换前按运行时实际需要决定是否创建；若 GORM 已完整维护时间，则删除该数据库级双重所有权。

## Cutover And Rollback

切换前必须记录四张正式表、索引、约束、序列和触发器的实际名称。表改名不会自动把索引/约束名称改成新版本约定，因此 v2 对象在创建时使用带版本后缀的临时名称，切换后再按 openGauss 支持语法规范化名称。

回滚条件包括服务启动失败、关键查询失败、父子聚合异常、调度扫描异常或新写入失败。回滚步骤：停止新服务，保留 v2 表现场，将正式表改回 v2 名，将 legacy 表恢复正式名，启动旧服务。新版本启动后产生的数据不会自动反写旧表，因此允许回滚的观察窗口内应限制业务写入，或明确接受人工补偿。

## Compatibility Risks

- openGauss 具体大版本对事务 DDL、`ALTER ... RENAME`、JSONB 转文本和索引重命名语法可能不同，所有语句必须在同版本演练库验证。
- 大表 `COUNT(*)`、聚合校验和建索引会消耗 IO；需限速或安排在维护窗口。
- 旧 SQL 同时用表级唯一约束和重复唯一索引定义 `modbus_code`，说明基线 SQL 不能直接当迁移脚本重复运行。
- `device_point_mapping` 和 `modbus_slave_config` 也存在字符串主键/软删除列变化，但用户已明确排除；未来如需迁移，应拆成独立脚本和独立回滚边界。

## Decision Pending

- openGauss 精确版本和生产真实 DDL 确认后，才能冻结最终方言 SQL。
