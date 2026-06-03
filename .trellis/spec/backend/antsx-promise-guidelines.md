# antsx Promise 使用规范

> `antsx.Promise` 异步结果容器和并行组合的 canonical source，覆盖签名、选型、泄漏防护和测试断言。

## When to read

- 使用或修改 `common/antsx` 的 Promise 系列函数。
- 需要异步结果容器、并行组合（All/AllSettled/Race/Any）或链式变换。
- 排查 goroutine 泄漏、Promise 永不完成或错误语义不精确。

## 核心签名

```go
// 结果容器
func NewPromise[T any]() *Promise[T]
func (p *Promise[T]) Resolve(val T)
func (p *Promise[T]) Reject(err error)
func (p *Promise[T]) Await(ctx context.Context) (T, error)

// 并行组合
func PromiseAll[T any](ctx context.Context, promises ...*Promise[T]) ([]T, error)
func PromiseAllSettled[T any](ctx context.Context, promises ...*Promise[T]) []PromiseResult[T]
func PromiseRace[T any](ctx context.Context, promises ...*Promise[T]) (T, error)
func PromiseAny[T any](ctx context.Context, promises ...*Promise[T]) (T, error)

// 结果类型
type PromiseResult[T any] struct {
    Val T
    Err error
}
func (r PromiseResult[T]) Succeeded() bool
```

## 并行组合四象限

| | fast-fail | 容错 |
|---|---|---|
| **等全部** | `PromiseAll` | `PromiseAllSettled` |
| **取最快** | `PromiseRace` | `PromiseAny` |

选型：

| 场景 | 用哪个 |
| --- | --- |
| 需要全部成功，一个失败就停 | `PromiseAll` |
| 需要全部结果，部分失败做降级 | `PromiseAllSettled` |
| 取第一个完成的（含错误） | `PromiseRace` |
| 取第一个成功的，跳过失败 | `PromiseAny` |

Contracts:

- `PromiseAll` 任一失败立即 cancel 返回第一个错误
- `PromiseAllSettled` 每个独立，不互相影响，返回 `[]PromiseResult`
- `PromiseRace` 第一个完成的无论成败都返回
- `PromiseAny` 第一个成功才返回，全部失败才报错
- 空输入：`PromiseAll` 返回空切片，`PromiseRace`/`PromiseAny` 返回 `ErrEmptyPromises`

## 错误语义

| 错误 | 来源 | 含义 |
| --- | --- | --- |
| `ErrEmptyPromises` | `PromiseRace` / `PromiseAny` | 传入空 promises 切片 |
| `io.EOF` | `StreamReader.Recv` | 流结束 |
| `ErrPendingExpired` | `PendingRegistry` | 条目超时过期 |

Good:

```go
results := antsx.PromiseAllSettled(ctx, p1, p2, p3)
for _, r := range results {
    if r.Succeeded() {
        process(r.Val)
    } else {
        log.Printf("failed: %v", r.Err)
    }
}
```

Base:

```go
val, err := antsx.PromiseAny(ctx, cdn1, cdn2, cdn3)
if err != nil {
    if errors.Is(err, antsx.ErrEmptyPromises) {
        // no promises provided
    }
}
```

Bad:

```go
// 不要用 context.Canceled 判断空输入
if errors.Is(err, context.Canceled) { }  // 语义不精确
```

## Goroutine 泄漏防护

`Then`、`Map`、`FlatMap`、`Go` 内部启动 goroutine，依赖 ctx 取消或 Promise 完成来退出。

**如果 ctx 为 `context.Background()` 且 Promise 永远不完成，goroutine 将泄漏。**

Wrong:

```go
// 没有超时的链式操作，可能泄漏
result := antsx.Then(context.Background(), p, transform)
```

Correct:

```go
// 带超时的链式操作
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
result := antsx.Then(ctx, p, transform)
```

## Reactor nil 防护

`Submit`、`Post`、`Go` 对 nil Reactor 返回 `errNilReactor`，不会 panic。

```go
_, err := antsx.Submit(ctx, nil, fn)
if errors.Is(err, errNilReactor) {
    // Reactor not initialized
}
```

## Post 签名

`Post` 是 fire-and-forget，签名不接受 error 返回值：

```go
func Post(ctx context.Context, r *Reactor, fn func(ctx context.Context)) error
```

如果需要 error 返回值，用 `Submit`。

## Validation & Error Matrix

