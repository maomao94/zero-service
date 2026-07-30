# Technical Design

## Boundary

修改 Trigger 的 plan/batch terminate Logic、Plan ExecItem model/store 状态条件和 cron claim/update 路径；不修改 gRPC contract。Logic 负责业务判定，model 负责 running 查询、条件更新和事务内数据访问。

## State Contract

- `StatusRunning` 是唯一禁止父级终止的 exec 状态。
- `waiting/delayed/paused` 不代表当前下游执行；父级终止后保留原状态，作为执行快照。
- `completed/terminated` 是终态。
- 父级成功终止后不得再产生新的 running item，且已有合法 callback 不应存在，因为终止前已无 running item。

## Data Flow

1. Terminate RPC 进入 Logic。
2. 在事务中检查作用域内 `exec_item.status = StatusRunning`。
3. 有 running 时回滚并返回领域业务错误；无 running 时仅更新 plan/batch，保持现有 finished 通知流程。
4. Cron claim 保持现有查询语义：候选查询通过 JOIN 筛选 enabled 的 plan/batch，随后使用 ExecItem 的 version/status/time 条件完成乐观 CAS；claim 后重新加载 exec、plan 和 batch，并在调用下游前补查父级状态。
5. Callback 的状态更新继续由 running 条件保护；若本次任务需要修复竞争误报，则统一检查 `RowsAffected`，零行不写成功流水、不继续聚合收尾。

## Concurrency Trade-off

终止入口通过带 `NOT EXISTS status=running` 的父级条件更新拒绝已有 running 项；cron claim 保持原有 ExecItem `version/status/next_trigger_time` 乐观 CAS，不引入 plan/plan_batch `FOR UPDATE`、额外父级子查询或补偿状态。

## Compatibility

成功终止的已有响应和通知结构保持不变。新增业务拒绝沿用现有 Logic 错误风格，不手写 gRPC status。exec item 状态和 progress 公式不变。

## Rollback

变更集中在手写 Logic/model/cron 与测试；可独立回退终止 guard，不改变数据结构和 proto。
