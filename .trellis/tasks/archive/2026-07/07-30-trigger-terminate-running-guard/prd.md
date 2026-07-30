# Trigger 终止前校验运行中执行项

## Goal

避免 plan/batch 已发送 `FINISHED` 通知后，仍有执行中的 exec item 通过 callback 改变状态和 progress，同时保留 exec item 原始状态以识别未启动、延期、暂停和已完成的执行项。

## Requirements

- `TerminatePlan` 终止前检查该 plan 下是否存在 `plan_exec_item.status = StatusRunning`。
- `TerminatePlanBatch` 终止前检查该 batch 下是否存在 `plan_exec_item.status = StatusRunning`。
- 只要存在 `StatusRunning`，终止请求应返回业务错误，不更新父级状态，不发送 finished 通知。
- 不因终止 plan/batch 批量修改 exec item 状态；保留 `waiting/delayed/paused/completed/terminated` 的原始状态。
- 不检查 `last_result = ongoing` 作为唯一条件；刚下发但尚未首次 callback 的执行项同样由 `StatusRunning` 覆盖。
- 终止成功后必须阻止新的 exec item 被 cron 抢占为 running，否则 finished 通知后的 progress 仍可能变化。
- gRPC proto/server 接口保持兼容；错误通过现有领域错误与 gRPC 映射边界返回。

## Acceptance Criteria

- [ ] plan 下存在 running exec item 时，`TerminatePlan` 失败且父级状态、finished time 和通知均不变。
- [ ] batch 下存在 running exec item 时，`TerminatePlanBatch` 失败且 batch 状态、finished time 和通知均不变。
- [ ] 仅存在 waiting、delayed、paused、completed、terminated exec item 时，plan/batch 可成功终止。
- [ ] 成功终止不修改任何 exec item 状态，且终止通知之后 progress 不再因合法 callback 改变。
- [ ] cron 与终止并发时，不出现“父级已 terminated、exec 新变 running/下游已启动”的结果。
- [ ] 终止与 callback 并发时，不覆盖已有状态，不把竞争失败当作成功流水。
- [ ] 目标 Go 包测试、并发测试（按实际修改范围）、`go vet` 和 `git diff --check` 通过。

## Confirmed Facts

- `StatusRunning = 100` 表示已下发/执行中，等待业务回调；`StatusDelayed` 表示当前没有执行，仅等待再次调度。
- plan progress 当前按 `completed / total` 动态计算，finished 通知不携带 progress 快照。
- 终止 plan/batch 当前只更新父级并发送 finished 通知，不更新 exec item。
- cron 的 claim 和终止入口目前没有共享的明确互斥契约，需补充条件/事务保护。

## Out Of Scope

- 不新增下游 cancel/stop RPC，不承诺终止已下发业务的远端执行。
- 不改变 progress 公式，不新增 progress 快照字段。
- 不修改 proto 结构或生成代码。
- 不把 plan/batch 终止转换为批量终止 exec item。

## Notes

- 终止请求被拒绝时，建议错误包含 running 数量，便于调用方提示用户先等待或终止具体执行项。
- 并发保护必须由代码验证；单次普通 running 查询不足以保证检查与 cron claim 之间没有窗口。
