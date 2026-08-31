# 并发与异步规范

## 适用范围

使用 goroutine、go-zero `mr`、`common/antsx`、`common/asynqx`、Promise、ReplyPool、工作流拦截器或共享 map/state 时读取。

## 选择工具

| 场景 | 项目选择 |
| --- | --- |
| 同一请求内少量并行查询/转换 | go-zero `mr.Finish` / `MapReduce`，按相邻代码处理取消和聚合 |
| typed 并行任务、需保持输入顺序 | `antsx.Invoke`；任一失败时 fast-fail 并取消其他任务 |
| 需要收集每项成功/失败 | `antsx.InvokeAllSettled` |
| 需要限制并发度 | `antsx.Reactor` 对应方法，所有者负责关闭 |
| 单个未来结果及组合 | `antsx.Promise`，所有等待都带可取消/超时 context |
| correlation ID 请求应答 | `antsx.ReplyPool` 或 `mqttx.ReplyRouter`，先注册后发送 |
| Redis 后台任务队列 | `asynqx` 封装 asynq Client/TaskServer/SchedulerServer，见下方 |

不要为简单同步流程引入 Promise，也不要用裸 goroutine 重写已有并发/关联组件。

依据：`common/antsx/invoke.go`、`common/antsx/promise.go`、`common/antsx/replypool.go`、`common/asynqx/`、仓库内 `mr.` 调用点。

## asynqx — asynq 任务队列封装

### 组件

| 组件 | 用途 |
|------|------|
| `NewAsynqClient(addr, pass, db)` | 创建任务生产者，将任务入队到 Redis |
| `NewTaskServer(server, mux)` | 任务消费者，`Start()` 运行 worker，`Stop()` 优雅关闭 |
| `NewSchedulerServer(server)` | 周期调度器，按 cron 表达式注册周期任务，`Start()`/`Stop()` |
| `LoggingMiddleware` | asynq 中间件，记录每次任务处理的耗时、类型、taskId |
| `BaseLogger` | asynq.Logger 实现，适配 go-zero logx |

### 配置约定

- Redis 连接统一设置 5s DialTimeout/ReadTimeout/WriteTimeout，连接池 50。
- TaskServer 默认并发度 20，队列优先级: `critical:6 / default:3 / low:1`。
- SchedulerServer 使用 `Asia/Shanghai` 时区，`PostEnqueueFunc` 记录入队错误。
- `IsFailure` 对所有 error 返回 true（失败触发 asynq 重试）。

依据：`common/asynqx/asynqClient.go`、`common/asynqx/asynqTaskServer.go`、`common/asynqx/asynqSchedulerServer.go`、`common/asynqx/log.go`。

### OTel 追踪

- `StartAsynqProducerSpan(ctx, typename)` — 生产者端创建 trace span（`SpanKindProducer`），设置 `asynq.type` 属性。
- `StartAsynqConsumerSpan(ctx, typename)` — 消费者端创建 trace span（`SpanKindConsumer`），设置 `asynq.type` 属性。
- 调用方负责在 span 内入队/处理任务，结束后关闭 span。

依据：`common/asynqx/asynqClient.go`、`common/asynqx/asynqTaskServer.go`。

### 反模式 (asynqx)

- 将 asynq 入队成功当作任务处理成功（队列成功 ≠ 消费成功）。
- 在 TaskServer handler 中 panic 而不返回 error（asynq 依赖 error 决定重试）。
- SchedulerServer 注册的 cron 任务没有 `Retention` 配置（任务过期被清理）。
- 绕过 `LoggingMiddleware` 直接注册 handler（丢失统一日志）。
- **Asynq handler 创建局部 logger 但不注入 context**——下游函数用 `logx.WithContext(ctx)` 时丢失业务字段（taskId、app、stream 等），导致日志无法关联同一任务。

### Asynq Handler Context 字段传播

**问题**：asynqx 自动注入 `type/taskId/payloadSize` 到 context，但业务 handler 解析 payload 后的字段（app、stream、uuid 等）只在局部 logger 中，传给下游的原始 ctx 丢失这些字段。

**模式**：在 Asynq handler 入口处，解析 payload 后立即用 `logx.ContextWithFields` 将业务字段注入 context，下游所有 `logx.WithContext(ctx)` 自动携带。

```go
// Correct: 解析 payload 后注入 context
func (h *Handler) ProcessTask(ctx context.Context, t *asynq.Task) error {
    var payload SomePayload
    if err := json.Unmarshal(t.Payload(), &payload); err != nil {
        return err
    }

    // 将业务字段注入 context，下游日志自动携带
    ctx = logx.ContextWithFields(ctx,
        logx.Field("app", payload.App),
        logx.Field("stream", payload.Stream),
        logx.Field("uuid", payload.UUID),
    )

    // 下游函数用 logx.WithContext(ctx) 即可，无需手动传递字段
    return h.svcCtx.SomeService.DoSomething(ctx, payload)
}

// Wrong: 创建局部 logger 但不注入 context
func (h *Handler) ProcessTask(ctx context.Context, t *asynq.Task) error {
    var payload SomePayload
    json.Unmarshal(t.Payload(), &payload)

    logger := logx.WithContext(ctx).WithFields(
        logx.Field("app", payload.App),
    )
    logger.Info("开始处理")  // 有 app

    // 下游丢失 app 字段
    return h.svcCtx.SomeService.DoSomething(ctx, payload)
}
```

