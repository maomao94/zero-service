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

## 验证

- 覆盖取消、超时、panic、重复完成、关闭、快速响应、过期和提交失败。
- 使用有上限轮询或可控同步点验证异步结果。
- 对目标并发包运行 `go test -race -count=10 ./path/to/package`，并检查测试断言真实竞争条件。
