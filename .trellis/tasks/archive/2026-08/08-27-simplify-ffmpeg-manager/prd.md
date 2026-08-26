# 简化 FFmpeg 进程状态管理

## Goal

将 `common/ffmpegx.Manager` 收敛为易读、可维护的进程容器，删除为非核心并发场景引入的状态机和命令参数推断，同时保留进程替换、主动停止、context 取消/超时和异常退出回调的正确语义。

## Background

- 当前 `process` 同时包含 `startDone`、`done`、`started` 和三态 atomic state，并通过 `context.AfterFunc` 协调 builder、Stop、replacement 与 Wait 的竞态。
- 当前 Manager 解析 `cmd.Args` 判断 `-progress` 是否输出到 stdout，增加了 FFmpeg 命令语义耦合。
- 实际业务只需要管理已经构造完成的命令；command builder 应快速、同步且只构造未启动的 `exec.Cmd`。

## Requirements

- `process` 仅保留进程管理所需的 `id`、`cmd`、`cancel`、`handlers`、`done`；handlers 属于单次 Start 创建的进程，不属于 Manager 全局状态。
- Manager 在注册进程前同步执行 command builder；builder 必须响应 context、不得启动命令、不得执行长时间阻塞操作。
- 删除 `startDone`、`started`、atomic state 和 `context.AfterFunc`。
- Manager 只通过 `context.WithCancel(ctx)` 从 Start 的首参派生 process context；不使用 `context.AfterFunc`，也不自动调用 `context.WithoutCancel`。builder 必须使用 Manager 传入的 process context 构造 `exec.CommandContext`。
- context 生命周期由调用方明确选择：请求内短任务可直接传请求 context；relay 等长期任务必须在业务边界先使用 `context.WithoutCancel(requestCtx)` 去除 gRPC cancel/deadline，同时保留 trace/value，再传给 Manager。
- `Stop`、`StopAll` 和同 ID replacement 通过 Manager 持有的 process cancel 停止进程；调用方传入 context 的 cancel/timeout 则由 `exec.CommandContext` 自然传播，不再注册重复取消回调。
- watcher 在 `Wait()` 返回后使用 process context 的 `Err()` 区分退出原因：context 已取消为正常停止，不触发 exit handler；context 未取消为进程自行结束，触发 exit handler。process context 可作为 watcher 参数传递，无需保存在 process 结构中。
- callback 使用同步语义，不创建额外 goroutine。自然退出顺序固定为：`Wait()` 返回 → 按对象身份删除 Manager 条目 → 同步执行 exit handler → `close(done)`。因此 `done` 关闭表示进程已回收且退出 callback 已执行完成。
- progress handler 同样由 progress reader 同步调用，以回调完成作为该 progress report 处理完成的边界；业务 handler 必须避免无限阻塞。
- 同 ID replacement 必须先停止并等待旧进程，再登记和启动新进程；旧 watcher 只能删除自身条目。
- Manager 的进程表使用 `sync.RWMutex`：`Has`、`Count` 使用读锁，登记、删除和快照清空使用写锁。锁只保护内存进程表，不覆盖 command builder、`cmd.Start()`、cancel、`Wait()`、progress reader、日志或 callback。
- 不使用全局 lifecycle mutex 串行化所有进程；不同 ID 的启动和停止可以并发，避免进程数量增加后一个慢进程阻塞全部生命周期操作。
- 同一 ID 的 Start/Stop/StopAll 必须由调用方串行化；Manager 保证顺序 replacement 正确，不为同 ID 并发生命周期操作建立状态机或 keyed lock。不同 ID 的操作可以并发。
- `WithProgressHandler` 是显式契约：配置该 option 时 Manager 才创建并消费 stdout pipe；调用方负责确保命令使用 `-progress pipe:1`。Manager 不再解析 `cmd.Args`。
- `WatchProgress` 保持 FFmpeg progress block 解析能力，不增加 watchdog、自动重启或业务健康判断。
- 进程启动成功后 `Wait()` 必须且只能调用一次；Stop/StopAll 有界等待 `done`。
- 更新 `common/ffmpegx` 单测、注释、relay 调用方和相关 Trellis 规范，使其与简化契约一致。

## Out of Scope

- builder 执行期间通过 `Manager.Stop(id)` 取消尚未登记的启动操作。
- 自动检测 FFmpeg `-progress` 参数及输出目标。
- frame/out_time 停滞检测、watchdog、自动 kill/restart。
- 修改 relay 的 Redis、租约、补拉和 proto 行为。

## Acceptance Criteria

- [ ] `process` 不再包含状态机、start channel 或 context 字段，仅保留 `id/cmd/cancel/handlers/done`。
- [ ] `process.go` 不再使用 `sync/atomic`、`context.AfterFunc`、`progressTargetsStdout` 或 `isStdoutProgressTarget`。
- [ ] Manager 不自动移除调用方 context 的 deadline/cancel；长期 relay 调用方显式传入 `context.WithoutCancel(ctx)`。
- [ ] command builder 使用 Manager 提供的 process context 构造 `exec.CommandContext`，Stop 无需额外操作 `cmd.Process`。
- [ ] 主动 Stop、StopAll、replacement、父 context cancel 和 timeout 不触发 exit handler。
- [ ] 进程自行成功或失败退出时，先清理 Manager 条目，再触发 per-process exit handler。
- [ ] exit/progress callback 均同步执行；自然退出的 `done` 只在 exit callback 返回后关闭。
- [ ] 顺序执行的同 ID replacement 不误删或误停新进程；测试和注释明确同 ID 并发操作不属于支持契约。
- [ ] `Has`、`Count` 使用 `RLock`，写操作使用 `Lock`；任何锁都不跨越 `Wait()` 或同步 callback。
- [ ] 不同 ID 的 Stop/Start 不因另一个进程的退出 callback 或停止等待而被全局生命周期锁阻塞。
- [ ] 无 progress handler 时不创建 stdout pipe；有 progress handler 时直接消费 FFmpeg progress stdout。
- [ ] builder/StdoutPipe/cmd.Start 失败时取消 context、清理 pipe 和 Manager 条目。
- [ ] `go test -race` 覆盖 `common/ffmpegx` 与 relay registry 并通过。
- [ ] `go build ./...`、目标 `go vet`、`go test ./...` 和 `git diff --check` 通过。

## Technical Notes

- Manager 不增加全局 lifecycle lock；进程表的 RWMutex 只保护 map，生命周期协调依赖 process 对象身份、context cancel 和 done。
- builder 在生命周期锁外完成，Manager 不承诺停止尚在 builder 中的操作。
- Manager 创建的 cancel 仅服务于 Stop、StopAll 和 replacement；父 context 自身取消由派生 context 和 `exec.CommandContext` 直接传播，不需要 `AfterFunc`。
- `WithProgressHandler` 的公开注释必须说明调用方需要配置 `-progress pipe:1`。
- callback 调用前不能持有 Manager 锁；exit handler 重入 `Has`/`Stop` 时应看到旧条目已清理，且不得死锁。
- progress handler 运行时进程仍在 progress reader 中，不能同步 Stop 或替换自身 ID；exit handler 执行前自身条目已清理，可以安全重入 Manager。
