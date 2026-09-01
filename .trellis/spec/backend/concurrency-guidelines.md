# 并发与异步规范

> 使用 goroutine、go-zero `mr`、`common/antsx`、`common/asynqx`、Promise、ReplyPool、工作流拦截器或共享 map/state 时读取。

## 工具选择速查

| 场景 | 项目选择 | 说明 |
|------|---------|------|
| 同一请求内少量并行 | go-zero `mr.Finish` / `MapReduce` | 按相邻代码处理取消和聚合 |
| typed 并行、需保持顺序 | `antsx.Invoke` | 任一失败时 fast-fail 并取消其他 |
| 收集每项成功/失败 | `antsx.InvokeAllSettled` | 不 fast-fail，逐项检查 Err |
| 限制并发度 | `antsx.Reactor` | 所有者负责关闭 |
| 单个未来结果 | `antsx.Promise` | 带可取消/超时 context |
| correlation ID 应答 | `antsx.ReplyPool` / `mqttx.ReplyRouter` | 先注册后发送 |
| Redis 后台任务队列 | `asynqx` | 见下方详细说明 |

不要为简单同步流程引入 Promise，也不要用裸 goroutine 重写已有并发组件。

依据：`common/antsx/invoke.go`、`common/antsx/promise.go`、`common/antsx/replypool.go`、`common/asynqx/`

## 生命周期规则

| 组件 | 规则 |
|------|------|
| `Invoke` | 结果与输入顺序一致；task 必须响应 context；panic 转为错误 |
| `InvokeAllSettled` | 不 fast-fail；必须逐项检查 `SettledResult.Err` |
| `Promise` | `Resolve`/`Reject` 只有第一次生效；不给永不完成的 Promise 用无界 context |
| `ReplyPool` | 创建 timing wheel 和统计 goroutine，所有者必须 `Close` |
| 任何异步操作 | 说明返回时点；排队/发送成功 ≠ 远端处理/持久化成功 |

## 共享状态与锁

| 规则 | 说明 |
|------|------|
| 保护方式 | 每个可变字段定义唯一保护：mutex/atomic/channel/事务/CAS |
| 指针读取 | 锁外使用时确认对象生命周期不会同时销毁；必要时复制快照 |
| 锁顺序 | 统一锁顺序；持锁区只更新内存状态，不执行慢操作 |
| 连接替换 | mark/snapshot 后锁外清理，回调不能重入 manager/session 锁 |

### 分布式锁（Redis）

```go
// ✓ 正确：使用 go-zero 自带 RedisLock
lock := redis.NewRedisLock(r, key)
lock.SetExpire(seconds)
ok, err := lock.AcquireCtx(ctx)
// ok=false: 未获得锁（并发竞争）
// err: Redis 故障
// 两者分开处理

// ✓ 释放：带随机值校验，防误删他人锁
lock.Release()
```

- 禁止自造 SETNX 锁或闭包工厂注入锁实现
- 锁 key 统一业务域前缀（如 `live:lock:meeting:*`）
- 持有方崩溃时靠 TTL 自动释放，不用手动 Del 兜底

依据：`common/djisdk/drc.go`、`common/socketiox/container.go`、`app/oryxserver/internal/relay`

## asynqx — asynq 任务队列

### 组件

| 组件 | 用途 |
|------|------|
| `NewAsynqClient(addr, pass, db)` | 任务生产者 |
| `NewTaskServer(server, mux)` | 任务消费者，`Start()`/`Stop()` |
| `NewSchedulerServer(server)` | 周期调度器，按 cron 注册任务 |
| `LoggingMiddleware` | 记录耗时、类型、taskId |

### 配置约定

- Redis：5s DialTimeout/ReadTimeout/WriteTimeout，连接池 50
- TaskServer：并发度 20，队列优先级 `critical:6 / default:3 / low:1`
- SchedulerServer：`Asia/Shanghai` 时区，`PostEnqueueFunc` 记录入队错误

