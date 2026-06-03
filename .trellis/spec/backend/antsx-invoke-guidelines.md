# antsx Invoke 使用规范

> antsx 的 `Invoke`/`InvokeAllSettled` 系列函数用于并行任务编排。本规范覆盖正确用法、常见错误和内部设计决策。

---

## 1. 核心签名

```go
// fast-fail: 任一失败立即取消其他任务
func Invoke[T any](ctx context.Context, tasks ...Task[T]) ([]T, error)

// fast-fail + 协程池
func InvokeWithReactor[T any](ctx context.Context, r *Reactor, tasks ...Task[T]) ([]T, error)

// 全量等待: 每个任务独立返回结果，互不取消
func InvokeAllSettled[T any](ctx context.Context, tasks ...Task[T]) []SettledResult[T]

// 全量等待 + 协程池
func InvokeAllSettledWithReactor[T any](ctx context.Context, r *Reactor, tasks ...Task[T]) []SettledResult[T]
```

---

## 2. 选型决策树

| 场景 | 用哪个 |
|------|--------|
| 需要全部成功，一个失败就停 | `Invoke` |
| 需要全部成功 + 控制并发数 | `InvokeWithReactor` |
| 需要所有结果（包括失败），互不影响 | `InvokeAllSettled` |
| 需要所有结果 + 控制并发数 | `InvokeAllSettledWithReactor` |
| 单任务 | 直接用 `Invoke` 单任务路径，不需要协程池 |

---

## 3. Task 定义规范

```go
type Task[T any] struct {
    Name    string                         // 必填: 用于 panic 错误和 SettledResult 定位
    Fn      func(ctx context.Context) (T, error)  // 必须检查 ctx.Done()
    Timeout time.Duration                  // 可选: 0 表示使用全局超时
}
```

**要求**:
- `Name` 必填，不能为空字符串
- `Fn` 必须检查 `ctx.Done()` 以响应 fast-fail 取消
- 单任务超时通过 `Timeout` 字段设置，`runTask` 会自动创建 `WithTimeout` 子 context

---

## 4. 错误处理

### 4.1 Invoke 的错误收集

`Invoke` 使用 `errors.Join` 收集所有错误（包括多个 goroutine 同时 panic）。调用方用 `errors.Is` 判断：

```go
results, err := antsx.Invoke(ctx, tasks...)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        // 超时
    }
    // err 包含所有 panic 和业务错误信息
}
```

### 4.2 InvokeAllSettled 的结果处理

每个任务返回独立的 `SettledResult`：

```go
for _, r := range results {
    if r.Succeeded() {
        // r.Val 可用
    } else {
        // r.Err 是业务错误或 panic 错误
    }
}
```

### 4.3 SettledResult 兜底值

`InvokeAllSettled` 在 goroutine 启动前预填 `SettledResult{Name: t.Name}`。即使 goroutine 异常退出，调用方也能拿到带 `Name` 的结果条目，而不会是空结构体。

---

## 5. Context 取消语义

### 5.1 Invoke 的 fast-fail

```
任务 A 失败
  → addErr(err) 调用 cancel()
    → 其他任务的 ctx.Done() 返回
      → 其他任务的 runTask 检测到取消，返回 context.Canceled
        → 所有错误通过 errors.Join 收集
```

**注意**: fast-fail 是"尽快取消"，不是"立即返回"。如果任务不检查 `ctx.Done()`，函数会等到任务自行结束。

### 5.2 InvokeAllSettled 无 fast-fail

`InvokeAllSettled` 不派生 `WithCancel`，任务之间互不影响。只有调用方传入的 ctx 取消才会影响所有任务。

---

## 6. 禁止模式

### 6.1 不要在 goroutine 闭包中直接捕获循环变量

**错误**:
```go
for i, task := range tasks {
    go func() {
        s.runTaskWithRecovery(i, task) // i 和 task 可能被覆盖
    }()
}
```