| 条件 | 正确行为 |
| --- | --- |
| 空 promises 切片 | `PromiseAll` 返回空切片，`PromiseRace`/`PromiseAny` 返回 `ErrEmptyPromises` |
| 单个 Promise | 所有函数正常工作 |
| 全部成功 | 按定义顺序返回结果 |
| 全部失败 | `PromiseAll` 返回第一个错误，`PromiseAllSettled` 返回所有错误，`PromiseAny` 返回聚合错误 |
| ctx 取消 | 通过 ctx.Err() 返回，goroutine 不泄漏 |
| nil Reactor | 返回 `errNilReactor` |

## Tests Required

- 空输入：所有四个组合函数
- 单个 Promise：成功和失败
- 全部成功：验证结果顺序
- 部分失败：`PromiseAll` fast-fail，`PromiseAllSettled` 独立返回
- 全部失败：错误聚合
- ctx 超时/取消：验证 goroutine 不泄漏
- `PromiseResult.Succeeded()`：成功和失败两种路径

## PendingRegistry 设计决策

### 时间轮自动推导

**问题**：用户需要理解 interval/slots 关系才能正确配置 TimingWheel。

**决策**：移除 `WithTimingWheel`，改为 `autoTimingWheel(ttl)` 自动推导。

```go
func autoTimingWheel(ttl time.Duration) (interval time.Duration, numSlots int) {
    const slots = 300  // 与 go-zero 生产默认对齐
    interval = ttl / slots
    if interval < 10*time.Millisecond { interval = 10 * time.Millisecond }
    if interval > time.Second { interval = time.Second }
    return interval, slots
}
```

**效果**：

| defaultTTL | interval | numSlots | 单圈容量 | 精度 |
|-----------|----------|----------|---------|------|
| 5s | 16.7ms | 300 | 5s | ±16.7ms |
| 30s | 100ms | 300 | 30s | ±100ms |
| 60s | 200ms | 300 | 60s | ±200ms |

### 双层计数器

**问题**：需要同时支持累计总计和区间增量统计。

**决策**：双层 `atomic.Uint64` 计数器。

```go
type PendingRegistry[T any] struct {
    // 累计总计
    registered atomic.Uint64
    resolved   atomic.Uint64
    rejected   atomic.Uint64
    expired    atomic.Uint64

    // 区间增量（StartStatsLoop 用 Swap(0) 获取并清零）
    deltaRegistered atomic.Uint64
    deltaResolved   atomic.Uint64
    deltaRejected   atomic.Uint64
    deltaExpired    atomic.Uint64
}
```

**关键点**：
- 计数器增量在锁内执行，保证与状态变更原子一致
- `Close` 用局部变量累加后一次性 `Add`，减少原子操作次数
- `Register` 失败时回滚计数器：`r.registered.Add(^uint64(0))`

### Stats 设计

```go
type RegistryStats struct {
    // 累计总计
    Registered uint64
    Resolved   uint64
    Rejected   uint64
    Expired    uint64
    Pending    int

    // 区间增量
    IntervalRegistered uint64
    IntervalResolved   uint64
    IntervalRejected   uint64
    IntervalExpired    uint64
    IntervalDuration   time.Duration
}
```

**使用模式**：

```go
reg := antsx.NewPendingRegistry[string](
    antsx.WithDefaultTTL(5 * time.Second),
    antsx.WithStatsLoop(1 * time.Minute),
)
defer reg.Close()

// 或手动启动
stop := reg.StartStatsLoop(ctx, time.Minute, func(s antsx.RegistryStats) {
    logx.Statf("pending - qpm: %d, hit_ratio: %.1f%%, pending: %d",
        s.IntervalRegistered+s.IntervalResolved+s.IntervalRejected+s.IntervalExpired,
        float64(s.IntervalResolved)/float64(s.IntervalRegistered)*100,
        s.Pending)
})
defer stop()
```

### RequestReply 自定义 TTL

```go
// 使用默认 TTL
resp, err := antsx.RequestReply(ctx, reg, "req-1", sendFn)

// 覆盖本次请求的 TTL
resp, err := antsx.RequestReply(ctx, reg, "req-2", sendFn, 10*time.Second)
```

### Tests Required (PendingRegistry)

- Register/Resolve/Reject 基本流程
- 自动过期（TTL 触发）
- 重复 ID 返回 `ErrDuplicateID`
- Close 后 Register 返回 `ErrRegistryClosed`
- Close 幂等
- 并发 Resolve/Reject 不竞争
- `handleTimeout` vs `Reject` 竞争（验证总计数一致）
- Stats 准确性（累计 + 区间）
- Stats Loop 无活动时不调用 logFn
- Close 后 rejected 计数准确
- 并发 Register/Resolve 统计准确
