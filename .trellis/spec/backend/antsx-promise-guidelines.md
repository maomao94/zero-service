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
| `ErrReplyExpired` | `ReplyPool` | 条目超时过期 |
| `ErrReplyClosed` | `ReplyPool` | 池已关闭 |
| `ErrDuplicateID` | `ReplyPool` | 重复关联 ID |

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

## ReplyPool 设计决策

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

### 增量计数器（取消累计）

**问题**：需要为 `statLoop` 提供区间增量统计，同时无需暴露外部统计 API。

**决策**：仅保留增量计数器，移除累计计数器。`Stats()` / `RegistryStats` 无生产消费者，一并移除。

```go
type ReplyPool[T any] struct {
    // 区间增量（statLoop 用 Swap(0) 获取并清零）
    deltaRegistered atomic.Uint64
    deltaResolved   atomic.Uint64
    deltaRejected   atomic.Uint64
    deltaExpired    atomic.Uint64
}
```

**关键点**：
- 计数器增量在锁内执行，保证与状态变更原子一致
- 累计计数器（`registered`/`resolved`/`rejected`/`expired`）曾存在但仅在测试中读取，已移除
- `pendingEntry.removed` 字段写入但从未读取，已移除
- 在生产代码中通过 `Len()` 和 `statLoop` 日志验证行为，非测试专用 `Stats()` API

### statLoop 内置（无需手动管理）

**问题**：`StartStatsLoop`/`WithStatsLoop` 需要用户手动启停，容易遗漏，且不同池子日志无法区分。

**决策**：移除 `StartStatsLoop` 和 `WithStatsLoop` API，构造时自动启动内置 statLoop，在 `Close()` 时自动停止。生命周期绑定 ReplyPool，不暴露外部管理入口。

```go
func (r *ReplyPool[T]) statLoop() {
    defer r.statsWG.Done()
    ticker := time.NewTicker(defaultStatsInterval) // 1 min
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            s := r.reset()
            logx.Statf("%s[1m] - registered: %d, resolved: %d, rejected: %d, expired: %d, pending: %d",
                r.cfg.name, s.registered, s.resolved, s.rejected, s.expired, r.Len())
        case <-r.statsCtx.Done():
            return
        }
    }
}
```

**关键点**：
- 构造后在独立 goroutine 启动：`go r.statLoop()`
- `Close()` 调用 `r.statsCancel()` 退出循环，`r.statsWG.Wait()` 等待 goroutine 结束
- 每分钟无条件打印一次心跳日志（即使无活动），持续可见
- 日志格式与 go-zero `cache.go` 对齐：`ReplyPool({name})[1m]`
- 增量数据通过 `reset()` 的 `Swap(0)` 读取并清零，无需维护重复的累积计数器

**使用模式**：

```go
reg := antsx.NewReplyPool[string](
    antsx.WithDefaultTTL(5 * time.Second),
)
defer reg.Close()

// 统计循环自动运行，无需手动启动
```

### WithName 选项

**问题**：当服务中使用多个 ReplyPool 池时，日志输出无法区分来自哪个池。

**决策**：添加 `WithName(name)` 选项和 `name` 字段，默认值为 `"reply-pool"`。

```go
type registryConfig struct {
    defaultTTL time.Duration
    name       string // 默认 "reply-pool"
}

func WithName(name string) RegistryOption {
    return func(cfg *registryConfig) {
        cfg.name = name
    }
}
```

日志输出示例：

```
drc-cache[1m] - registered: 12, resolved: 9, rejected: 1, expired: 2, pending: 3
reply-pool[1m] - registered: 0, resolved: 0, rejected: 0, expired: 0, pending: 0
```

**设计决策**：统计用绝对值而非百分比，因为 `registered` 和 `resolved`/`rejected`/`expired` 跨窗口不对齐——百分比分母选 `done` 或 `registered` 都会产生误导。绝对值只陈述"这一分钟发生了什么"，不暗示因果。