**正确**:
```go
for i, task := range tasks {
    idx, t := i, task
    go func() {
        s.runTaskWithRecovery(idx, t)
    }()
}
```

蚂蚁池的 `r.Go` 同理，即使内部有 goroutine 调度，也要复制循环变量。

### 6.2 不要假设 Invoke/InvokeAllSettled 立即返回

fast-fail 的 cancel 是信号，不是强制终止。如果任务在不可取消的操作中阻塞，调用会等到任务完成。

### 6.3 不要忘记 ctx.Done()

```go
// 错误: 不检查 ctx，fast-fail 无效
Task[int]{Name: "slow", Fn: func(ctx context.Context) (int, error) {
    time.Sleep(10 * time.Second)
    return 42, nil
}}

// 正确: 检查 ctx
Task[int]{Name: "slow", Fn: func(ctx context.Context) (int, error) {
    select {
    case <-time.After(10 * time.Second):
        return 42, nil
    case <-ctx.Done():
        return 0, ctx.Err()
    }
}}
```

---

## 7. 内部设计决策

### 7.1 三层 panic 防护

| 层级 | 机制 | 文件 |
|------|------|------|
| goroutine 边界 | `threading.GoSafe` / ants 池 recovery | `invoke.go:goTask` / reactor | 
| 任务执行层 | `runTaskWithRecovery` / `settleTask` 的 `defer recover()` | `invoke.go:60-73` / `invoke.go:267-278` |
| 业务方法层 | 调用方自己的 `recover` | — |

`wg.Done()` 放在最外层 defer，任何一层 panic 都不会跳过。

### 7.2 错误收集用 errors.Join 而非 errOnce

**原方案** (已弃用): `sync.Once` + `firstErr`，只记录第一个错误，后续 panic 被静默吞掉。

**现方案**: `sync.Mutex` + `[]error` + `errors.Join`，所有错误和 panic 都可见。

### 7.3 Invoke 的第一个任务同步执行

多任务时，第一个在当前 goroutine 同步执行，其余并发。第一个放在**最后**启动，让其他任务的错误可以先触发 cancel，第一个任务通过 `ctx.Done()` 响应。

---

## 8. 测试要求

### 8.1 必须覆盖的场景

- 空任务列表
- 单任务成功/失败/panic
- 多任务全部成功
- fast-fail: 一个失败，其他检查 ctx.Done() 退出
- 多任务全部 panic: 验证 errors.Join 包含所有 panic 信息
- InvokeAllSettled: 一个失败不影响其他
- SettledResult Succeeded() 正确反映状态
- Reactor 变体: 协程池满/关闭时的行为

### 8.2 测试断言要点

```go
// fast-fail 不要用字符串相等判断错误
if err.Error() != "task2 boom" { } // 错误: errors.Join 包含多个错误

// 用 strings.Contains 或 errors.Is
if !strings.Contains(err.Error(), "task2 boom") { } // 正确

// SettledResult
if !r.Succeeded() && r.Name != "" { } // 正确: Name 一定被填过
```

---

## 9. 常见错误

### 错误: SettledResult.Succeeded() 为 true 但任务实际失败了

**原因**: goroutine panic 被 GoSafe 捕获，但 `results[idx]` 未被 `settleTask` 设置，预填的 `Name` 存在但 `Err` 为 nil。

**当前状态**: 已通过预填 `SettledResult{Name: t.Name}` 缓解，但预填值本身没有 `Err`。正常情况 `settleTask` 自己的 recovery 会保证返回值有 `Err`，不会触发此问题。

### 错误: InvokeWithReactor 的闭包捕获了错误的变量

**症状**: `TestInvokeWithReactor_PanicRecovery` 中 panic 的任务名称或结果索引错乱。

**根因**: `for i, task := range tasks { r.Go(func() { ... i, task ... }) }` 中闭包捕获了循环变量。

**修复**: `idx, t := i, task` 后传入闭包。
