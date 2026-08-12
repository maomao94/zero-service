# CronJob Callback Contract Design

## Contract Boundaries

Trigger 是 CronJob 配置和回调数据的来源，StreamEvent PB 是下游边界。回调只传递下游业务处理所需的关键字段，不传递调度器内部运行状态。

`TaskConfig.Extra` 是 Trigger 内部为通用 Scheduler 重建业务模型列的运行时载体，不属于下游业务契约。管理视图和回调均不传递该 JSON；handler 通过 `ParseExtra` 读取业务字段（Type、GroupId、Description、Ext1-5、DeptCode）后以扁平字段发送。

## Data Flow

1. Create/Submit 接收最长 128 rune 的 TaskCode。
2. CronJob 模型将 TaskCode 保存到 128 长度列，并继续以唯一索引识别业务任务。
3. Handler 将 TaskConfig 与 Extra 解析结果映射为扁平 `HandleCronJobEventReq`（关键业务字段 + 本次 `scheduled_time`），调用 Eventstream。
4. StreamEvent 的业务回执和 Trigger 的完成/重试处理保持不变。

## Compatibility

- 用户已明确允许覆盖 Proto：删除 `extra` 字段，字段号顺延连续排列，不保留字段号兼容。
- 仍保持 RPC 方法名和回执响应不变，调用方只需迁移请求结构。
- 旧客户端和短 TaskCode 行为不变。
- 数据库扩列是向后兼容 schema 变更，但生产发布必须先执行 DDL。

## Validation Trade-Offs

- TaskCode 长度校验只保留在 Trigger 请求 PB（`max_len: 128`，按 rune 计数，与现有错误信息一致），回调 PB 不引入 PGV validation，避免下游契约携带与业务无关的重复规则。
- `scheduled_time` 的完整日期合法性由 Trigger 的 `formatTime` 来源保证。

## Proto Ownership

- `HandleCronJobEventReq` 是扁平请求，不引入嵌套 CronJob 模型，无需跨服务共享 PB，也不产生 proto 包循环。
- 字段序号从 1 开始连续排列；Go 字段名由 proto 字段名派生，序号调整不影响 Trigger 调用方代码。

## Rollback

- Proto 字段号、生成物与回调字段映射作为一个单元回滚。
- 数据库若已扩列，无需缩回 64，旧代码仍可使用更宽列。
