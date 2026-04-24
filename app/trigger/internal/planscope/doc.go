// Package planscope 为 trigger 计划域（计划 / 批次 / 执行项）日志提供统一结构化字段与约定。
//
// 字段约定：
//   - entry：请求入口，rpc | cron | callback。
//   - tag：子场景，如 plan、plan-batch、plan-exec、plan-trigger、plan-callback、cron-lock。
//   - ref：业务串联检索键——仅计划时为 plan_id；批次为 plan_id/batch_id；执行项为 plan_id/batch_id/exec_id。
//   - notify_event：仅在调用 StreamEvent.NotifyPlanEvent 前打印，取值见 NotifyEvent* 常量。
//
// 消息前缀约定（message 首段，便于检索与对齐语义）：
//   - 「RPC …」：gRPC 同步接口内、事务已提交类成功日志。
//   - 「下游通知：」：调用 NotifyPlanEvent 之前的信息类日志。
//   - 「定时扫表：」：Cron 抢占/锁定/扫表路径。
//   - 「计划执行回调：」/「下游返回：」：Cron 内 HandlerPlanTaskEvent 回调链路。
//
// 实现侧应优先使用本包 Scope.Logger，避免裸 logx 丢失 entry/tag/ref。
package planscope
