# Implement — oryxserver relay distributed simplify refactor

## Ordered checklist

1. 重写 `internal/relay/state.go`：
   - `RelayState{RelayID, Source, Target}`；`Store`（Get/Save/DeleteState、HasLease、TryClaim=SetnxEx、Renew=GET+Expire、Release=Del）。
   - 常量 `RelayQueue`/`RelayTaskPrefix`/`RelayReconcileTask`/`leaseTTL`/`progressInterval` 移入本文件。
   - 删除 Lua 脚本、version/token/status 字段、`RelayTarget`/`MakeTargetKey`/`ParseTargetKey`/`ScanRunningStates`。
2. 重写 `internal/relay/distributed.go`：
   - `DistributedRelay{store, manager, nodeID, asynqClient}`；
   - `Start`/`Stop`/`Reconcile`/`EnqueueReconcile`/`OnProgress` 按 design 状态机实现；
   - 删除 `GenerateToken`。
3. 改 `internal/relay/manager.go`：
   - `TargetKey` 用 `:` 分隔；`Task.done` 保留并注释「进程完全退出信号」；
   - 新增 `OnExit`/`OnProgress` 回调字段。
4. 改 `internal/relay/ffmpeg.go`：
   - 命令追加 `-progress pipe:1 -stats_period 5`；http/https 源追加 reconnect 参数；
   - stdout reader：`bufio.Scanner` 解析 `key=value`，`progress=continue` 帧调 `OnProgress`；
   - `watchPullProcess`：非 cancel 的异常退出调 `OnExit`。
5. 瘦身 `internal/task/reconcile.go`：
   - payload `{app, stream}`；`ProcessTask` → `svcCtx.DistRelay.Reconcile`；删本地 `generateToken`。
6. 改 `internal/svc/servicecontext.go`：
   - `StateManager` → `Store`；`DistRelay` 注入 `AsynqClient`；
   - 接 `Manager.OnExit`（入队）与 `Manager.OnProgress`（续租）。
7. 检查 logic 两个文件对 DistRelay 的调用与新签名一致。
8. 全量验证。

## Validation

- `go build ./app/oryxserver/...`
- `go vet ./app/oryxserver/...`
- `go test ./app/oryxserver/...`
- `git diff --check`
- 残留检查：`rg -n "DesiredVersion|LeaseToken|EvalCtx|lua|GenerateToken|ScanRunningStates|MakeTargetKey" app/oryxserver/internal/relay app/oryxserver/internal/task` 应为空。

## Risky files / rollback points

- `state.go`：Redis 操作语义变化最大；rollback = git revert。
- `ffmpeg.go`：progress reader 引入 stdout 管道，注意避免阻塞导致 ffmpeg 卡死（reader 必须常驻消费）。
- `manager.go`：回调与 done 信号语义不能被破坏（替换语义依赖 done）。
- `reconcile.go`：asynq 返回 error 的路径要与「该 ack 的 ack」区分清楚。

## Follow-up checks before `task.py start`

- [ ] 确认 `SetnxEx`/`Expire`/`Get`/`Del` Ctx 变体与 go-zero 版本匹配（已用 go doc 验证存在）。
- [ ] 确认 `asynq.TaskID` 选项可用（已用 go doc 验证存在）。
