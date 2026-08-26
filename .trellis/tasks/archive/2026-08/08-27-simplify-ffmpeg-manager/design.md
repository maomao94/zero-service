# 技术设计：简化 FFmpeg Manager

## 职责边界

`ffmpegx.Manager` 只管理已经由 builder 构造完成的命令。builder 在进程登记前同步执行，必须快速返回未启动的 `*exec.Cmd`。Manager 不管理 builder 的执行状态，也不解析 FFmpeg 参数。

```go
type process struct {
	id       string
	cmd      *exec.Cmd
	cancel   context.CancelFunc
	handlers startOptions
	done     chan struct{}
}

type Manager struct {
	mu        sync.RWMutex
	processes map[string]*process
}
```

## 生命周期

`Start` 使用 `context.WithCancel(ctx)` 创建 process context，并将其传给 builder。builder 必须使用该 context 构造未启动的 `exec.CommandContext`。Manager 不注册 `context.AfterFunc`：父 context 的 cancel/deadline 会通过派生 context 直接终止命令，Manager 持有的 cancel 只供 Stop、StopAll 和 replacement 使用。

Manager 不擅自调用 `context.WithoutCancel`。是否脱离请求生命周期是业务决策：relay 在 `PullRegistry` 边界显式传入 `context.WithoutCancel(requestCtx)`；需要遵循请求 timeout 的短任务直接传请求 context。

`Start` 先构造 command，再按顺序停止同 ID 旧进程、配置可选 progress pipe、启动 command、登记 process 并启动唯一 watcher。调用方不得并发操作同一个 ID；不同 ID 的操作可以并发。

watcher 顺序固定为：消费 progress stdout 至 EOF、调用一次 `Cmd.Wait()`、按对象身份删除 Manager 条目、根据 process context 是否取消决定是否同步执行 exit handler，最后关闭 `done`。

- `processCtx.Err() != nil`：Stop、StopAll、replacement、调用方 context cancel 或 timeout，属于正常取消，不调用 exit handler。
- `processCtx.Err() == nil`：命令自行成功或失败结束，先清理条目，再同步调用 exit handler。
- `done` 关闭表示 Wait 和同步 exit handler 均已完成。

## Progress

`WithProgressHandler` 是显式契约。配置该 option 时 Manager 调用 `StdoutPipe` 并用 `WatchProgress` 同步分发完整报告；调用方必须保证命令使用 `-progress pipe:1`。未配置时 Manager 不读取 stdout。

Manager 不检测 `cmd.Args`，不判断流是否健康，不实现 watchdog 或自动重启。

## 锁边界

- `Has`、`Count` 使用 `RLock`。
- 登记和按对象身份删除使用 `Lock`。
- builder、Start、cancel、Wait、progress reader、日志和 callback 全部在锁外。
- 不增加全局 lifecycle lock；同 ID 并发生命周期操作不属于支持契约。

## 失败处理

- builder 失败：取消 process context，返回包装错误，Manager 无登记。
- `StdoutPipe` 失败：取消 context，返回包装错误。
- `cmd.Start` 失败：关闭已创建的 stdout pipe、取消 context，返回包装错误。
- Stop 等待超过 `StopWaitTimeout`：记录错误并返回，watcher 后续仍负责 Wait 与清理。

## 兼容性

公开 `Start`、`StartOption` 和 handler 签名保持不变。变化仅是删除 Manager 自动检查 `-progress` 参数；现有 relay builder 已显式配置 `-progress pipe:1`，行为不变。
