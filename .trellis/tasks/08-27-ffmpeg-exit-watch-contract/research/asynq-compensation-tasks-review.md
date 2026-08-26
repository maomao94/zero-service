# Research: Asynq Compensation Tasks Review

- **Query**: 审查 Asynq 补偿任务处理（Reconcile/Stop handlers）
- **Scope**: internal
- **Date**: 2026-08-27

## Findings

### 1. Task Routes (`internal/task/routes.go`)

**无问题。** 注册了两个任务类型：
- `RelayReconcileTask` → `ReconcileHandler`
- `RelayStopTask` → `StopHandler`

队列隔离：使用独立队列 `oryx-relay`，与 trigger 的 critical/default/low 完全隔离。

### 2. ReconcileHandler (`internal/task/reconcile.go`)

**无问题。**

- **payload 校验**: 反序列化失败返回 `asynq.SkipRetry`（不重试，payload 损坏重试无意义）(line 27)
- **空 target 校验**: 返回 `asynq.SkipRetry` (line 36)
- **业务错误传播**: `Reconcile` 返回的 err 直接传播给 Asynq，Asynq 默认重试策略生效 (line 38-41)
- **日志完整**: 包含 target、taskType、taskId 字段 (line 29-33)

#### 关于 Asynq 默认重试策略

Asynq 默认 `MaxRetry = 25`，重试间隔指数退避（1s, 8s, 27s, ... 最大 1day）。对于 Reconcile 任务：
- Redis 不可用 → 重试有意义
- 状态已删除 → Reconcile 返回 nil（不重试）
- ffmpeg 启动失败 → 重试有意义

**结论**: 默认重试策略合理。

### 3. StopHandler (`internal/task/stop.go`)

#### 问题 1: 广播失败直接返回 err，会触发 Asynq 重试 (line 59-62)

```go
if broadcastErr != nil {
    logger.Errorf("broadcast stop failed: %v", broadcastErr)
    return broadcastErr
}
```

这里没有区分错误类型：
- **MQTT 连接断开**: 重试有意义
- **所有节点都不持有该进程**（所有 executor 返回 `ErrSkipAck`）: 广播框架会怎么处理？

检查 `dispatcher.go:57-78`：如果 executor 返回 `ErrSkipAck`，dispatcher 不发 ack。如果所有节点都返回 `ErrSkipAck`，`BroadcastReply` 会超时（10s timeout），返回 `ErrReplyExpired`。

但 StopHandler 没有检查 `ErrReplyExpired`！超时错误会触发 Asynq 重试，重试后又超时，循环 25 次。

**风险**: 中。如果目标 relay 已经停止（所有节点都没有该进程），补停任务会重试 25 次后才放弃。

**修复建议**: 检查 `errors.Is(broadcastErr, antsx.ErrReplyExpired)`，如果是超时，返回 nil（认为已停止）或 `asynq.SkipRetry`。

#### 问题 2: 补停任务不删 Redis 状态 (line 48-64)

```go
stopReq := &oryxserver.StopRelayPullReq{App: payload.App, Stream: payload.Stream}
payloadBytes, err := protojson.Marshal(stopReq)
// ... broadcast ...
```

补停任务只做广播停止，不负责清理 Redis 状态。这是正确的——`StopRelayPullLogic` 在广播前已经删除了状态 (line 62-64)。

**无问题。**

#### 问题 3: 补停任务 payload 中的 Target 字段未使用 (line 27, 48)

```go
var payload relay.StopPayload  // {Target, App, Stream}
stopReq := &oryxserver.StopRelayPullReq{App: payload.App, Stream: payload.Stream}
```

`StopPayload.Target` 在 StopHandler 中未使用，只用了 App 和 Stream。这是因为广播停止是按 app+stream 匹配（`PullRegistry.StopPull`），不是按 target。

**无问题。** Target 字段是 `EnqueueStop` 写入的冗余信息，可能用于日志但当前未使用。

### 4. EnqueueReconcile / EnqueueStop (`internal/relay/distributed.go`)

**无问题。**

- **幂等去重**: 使用 `asynq.TaskID(RelayTaskPrefix+target)` 和 `asynq.TaskID(RelayTaskPrefix+"stop:"+target)` 作为 TaskID (line 180, 240)
- **重复入队**: `asynq.ErrDuplicateTask` 被捕获并忽略 (line 183, 243)
- **队列指定**: 使用 `asynq.Queue(RelayQueue)` 路由到隔离队列

#### 细节: Reconcile 和 Stop 的 TaskID 前缀不同

- Reconcile: `oryx:relay:host:1935/live/stream`
- Stop: `oryx:relay:stop:host:1935/live/stream`

不会冲突。

### 5. Asynq Server 配置 (`common/asynqx/asynqTaskServer.go`)

#### 问题 4: IsFailure 始终返回 true (line 80)

```go
IsFailure: func(err error) bool { return true },
```

所有返回的 error 都被视为失败，触发重试。这意味着 `Reconcile` 返回的业务错误（如 "start ffmpeg failed"）会被重试。

对于 Reconcile 任务这是合理的（ffmpeg 可能暂时不可用）。
对于 Stop 任务，广播超时被重试是不合理的（见问题 1）。

#### 问题 5: 未配置 RetryDelay

使用 Asynq 默认的指数退避策略。对于 Stop 任务，如果目标已停止，每次重试间隔越来越长（最长 1 天），任务会挂起很久。

**风险**: 低。TaskID 去重意味着不会有重复任务堆积，但单个任务可能挂 25 次重试周期。

## Files Found

| File Path | Description |
|---|---|
| `app/oryxserver/internal/task/routes.go` | 任务注册 |
| `app/oryxserver/internal/task/reconcile.go` | 补拉 handler |
| `app/oryxserver/internal/task/stop.go` | 补停 handler |
| `common/asynqx/asynqTaskServer.go` | Asynq server 配置 |
| `common/asynqx/asynqClient.go` | Asynq client/inspector |

## Summary of Issues

| # | 文件 | 行号 | 严重性 | 描述 |
|---|---|---|---|---|
| 1 | task/stop.go | 59-62 | 中 | 广播超时（所有节点都没有该进程）触发无意义重试 |
| 4 | asynqx/asynqTaskServer.go | 80 | 信息 | IsFailure 始终返回 true，对 Stop 任务的超时场景不友好 |

## Caveats / Not Found

- StopHandler 没有区分 `ErrReplyExpired` 和其他广播错误，导致超时场景重试 25 次。
