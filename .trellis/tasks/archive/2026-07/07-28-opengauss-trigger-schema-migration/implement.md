# Implementation Plan

## 1. Confirm Production Facts

- [ ] 确认 openGauss 产品名称、精确大版本/补丁版本、兼容模式和当前 schema。
- [ ] 导出四张表的真实 DDL、索引、约束、序列、触发器和依赖视图，不只依据仓库旧 SQL。
- [ ] 统计表大小、行数，并据此计算全量复制、建索引和校验所需停服时间。
- [ ] 确认所有写入方，不只停止 `app/trigger` 本身。
- [ ] 确认备份恢复流程和恢复耗时满足回滚窗口。

## 2. Maintain Versioned Migration Files

- [ ] 在仓库既有数据库部署位置确认后，新建不可变版本目录；建议文件拆分为 `00_precheck.sql`、`01_create_shadow.sql`、`02_copy.sql`、`03_validate.sql`、`04_cutover.sql`、`05_rollback.sql`、`06_cleanup_after_observation.sql`。
- [ ] 每个脚本写明目标 openGauss 版本、前置状态、预期状态、是否可重复执行、事务边界和失败处理。
- [ ] 不把迁移 DDL 混进 `AutoMigrate`，不通过反复修改 `model/sql/postgres.sql` 维护已上线版本。

## 3. Build And Rehearse

- [ ] 用脱敏生产快照建立同版本演练库。
- [ ] 创建四张影子表，字段契约逐项对照 `app/trigger/model/gormmodel/plan.go`。
- [ ] 编写显式列清单的 `INSERT ... SELECT`，禁止 `SELECT *`。
- [ ] 四张表全量复制，不按 `del_state`、状态或时间过滤。
- [ ] 验证数值 ID 文本化、`del_state -> is_deleted`、JSONB 文本化和 `rrule_str` 回填。
- [ ] 数据复制后创建索引和唯一约束并执行统计信息收集。
- [ ] 记录复制、建索引、校验、切换和回滚耗时。

## 4. Validation Gates

- [ ] 新旧表逐表行数一致。
- [ ] 新表 `id` 无 NULL、空串和重复。
- [ ] `plan_batch.plan_pk` 全部命中 `plan.id`，允许的历史缺失引用必须有书面例外清单。
- [ ] `plan_exec_item.plan_pk/batch_pk` 全部命中父表。
- [ ] `plan_exec_log.plan_pk/batch_pk/item_pk` 全部命中对应表，或按已批准的历史日志保留策略列出例外。
- [ ] `is_deleted` 只包含 0/1，且按值分组计数与旧 `del_state` 一致。
- [ ] 业务唯一键无冲突，关键状态/日期分组统计一致。
- [ ] 用当前 Trigger 二进制对影子表副本执行读取、创建和事务回滚测试。

## 5. Production Cutover

- [ ] 宣布维护窗口，停止所有写入方并确认无活跃写事务。
- [ ] 获取最终备份/快照和迁移水位。
- [ ] 执行最终复制与校验；任一 gate 失败立即停止，不切表。
- [ ] 执行短事务表名切换和对象名称整理。
- [ ] 启动新服务，验证列表/详情、创建计划、展开批次和执行项、claim、回调、日志及父级聚合。
- [ ] 监控数据库错误、慢 SQL、调度积压和重复执行。

## 6. Rollback And Cleanup

- [ ] 在演练库完整执行 `05_rollback.sql`。
- [ ] 观察期内限制或记录新版本写入，确保回滚时可补偿。
- [ ] 达到稳定观察期后，另行审批删除 legacy 表和旧序列；清理脚本不得与首次切换同时执行。

## Review Gates

- [x] 用户确认直接停服迁移，采用影子表全量复制和切换，不设计在线双写。
- [x] 用户确认全量保留四张表历史数据和日志。
- [x] 用户确认迁移范围固定为 `plan`、`plan_batch`、`plan_exec_item`、`plan_exec_log`，排除其他表。
- [ ] DBA 审核 openGauss 方言、锁和备份恢复步骤。
- [ ] 应用负责人审核字段回填和历史异常数据策略。
- [ ] 最终 SQL 在同版本演练库通过两次：一次正常迁移，一次失败注入后回滚。
