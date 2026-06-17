# antsx ReplyPool 使用规范

> `antsx.ReplyPool` 异步请求/应答池的 canonical source，覆盖设计决策、配置推导、统计循环、生命周期和测试断言。
> Promise 基础容器和并行组合见 [`antsx-promise-guidelines.md`](./antsx-promise-guidelines.md)。

## When to read

- 使用或修改 `common/antsx` 的 `ReplyPool`、`RequestReply`。
- 排查 entry 超时、重复 ID、统计日志或 Close 行为。
- 需要理解时间轮配置、statLoop 输出或 entry 生命周期。

## 设计决策

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

## Entry 生命周期调试日志

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

## Gotcha: entry 生命周期独立于 caller context

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

## Tests Required

- Register/Resolve/Reject 基本流程
- 自动过期（TTL 触发）
- 重复 ID 返回 wrapped `ErrDuplicateID`（用 `errors.Is` 判断，不可 `==`）
- Close 后 Register 返回 `ErrReplyClosed`
- Close 幂等
- 并发 Resolve/Reject 不竞争
- `handleTimeout` vs `Reject` 竞争（验证 `Len() == 0`）
- 每分钟无条件打印心跳（无活动时段也输出）
- Close 后 rejected 计数准确
- 并发 Register/Resolve 统计准确

## 命名约定

**ReplyPool 专属错误必须带 `Reply` 前缀**，避免与其他池子混淆：

| 错误 | 含义 | 说明 |
|------|------|------|
| `ErrReplyExpired` | 条目超时过期 | ReplyPool 专属 |
| `ErrReplyClosed` | 池已关闭 | ReplyPool 专属 |
| `ErrDuplicateID` | 重复关联 ID | 通用，不加前缀；被 `fmt.Errorf("%w: %s", ErrDuplicateID, id)` wrap，需用 `errors.Is` |

**日志格式**：`%s[1m]`，name 直接作为前缀，不硬编码类型名。使用绝对值而非百分比（理由见"statLoop 内置"一节的设计决策）：

```go
logx.Statf("%s[1m] - registered: %d, resolved: %d, rejected: %d, expired: %d, pending: %d",
    r.cfg.name, s.registered, s.resolved, s.rejected, s.expired, r.Len())
```