**约定**：
- 每个不同用途的池子应设定不同的 `name`
- 名称仅用于日志区分，无其他语义影响
- 默认名称 `"reply-pool"` 适合单一池子场景

### RequestReply 自定义 TTL

```go
// 使用默认 TTL
resp, err := antsx.RequestReply(ctx, reg, "req-1", sendFn)

// 覆盖本次请求的 TTL
resp, err := antsx.RequestReply(ctx, reg, "req-2", sendFn, 10*time.Second)
```

### Entry 生命周期调试日志

ReplyPool 在三个出口点打印 debug 日志，通过 `tid` 字段串联 entry 完整生命周期：

```go
// handleTimeout - 时间轮触发过期
logx.Debugw(fmt.Sprintf("[%s] entry %s expired by timing wheel, pending=%d", r.cfg.name, id, r.Len()),
    logx.Field("tid", id),
)

// Resolve - 外部异步响应到达
logx.Debugw(fmt.Sprintf("[%s] entry %s resolved, pending=%d", r.cfg.name, id, r.Len()),
    logx.Field("tid", id),
)

// Reject - 显式拒绝或 sendFn 失败
logx.Debugw(fmt.Sprintf("[%s] entry %s rejected: %v, pending=%d", r.cfg.name, id, err, r.Len()),
    logx.Field("tid", id),
)
```

**约定**：
- `tid` 作为结构化字段，用于日志检索过滤
- 消息文本包含 pool 名称、entry id、动作和 pending 计数
- `RequestReply` 本身**不**打印日志：ctx 超时错误已通过返回值传递，entry 的最终出口由上述三个点覆盖

### Gotcha: entry 生命周期独立于 caller context

**场景**：gRPC ctx 10 秒超时 → `RequestReply` 返回 `context.DeadlineExceeded`，但异步任务（如 DJI SDK 指令）仍在执行。

**错误做法**：在 ctx 超时后主动 `Reject` 清理 entry。

```go
// Wrong - 会阻止异步任务正常 Resolve
val, err := promise.Await(ctx)
if err != nil && reg.Has(id) {
    reg.Reject(id, err) // 异步任务回调时 entry 已不存在
}
```

**正确做法**：entry 保留在池中，由时间轮管理其生命周期。

```go
// Correct - entry 独立于 caller context
return promise.Await(ctx)
```

- 异步任务完成后 `Resolve` 正常触发，entry 清理
- 若异步任务始终不响应，时间轮兜底触发 `handleTimeout`
- 日志通过 `tid` 可追踪：ctx done → entry still pending → resolved/expired`

### Tests Required (ReplyPool)

- Register/Resolve/Reject 基本流程
- 自动过期（TTL 触发）
- 重复 ID 返回 `ErrDuplicateID`
- Close 后 Register 返回 `ErrReplyClosed`
- Close 幂等
- 并发 Resolve/Reject 不竞争
- `handleTimeout` vs `Reject` 竞争（验证 `Len() == 0`）
- 每分钟无条件打印心跳（无活动时段也输出）
- Close 后 rejected 计数准确
- 并发 Register/Resolve 统计准确

### 命名约定

**ReplyPool 专属错误必须带 `Reply` 前缀**，避免与其他池子混淆：

| 错误 | 含义 | 说明 |
|------|------|------|
| `ErrReplyExpired` | 条目超时过期 | ReplyPool 专属 |
| `ErrReplyClosed` | 池已关闭 | ReplyPool 专属 |
| `ErrDuplicateID` | 重复关联 ID | 通用，不加前缀 |

**日志格式**：`%s[1m]`，name 直接作为前缀，不硬编码类型名。使用绝对值而非百分比（理由见"statLoop 内置"一节的设计决策）：

```go
logx.Statf("%s[1m] - registered: %d, resolved: %d, rejected: %d, expired: %d, pending: %d",
    r.cfg.name, s.registered, s.resolved, s.rejected, s.expired, r.Len())
```