**Why**：asynqx 的 `StartAsynqConsumerSpan` 通过 `logx.ContextWithFields` 注入 `type/taskId/payloadSize`，这些字段贯穿整个 context 生命周期。业务字段应采用相同模式，确保同一任务的所有日志可通过 taskId 或业务 ID 关联检索。

依据：`common/asynqx/asynqTaskServer.go:107`（context 字段注入）、`app/oryxserver/internal/task/reconcile.go`（业务字段注入示例）。

## 生命周期与错误

- `Invoke` 保持结果与输入顺序一致；task 必须响应 context。panic 会转换为错误，调用方不能依赖进程崩溃。
- `InvokeAllSettled` 不 fast-fail；调用方必须逐项检查 `SettledResult.Err`，不能只看切片非空。
- `Promise.Resolve` / `Reject` 只有第一次生效。派生 goroutine 依赖源 Promise 完成或 context 取消；不得给可能永不完成的 Promise 使用无界 context。
- `ReplyPool` 创建 timing wheel 和统计 goroutine，所有者必须 `Close`；重复 ID、超时、已关闭分别保留明确错误。
- 任何异步操作都要说明返回时点。排队/发送成功不等于远端处理或持久化成功。

## 共享状态与锁

- 为每个可变字段定义唯一保护方式：mutex、atomic、channel、事务或 CAS，不混合未说明的访问路径。
- 读取指针后若锁外使用，确认对象生命周期不会同时销毁；必要时复制快照或使用版本/session ID 验证。
- 统一锁顺序；持锁区只更新内存状态，不执行网络、数据库、日志回调或用户代码。
- 删除/替换连接时采用 mark/snapshot 后锁外清理，回调不能重入持有的 manager/session 锁。

依据：`common/djisdk/drc.go`、`common/socketiox/container.go`、`common/antsx/replypool.go`。

## 反模式

- `go func()` 后没有退出、等待、回收或错误通道。
- 固定 `time.Sleep` 推断异步任务已经完成。
- 先发送消息再注册 correlation ID，留下快速响应丢失窗口。
- 只因 `go test` 通过就声称并发安全，未检查字段所有权和 CAS。
- 持 manager 锁获取 session 锁后，在另一条路径反向加锁。

## Scenario: `ffmpegx.Manager` 子进程生命周期

### 1. Scope / Trigger

- 使用 `common/ffmpegx.Manager` 启动、替换、停止 FFmpeg 或其他 `os/exec` 子进程时适用。
- 目标是保证 Start 成功后 `Wait()` 恰好一次、同 ID 不会隐式替换、主动取消有结构化退出结果，且锁内不执行 builder、等待或回调。

### 2. Signatures

```go
type CommandBuilder func(ctx context.Context) (*exec.Cmd, error)
func (m *Manager) Start(ctx context.Context, id string, build CommandBuilder, opts ...StartOption) error
func WithExitHandler(func(ExitResult)) StartOption
func WithStdoutHandler(func(id, line string)) StartOption
func WithStderrHandler(func(id, line string)) StartOption
func (m *Manager) Stop(id string) bool
func (m *Manager) StopAll()
```

### 3. Contracts

- Manager 使用 `context.WithCancel` 从首参 context 派生进程 context/cancel，不自动调用 `context.WithoutCancel`；长期任务由业务边界显式选择是否脱离请求取消。
- exit/stdout/stderr hook 仅通过单次 Start option 配置，不使用 Manager 全局 hook，也不把 hook 放入 context。
- 配置 `WithStdoutHandler` 或 `WithStderrHandler` 才分别创建并消费 `StdoutPipe` 或 `StderrPipe`；未配置的流保持调用方设置，Manager 不解析命令参数。relay 调用方负责配置 `-progress pipe:1`。
- stdout 和 stderr 必须并发持续消费；一个 pipe 等待数据或缓慢 callback 不得阻塞另一 pipe 的读取。Go 1.26 使用 `sync.WaitGroup.Go` 启动 reader，两个 reader 都结束后 watcher 唯一调用 `Wait()`。
- Start 成功后 watcher 唯一调用 `Wait()`；清理只删除与自身指针匹配的条目。自然退出、Stop、StopAll、父 context cancel 和 deadline 都会调用该进程的 exit hook，并以 `ExitResult` 交给业务层判断。
- builder 在登记前同步执行，只构造绑定 Manager 所传 context 的未启动命令；同一 ID 的 Start/Stop/StopAll 由调用方串行化。Manager 不维护 `starting` 状态或全局生命周期锁；不同 ID 独立。
- stdout/stderr 和 exit hook 同步执行。退出顺序为：两个 pipe reader 结束、`Wait`、身份清理、exit hook、关闭 `done`；锁不覆盖 builder、Start、cancel、Wait、日志或 hook。
- exit hook 执行时自身条目已清理，可以重入 Manager；stdout/stderr callback 仍处于该进程 pipe reader，不能同步 Stop 或替换自身 ID，否则会等待自身 reader 完成。
- 业务层 registry 只停止自己登记的 ID；服务全局关闭由共享 `Manager.StopAll` 负责。

