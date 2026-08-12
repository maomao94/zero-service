# CronJob 分组更新与日志设计

## Boundaries

本任务只扩展 Trigger CronJob 管理和 Eventstream 回执审计。公共 `crontask.Handler` 签名、单 RRULE 调度器、传统 Plan 日期展开和 Eventstream proto 均保持不变。

## Group Identity

`group_id` 是创建时确定的稳定业务分组身份。创建路径使用请求值，空值通过 `uuid.NewString()` 生成。Update 和 Submit 更新路径从已持久化任务的 `CronJobExtra` 取得原值，不重新生成。

稳定身份字段为 `task_code/group_id/dept_code/type`。Update proto 为兼容保留现有字段，但服务端将请求空值视为“未声明变更”，相同值视为一致，不同值返回参数错误。Store 的更新白名单不包含这些列。

Create/Update/Submit 响应新增 `group_id`，返回本次最终配置中的稳定值。新增 proto 字段使用未占用字段号，旧客户端可忽略。

## Configuration Ownership

配置更新拥有 `task_name/description/rrule_str/start_time/end_time/rule/exclude_dates/priority/payload/extra/lock_timeout/max_delay/ext1-ext5`。Enable/Disable 独占 `status`，Scheduler 独占 `scheduled_time/last_run/last_scheduled_run` 和在途 lease，Delete 独占软删除字段。

DB Store 的配置 UPDATE 增加 `scheduled_time IS NULL` 条件并检查 `RowsAffected`，保证检查与写入原子。更新普通配置成功后再写 `next_run`；由于第一步已确认无在途任务，第二步仍保留现有防御性条件。目标不存在或在途均返回 `ErrUpdate`，Logic 在预读已确认存在的前提下将其映射为业务状态冲突。

Submit 在按 task code 找到已有任务后复用同一更新约束；并发 Insert 冲突转更新时重新读取已有稳定身份。

## Schedule Range

保留 `CompileSchedule` 作为 Plan 单规则入口并继续使用 3 年范围。新增 CronJob 专用编译入口或为内部范围归一化传入明确上限，CronJob 使用 100 年。两条路径共用规则转换和序列化，避免复制 RRULE 语义。

默认范围不变：未传结束时间时仍补到开始年份年末。100 年仅是显式范围上限。Scheduler 和 Preview 继续使用 `After` 按需迭代，不调用 `All()`。

## Callback Message Logging

`CronExecLog` 新增 `Message` text 列。`handler.go` 增加私有调用函数，返回 `(message string, err error)`：

- gRPC transport error 或 nil response：message 为空。
- 非 nil response：先保存 `response.Message`，再根据 receipt 映射 nil、`ErrDeleteTask` 或普通 error。
- `NewEventHandler` 只返回 error，保持公共接口不变。
- `NewLoggingEventHandler` 同时取得 message/error，分别写入 `Message` 和 `ErrorMessage`。

Dev/Test AutoMigrate 可增加列；生产需要发布侧 DDL。回滚代码时新增 nullable/text 列可保留，不影响旧版本。

## Documentation

API 指南增加分组章节，明确同组 CronJob 独立调度且不会自动去重。固定节日示例分为：

- 公历固定日期：YEARLY + `month/day`，如元旦 1 月 1 日、国庆 10 月 1 日。
- 农历或每年浮动日期：通过 Trigger 节假日查询获得当年公历日期，再创建 `start_time == end_time` 的有界单次规则；不能把中秋写成固定公历 9 月 15 日。

## Risks And Rollback

- 收紧 Update 可能暴露依赖修改 `group_id/dept_code/type` 的旧调用方；通过明确参数错误避免静默行为变化。
- 在途更新从“接受但可能旧计划回写”改为拒绝，调用方需在当前执行完成后重试。
- 100 年规则仍由 `rrule-go.After` 按需计算；测试覆盖年度和分钟规则预览不展开全量数据。
- 回滚时可恢复旧响应和更新白名单；数据库新增 message 列无需删除。
