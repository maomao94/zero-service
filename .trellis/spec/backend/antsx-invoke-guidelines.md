# antsx Invoke 使用规范

> `antsx.Invoke` / `InvokeAllSettled` 并行任务编排的 canonical source，覆盖签名、选型、取消、panic 防护和测试断言。

## When to read

- 使用或修改 `common/antsx` 的 Invoke 系列函数。
- 并行任务需要 fast-fail、全量等待、协程池、任务超时或 panic 防护。
- 排查并发任务取消不及时、结果错位、panic 被吞或错误聚合异常。

## 核心签名

```go
func Invoke[T any](ctx context.Context, tasks ...Task[T]) ([]T, error)
func InvokeWithReactor[T any](ctx context.Context, r *Reactor, tasks ...Task[T]) ([]T, error)
func InvokeAllSettled[T any](ctx context.Context, tasks ...Task[T]) []SettledResult[T]
func InvokeAllSettledWithReactor[T any](ctx context.Context, r *Reactor, tasks ...Task[T]) []SettledResult[T]
func InvokeCallback[T, U any](ctx context.Context, tasks []Task[T], callback func([]T) (U, error)) (U, error)

type Task[T any] struct {
    Name    string
    Fn      func(ctx context.Context) (T, error)
    Timeout time.Duration
}
```

Contracts:

- `Name` 必填，用于 panic 错误和 `SettledResult` 定位。
- `Fn` 必须检查 `ctx.Done()`，否则 fast-fail 只能发出取消信号，不能强制终止阻塞任务。
- `Timeout=0` 使用全局 ctx；非 0 时 `runTask` 创建子 context。
- Reactor 变体用于控制并发，不改变 fast-fail 或 all-settled 语义。

## 选型

| 场景 | 用哪个 |
| --- | --- |
| 需要全部成功，一个失败就停 | `Invoke` |
| 需要全部成功，并控制并发数 | `InvokeWithReactor` |
| 需要所有结果，失败互不影响 | `InvokeAllSettled` |
| 需要所有结果，并控制并发数 | `InvokeAllSettledWithReactor` |
| 全部成功后聚合转换为另一类型 | `InvokeCallback` |
| 单任务 | 直接用 `Invoke` 单任务路径，不需要协程池 |

## 错误和取消语义

| 函数 | 错误行为 | 取消行为 |
| --- | --- | --- |
| `Invoke` | 返回第一错误（errgroup g.Wait） | errgroup 自动 cancel groupCtx，所有 goroutine 响应 ctx.Done() |
| `InvokeWithReactor` | 返回第一错误（errOnce） | errOnce 调用 cancel()，已在池中的 goroutine 检测 ctx.Done()；阻塞在 pool.Submit 的 goroutine 等池位释放后提交，检测取消后立即退出 |
| `InvokeAllSettled` | 每个任务返回独立 `SettledResult` | 不派生 fast-fail cancel，只有调用方 ctx 影响所有任务 |
| Reactor 变体 | 保留上述语义，额外处理协程池满或关闭 | 调度失败也应进入错误结果或聚合错误 |

> `InvokeWithReactor` 不使用 errgroup。原因是 ants 的 `pool.Submit` 阻塞时不检查 ctx 取消，errgroup 的 g.Go goroutine 会卡在 Submit 上导致 `g.Wait()` 永久挂起。改为 `WaitGroup + errOnce + cancel` 模式。

Good:

```go
results, err := antsx.Invoke(ctx, tasks...)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        // timeout path
    }
}
```

Base:

```go
for _, r := range antsx.InvokeAllSettled(ctx, tasks...) {
    if r.Succeeded() {
        use(r.Val)
    } else {
        logx.Errorf("task=%s err=%v", r.Name, r.Err)
    }
}
```

Bad:

```go
if err.Error() == "task2 boom" { }
```

Use `errors.Is` or `strings.Contains` for error checking. For `InvokeAllSettled`, check `r.Succeeded()` before accessing `r.Val`.

## Validation & Error Matrix

| 条件 | 正确行为 |
| --- | --- |
| 空任务列表 | 返回空结果，不 panic |
| 单任务成功 | 返回单个结果，err 为 nil |
| 单任务失败或 panic | 返回错误，panic 转为错误 |
| 多任务任一失败 | `Invoke` cancel 其他任务并返回第一错误 |
| 任务不检查 ctx | `Invoke` 等该任务自行结束 |
| `InvokeAllSettled` 某任务失败 | 其他任务继续执行，失败项 `Err` 非 nil |
| goroutine 启动前异常 | 结果预填 `SettledResult{Name: t.Name}`，调用方能定位任务 |
| Reactor 池满或关闭 | 不丢任务结果，不吞错误 |

## Wrong vs Correct

Wrong, 闭包捕获循环变量：

```go
for i, task := range tasks {
    go func() {
        s.run(i, task)  // 闭包捕获循环变量，idx 和 task 在 goroutine 启动后才取到
    }()
}
```

Correct:

```go
for i, task := range tasks {
    idx, t := i, task
    go func() {
        s.run(idx, t)
    }()
}
```

Wrong, 不响应取消：

```go
Task[int]{Name: "slow", Fn: func(ctx context.Context) (int, error) {
    time.Sleep(10 * time.Second)
    return 42, nil
}}
```

Correct:

```go
Task[int]{Name: "slow", Fn: func(ctx context.Context) (int, error) {
    select {
    case <-time.After(10 * time.Second):
        return 42, nil
    case <-ctx.Done():
        return 0, ctx.Err()
    }
}}
```

## Internal design constraints

- Panic 防护统一在 `invokeSingle` 中，所有执行路径最终经过它。
- `Invoke` 使用 `golang.org/x/sync/errgroup`：g.Go 管理 goroutine，auto cancel，g.Wait 返回第一错误。决不在此函数中引入自定义 sync.Mutex + errors.Join。
- `InvokeWithReactor` 使用 `WaitGroup + errOnce + cancel`：goroutine 由 ants 池调度，不用 errgroup 桥接（pool.Submit 阻塞不认 ctx 取消）。
- `InvokeWhileReactor` 在 for 循环中检查 `ctx.Err()` 防止 cancel 后继续提交；末尾兜底 `ctx.Err()` 处理 ctx 过期但无人报错的边缘情况。
- `InvokeAllSettled` 首任务在当前 goroutine 同步执行，其余用 `threading.GoSafe` 并行。
- `InvokeAllSettled` 预填 `SettledResult{Name: t.Name}`，防止异常路径返回空结构体。

## Tests Required

- 空任务列表。
- 单任务成功、失败、panic。
- 多任务全部成功。
- fast-fail：一个失败，其他检查 `ctx.Done()` 后退出。
- 多任务全部 panic：断言 `errors.Join` 包含所有 panic 信息。
- `InvokeAllSettled`：一个失败不影响其他，`Succeeded()` 正确反映状态。
- Reactor 变体：协程池满或关闭时的行为。
- 闭包变量：断言 panic 的任务名称和结果索引不乱。