### 4. Validation & Error Matrix

- nil context、空 ID、nil builder -> Start 返回参数错误，不登记进程。
- builder、StdoutPipe、StderrPipe 或 cmd.Start 失败 -> 取消 context、关闭已建 pipe且不保留 Manager 条目，并返回包装错误。
- 同 ID 已存在 -> 返回包装 `ErrProcessExists`，现有进程不受影响，且不得调用新 builder；替换由调用方先 `Stop` 再 `Start`。
- 主动停止与 Wait 同时发生 -> process context 的取消状态决定退出原因，不重复 Wait；exit hook 仍以 `ExitResult` 报告。
- 子进程自行成功或失败退出 -> 先删除自身登记，再调用该进程的 exit hook。

### Scenario: `ffmpegx.Manager` exit and stdout contract

#### Signatures

```go
var ErrProcessExists error
type ExitResult struct { ID string; WaitErr error; ContextErr error }
func WithExitHandler(func(ExitResult)) StartOption
func WithStdoutHandler(func(id, line string)) StartOption
func WithStderrHandler(func(id, line string)) StartOption
func WatchOutput(io.ReadCloser, func(line string)) error
```

- `Start` returns an error wrapping `ErrProcessExists` for a duplicate ID and leaves the existing process untouched.
- Every started process invokes its exit callback synchronously, including natural exit, cancellation, Stop, and deadline. `WaitErr` is the raw `Cmd.Wait` result; `ContextErr` is the process context error. `done` closes only after the callback returns.
- stdout/stderr is consumed only when its matching handler is configured. `WatchOutput` forwards each scanner line without parsing or aggregation; it is valid for either pipe.
- Long-running commands must stream stderr through `WithStderrHandler`, never retain it in an unbounded `bytes.Buffer`.
- Process lifecycle logs use `logx.WithContext(processCtx)` and the `[ffmpegx]` prefix.

### 5. Good/Base/Bad Cases

- Good：需要进度的命令显式把 `-progress` 指向 stdout，业务层按行聚合并在完整报告边界续租；stderr 逐行交给业务日志或告警处理。
- Base：无 stdout/stderr handler 的普通命令保留调用方配置，Manager 不创建对应 pipe。
- Bad：串行读取 stdout 后才读取 stderr；stdout 沉默时 stderr pipe 无人消费，最终可能阻塞子进程。

### 6. Tests Required

- 覆盖 Start/Has/Count、快速退出、builder/pipe/Start 失败、Stop/StopAll/context cancel/timeout、重复 ID 不调用 builder、同步 callback 和不同 ID 非阻塞。
- 覆盖显式 stdout/stderr 消费、两路并发读取，以及无 handler 时保留调用方对应流配置。
- registry 覆盖 metadata 先登记、exit 先清 metadata 后 hook、旧 exit 不删 replacement、StopAll 只停自身 target。
- 对 `common/ffmpegx` 和直接 registry 调用方运行 `go test -race`。

### 7. Wrong vs Correct

```go
// Wrong: stdout reader 阻塞时，stderr 永远不会被消费。
WatchOutput(stdout, onStdout)
WatchOutput(stderr, onStderr)

// Correct: both pipes are continuously consumed before Wait observes exit.
var readers sync.WaitGroup
readers.Go(func() { _ = WatchOutput(stdout, onStdout) })
readers.Go(func() { _ = WatchOutput(stderr, onStderr) })
readers.Wait()

// Duplicate IDs are explicit and exit reason is structured.
m.Start(ctx, id, build,
	ffmpegx.WithExitHandler(onExit),
	ffmpegx.WithStdoutHandler(onLine),
	ffmpegx.WithStderrHandler(onLine),
)
```

依据：`common/ffmpegx/process.go`、`app/oryxserver/internal/relay/registry.go`。

## 验证

- 覆盖取消、超时、panic、重复完成、关闭、快速响应、过期和提交失败。
- 使用有上限轮询或可控同步点验证异步结果。
- 对目标并发包运行 `go test -race -count=10 ./path/to/package`，并检查测试断言真实竞争条件。
