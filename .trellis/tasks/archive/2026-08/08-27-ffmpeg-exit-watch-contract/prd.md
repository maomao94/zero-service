# 优化 FFmpeg 退出与输出契约

## Goal

进一步简化 `ffmpegx.Manager` 的重复 ID、退出结果和 stdout 观察契约，并通过 Go context 为 relay 增加可选最大运行时长。

## Confirmed Facts

- `ffmpeg-go v0.5.0` 的 `OutputContext` 保存传入 context，`Compile()` 使用 `exec.CommandContext`，因此 context cancel/deadline 会终止 FFmpeg 进程。
- Go `os/exec.Cmd.Wait()` 在 context 取消时不保证返回 context error：若 `Process.Wait` 已产生错误，通常优先返回 `*exec.ExitError`；只有进程错误为空时才采用 context watcher 的 `context.Canceled` 或 `context.DeadlineExceeded`。
- 因此不能仅通过 `errors.Is(waitErr, context.Canceled/DeadlineExceeded)` 稳定判断退出原因，必须同时观察 process context 的 `Err()`。
- 当前 relay 已在 `PullRegistry` 使用 `context.WithoutCancel(requestCtx)` 脱离 gRPC 生命周期，适合在其上按业务参数增加 `context.WithTimeout`。
- 当前 `WatchProgress` 聚合完整 FFmpeg progress block；用户要求改成通用 stdout 逐条同步回调，由业务方自行识别 key/value、记录状态并执行续租。

## Requirements

- `Manager.Start` 遇到已存在的 process ID 时不再主动取消旧进程，直接返回可通过 `errors.Is` 判断的 `ErrProcessExists`。
- 进程退出回调接收结构化结果，至少包含 `ID`、`WaitErr` 和 `ContextErr`；callback 同步执行。
- exit callback 对自然退出、Stop、cancel 和 timeout 均执行；`ffmpegx` 只报告结果，不筛选业务事件。业务方可在主动 Stop 前先删除自身 metadata，使回调按业务身份校验自然跳过。
- `WaitErr` 保留 `Cmd.Wait()` 原始错误；`ContextErr` 取 process context 的 `Err()`，用于稳定区分主动 cancel 和 deadline timeout。
- Manager 使用 `logx.WithContext(processCtx)` 记录启动、停止和退出日志，保留调用链字段。
- 将 progress 专属 watcher 改为通用 stdout watcher：scanner 每读取一行就同步回调，不保存 key/value，不判断 `progress=continue/end`。
- stdout watcher 只负责消费和回调；FFmpeg progress key/value 聚合、续租、状态记录属于 relay 或其他业务层。
- 只有显式配置 stdout handler 时 Manager 才创建 `StdoutPipe`；无 handler 时不消费 stdout。
- `ffmpegx` 不拥有补拉、续租、Redis 清理或最大运行时长策略；它只执行 Start/Stop、消费可选 stdout，并同步报告进程退出结果。
- relay 支持最大运行时长秒数：`0` 表示不限制，正数由 relay 业务层通过 context deadline 控制，不添加 FFmpeg duration 参数。
- 最大运行时长由 relay 转换为绝对截止时间并随 `RelayState` 持久化，Reconcile 根据剩余时间创建新的 deadline context，不能重置完整时长。
- 最大运行时长使用秒作为 proto 边界单位，进入 Go 业务层后转换为 `time.Duration`；持久化保存绝对截止时间。
- deadline 到期、手动 cancel 和流异常后的 Redis/lease/补拉决策全部由 relay 根据退出结果和持久化状态决定；Manager 不执行任何业务动作。
- relay 不直接调用 `ffmpeg-go.WithTimeout`：其内部丢弃 cancel，且相对时长会在每次 Reconcile 时重置。本项目由 relay 创建可取消的 deadline context，再交给 `ffmpegx.Manager` 和 `OutputContext/Compile`。

## Out of Scope

- FFmpeg 参数级时长控制。
- Manager 自动解析 stdout key/value 或维护 progress 状态。
- watchdog、卡流检测、自动重启策略重构。
- 在 `ffmpegx` 中实现 Redis、lease、续租或补拉策略。

## Acceptance Criteria

- [ ] 重复 ID 返回 `ErrProcessExists`，旧进程保持运行。
- [ ] exit callback 能可靠区分自然退出、主动 cancel 和 deadline exceeded，即使 `WaitErr` 是 `*exec.ExitError`。
- [ ] 所有已启动进程退出都同步执行 exit callback，且 callback 完成后才关闭 `done`。
- [ ] Manager 日志使用 process context。
- [ ] stdout watcher 按行同步回调且不保存业务状态。
- [ ] relay 业务自行解析 progress 行并仅在完整 progress 报告边界续租。
- [ ] 最大运行时长为 0 时无限制，正数到期后 exit result 的 `ContextErr` 为 `DeadlineExceeded`。
- [ ] Reconcile 使用持久化绝对截止时间计算剩余时长，不重置完整时限。
- [ ] relay 单测覆盖手动 cancel、deadline 到期和流异常三类退出后的业务决策；`ffmpegx` 测试只断言退出结果，不断言补拉行为。
- [ ] 相关单测、race、build、target vet、全仓 test 和 diff 检查通过。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
