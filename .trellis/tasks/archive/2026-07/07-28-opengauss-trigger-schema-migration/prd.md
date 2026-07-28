# 规划 openGauss 生产表迁移

> 状态：用户已取消迁移。本任务未进入实施阶段，未生成生产迁移 SQL，也未修改数据库。

## Goal

为已上线的华为 openGauss 数据库制定可审计、可校验、可回滚的 Trigger 计划表增量迁移方案，使现有数据可被当前 `app/trigger` GORM 模型直接读取和继续调度。

## Confirmed Facts

- 生产旧表使用 `BIGSERIAL/BIGINT` 主键和关联列；当前 Trigger 模型使用应用生成的 `VARCHAR(64)` 字符串主键。
- 迁移对象至少包含 `plan`、`plan_batch`、`plan_exec_item`；`plan_exec_log` 保存对三者主键的引用，也必须纳入同一迁移一致性边界。
- 旧软删除列名是 `del_state`，当前 `gormx.LegacyStringBaseModel` 使用 `is_deleted`。
- 当前模型新增 `plan.rrule_str`，并将 `recurrence_rule` 从 `JSONB` 映射为 `TEXT`。
- 当前模型要求 `plan_pk`、`batch_pk`、`item_pk` 与对应字符串主键一致。
- Trigger 只在 dev/test 执行 `AutoMigrate`；生产迁移不能依赖 GORM 自动改表。
- GaussDB/openGauss 在项目中通过 PostgreSQL 兼容 DSN 使用，但迁移 SQL 仍需规避未经目标版本验证的 PostgreSQL 专属扩展。
- 用户确认生产发布时直接停服迁移，不要求在线双写或零停机切换。
- 用户确认四张计划表全量迁移，包括软删除、已完成、已终止记录和全部执行日志，不按状态筛选。
- 用户确认本次只迁移 Plan 相关表，排除 `device_point_mapping` 和 `modbus_slave_config`。
- 用户确认 `plan_exec_log` 与前三张 Plan 表一起迁移，四张表构成不可拆分的一致性边界。

## Requirements

- 迁移脚本必须显式版本化，和建库基线 SQL 分离，禁止修改历史迁移文件后期待生产库自动同步。
- 迁移前必须检查表结构、约束/索引、触发器、行数、孤儿引用、唯一键冲突、超长文本和磁盘空间。
- 主键转换必须保持 `plan.id -> plan_batch.plan_pk -> plan_exec_item.plan_pk/batch_pk -> plan_exec_log.plan_pk/batch_pk/item_pk` 的引用关系。
- 旧数值主键可按十进制文本无损迁移；新写入继续由 GORM 生成 UUID，不要求为历史行重新生成随机 UUID。
- 迁移必须保留旧表和旧序列，直到新版本稳定并完成人工确认；回滚优先通过表名切回，不做字符串到自增主键的逆向变更。
- 数据复制不得添加状态或软删除过滤条件；任何历史孤儿引用都必须在迁移前报告并由用户决定阻断或按例外保留。
- 必须提供迁移前检查、结构创建、数据复制、校验、切换、回滚和稳定后清理的独立步骤。
- 切换期间必须停止旧服务和所有其他写入方，确认没有活跃写事务后再复制和切表。
- 所有 DDL/DML 先在与生产相同 openGauss 大版本的演练库执行并记录耗时、锁行为和执行计划。

## Acceptance Criteria

- [ ] `prd.md` 明确迁移范围、运行约束和可测试验收标准。
- [ ] `design.md` 给出 openGauss 兼容的分阶段迁移、主键映射、切换和回滚设计。
- [ ] `implement.md` 给出脚本维护位置、执行顺序、验证 SQL、停机点和回滚点。
- [ ] 迁移后四张表行数与旧表一致，所有主键非空且唯一，父子引用不存在孤儿。
- [ ] `del_state` 数据完整映射到 `is_deleted`，新增字段有明确回填值。
- [ ] 当前 GORM 模型可读取历史记录，并能新增 Plan/Batch/ExecItem/Log；新记录使用字符串 UUID 主键。
- [ ] 核心扫描索引使用 `is_deleted, next_trigger_time, status`，业务唯一约束和查询索引保持有效。
- [ ] 回滚演练能够在约定时间内停止新服务、切回旧表并启动旧版本服务。

## Out Of Scope

- 本规划阶段不连接或修改生产数据库。
- 不使用生产环境 `AutoMigrate`。
- 不在未确认 openGauss 精确版本和生产真实 DDL 前生成可直接在生产执行的一键脚本。
- `device_point_mapping`、`modbus_slave_config` 及其他非 Plan 表结构迁移。

## Open Question

- 生产 openGauss 的精确版本、兼容模式和真实 DDL 尚待提供。
