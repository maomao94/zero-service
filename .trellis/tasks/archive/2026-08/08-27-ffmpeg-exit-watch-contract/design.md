# 技术设计：FFmpeg 退出与 stdout 契约

## 1. 通用进程 API

```go
var ErrProcessExists = errors.New("ffmpeg process already exists")

type ExitResult struct {
	ID         string
	WaitErr    error
	ContextErr error
}

type StartOption func(*startOptions)

func WithExitHandler(fn func(ExitResult)) StartOption
func WithStdoutHandler(fn func(id, line string)) StartOption
```

`Manager.Start` 在写锁下检查 ID。已存在时返回包装 `ErrProcessExists`，不 cancel、不替换旧进程。显式替换由业务方调用 `Stop` 后再 `Start`。

Manager 使用 `logx.WithContext(processCtx)` 记录进程日志。`Cmd.Wait()` 的原始结果进入 `WaitErr`，`processCtx.Err()` 进入 `ContextErr`。标准库可能优先返回 `*exec.ExitError`，因此业务不能只用 `WaitErr` 判断 cancel/timeout。

所有已启动进程退出都按以下顺序处理：

```text
Wait 返回
→ 按对象身份删除 Manager 条目
→ 同步调用 exit handler(ExitResult)
→ cancel process context
→ close(done)
```

## 2. 通用 stdout watcher

删除 `Progress`、`ProgressStatus` 和 `WatchProgress`。替换为逐行同步观察：

```go
func WatchOutput(rc io.ReadCloser, fn func(line string)) error
```

- `Scanner` 每读取一行立即同步调用 `fn`。
- watcher 不解析 `key=value`，不聚合 block，不保存状态。
- 配置 `WithStdoutHandler` 时 Manager 创建 `StdoutPipe`；未配置时不碰 stdout。
- callback 不能同步 Stop/替换自身 ID，因为 reader 完成后 watcher 才能 Wait。

## 3. Relay stdout 业务

`PullRegistry` 使用 `WithStdoutHandler`。relay 业务逐行解析 `key=value`，遇到 `progress=continue` 时触发续租；其他 key/value 是否记录由业务自行决定。`progress=end` 不续租。

## 4. Relay 最大运行时长

Proto 新增：

```protobuf
uint64 max_duration_seconds = 6 [json_name = "maxDurationSeconds"];
```

语义：`0` 不限制，正数为该 relay 从首次 Start 起允许运行的总秒数。

Redis state 新增：

```go
DeadlineAtUnix int64 `json:"deadline_at_unix,omitempty"`
```

- 初次 Start 将秒数转换为绝对 Unix 截止时间并保存。
- Reconcile 读取同一个截止时间，计算剩余 duration；不能重新获得完整时长。
- `PullRegistry` 在 `context.WithoutCancel(requestCtx)` 基础上创建 deadline context，并交给 Manager/builder。
- 不调用 `ffmpeg-go.WithTimeout`：它丢弃 cancel，且相对时长会在 Reconcile 时被重置。

## 5. Relay 退出策略

`ffmpegx` 不包含任何补拉策略。relay 根据 `ExitResult` 和 `RelayState` 决策：

- 手动 Stop：业务先删除 metadata/state/lease，exit callback 身份校验后跳过。
- `ContextErr == context.DeadlineExceeded`：达到最大时长，删除 state/lease，不补拉。
- `ContextErr == context.Canceled`：按业务当前状态处理，不由 Manager 推断。
- `ContextErr == nil`：FFmpeg 自行结束；截止时间未到且 state 仍存在时释放 lease 并入队 Reconcile。
- Reconcile 发现截止时间已到：删除 state/lease，不启动进程。

## 6. 兼容与生成

- Proto 字段是向后兼容新增字段，默认 0 保持无限时长。
- 修改 `oryxserver.proto` 后运行 `app/oryxserver/gen.sh`，不手工修改生成文件。
- Redis 旧 state 没有 `deadline_at_unix`，按 0（无限）兼容。