### Context 字段传播

```go
// ✓ 正确：解析 payload 后注入 context
func (h *Handler) ProcessTask(ctx context.Context, t *asynq.Task) error {
    var payload SomePayload
    if err := json.Unmarshal(t.Payload(), &payload); err != nil {
        return err
    }
    ctx = logx.ContextWithFields(ctx,
        logx.Field("app", payload.App),
        logx.Field("stream", payload.Stream),
    )
    return h.svcCtx.SomeService.DoSomething(ctx, payload)
}

// ✗ 错误：创建局部 logger 但不注入 context
func (h *Handler) ProcessTask(ctx context.Context, t *asynq.Task) error {
    var payload SomePayload
    json.Unmarshal(t.Payload(), &payload)
    logger := logx.WithContext(ctx).WithFields(logx.Field("app", payload.App))
    logger.Info("开始处理")  // 有 app
    return h.svcCtx.SomeService.DoSomething(ctx, payload) // 下游丢失 app
}
```

依据：`common/asynqx/asynqTaskServer.go:107`、`app/oryxserver/internal/task/reconcile.go`

### 反模式

| 错误做法 | 正确做法 |
|---------|---------|
| 入队成功当作处理成功 | 队列成功 ≠ 消费成功 |
| handler 中 panic | 返回 error 触发重试 |
| cron 任务没有 Retention | 配置 Retention |
| 绕过 LoggingMiddleware | 使用统一封装 |

## 反模式

| 错误做法 | 正确做法 |
|---------|---------|
| `go func()` 没有退出/等待/回收 | 必须有退出策略 |
| 固定 `time.Sleep` 推断异步完成 | 用有上限轮询或可控同步点 |
| 先发送再注册 correlation ID | 先注册后发送 |
| `go test` 通过就声称并发安全 | 检查字段所有权和 CAS |
| 持 manager 锁获取 session 锁后反向加锁 | 统一锁顺序 |

## 验证

- 覆盖取消、超时、panic、重复完成、关闭、快速响应、过期和提交失败
- 使用有上限轮询或可控同步点验证异步结果
- 对目标并发包运行 `go test -race -count=10 ./path/to/package`

---

## Scenario: ffmpegx.Manager 子进程生命周期

> 使用 `common/ffmpegx.Manager` 启动、替换、停止 FFmpeg 或其他 `os/exec` 子进程时适用。

### 核心契约

| 规则 | 说明 |
|------|------|
| context 派生 | 使用 `context.WithCancel`，不自动调用 `WithoutCancel` |
| hook 配置 | 仅通过单次 Start option，不使用全局 hook |
| stdout/stderr | 必须并发持续消费，一个 pipe 不得阻塞另一个 |
| Wait 调用 | Start 成功后 watcher 唯一调用 `Wait()` |
| 重复 ID | 返回 `ErrProcessExists`，不调用新 builder |
| 退出顺序 | pipe reader 结束 → `Wait` → 身份清理 → exit hook → 关闭 `done` |

### Good/Bad 对比

```go
// ✗ 错误：串行读取，stdout 阻塞时 stderr 永远不被消费
WatchOutput(stdout, onStdout)
WatchOutput(stderr, onStderr)

// ✓ 正确：并发消费，两路都结束后再 Wait
var readers sync.WaitGroup
readers.Go(func() { _ = WatchOutput(stdout, onStdout) })
readers.Go(func() { _ = WatchOutput(stderr, onStderr) })
readers.Wait()

// ✓ 正确：重复 ID 显式处理，退出原因结构化
m.Start(ctx, id, build,
    ffmpegx.WithExitHandler(onExit),
    ffmpegx.WithStdoutHandler(onLine),
    ffmpegx.WithStderrHandler(onLine),
)
```

依据：`common/ffmpegx/process.go`、`app/oryxserver/internal/relay/registry.go`
