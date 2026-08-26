# Research: Relay Business Logic Review

- **Query**: 全面审查 app/oryxserver/ 服务的 relay 业务逻辑
- **Scope**: internal
- **Date**: 2026-08-27

## Findings

### 1. StartRelayPullLogic (`internal/logic/startrelaypulllogic.go`)

**无问题。** 逻辑清晰，参数校验完整。

- **参数校验**: source_url 为空返回 `Code__1_01_PARAM_MISSING` (line 49-51)
- **默认值**: app/stream 缺省时走配置或自动生成 (line 35-45)
- **鉴权拼接**: 使用 `url.QueryEscape` 安全拼接 (line 90)
- **target 构造**: `SrsRtmpAddr + "/" + app + "/" + stream` (line 53)，不含鉴权参数

**注意点**: `target` 在此处是 `rtmp://host:1935/live/stream` 格式（含 scheme），下游 `DistributedRelay.Start` 和 `PullRegistry.StartPull` 都会调用 `NormalizeTarget` 去掉 scheme，所以不会出现重复 key 问题。

### 2. StopRelayPullLogic (`internal/logic/stoprelaypulllogic.go`)

#### 问题 1: Redis 错误被静默忽略 (line 52)

```go
st, _ := l.svcCtx.StateStore.GetState(l.ctx, target)
```

`GetState` 的 error 被丢弃。如果 Redis 不可用，`st` 为 nil，`redisExists` 为 false，可能误判 relay 不存在而跳过停止。

**风险**: 低。Redis 不可用时 relay 本身也无法正常运行（租约/状态都无法管理），但语义上不够严谨。

#### 问题 2: DeleteState/DeleteLease 错误被忽略 (line 63-64)

```go
_ = l.svcCtx.StateStore.DeleteState(l.ctx, target)
_ = l.svcCtx.StateStore.DeleteLease(l.ctx, target)
```

如果删除失败，Redis 中残留状态会导致 Reconcile 任务补拉起新进程。

**风险**: 中。但考虑到 Stop 流程是"尽力而为"（后续有广播 + 补停兜底），这个设计可接受。

#### 问题 3: 广播超时只处理了 ErrReplyExpired (line 91)

```go
if errors.Is(broadcastErr, antsx.ErrReplyExpired) {
```

如果广播返回其他错误（如 MQTT 连接断开、publish 失败），直接返回 gRPC 错误，不入队补停。这意味着非超时类的广播失败没有兜底。

**风险**: 中。MQTT publish 失败时，目标节点的 relay 进程可能仍在运行，但没有补停机制。

#### 问题 4: 广播成功不代表目标进程已停止 (line 85-88)

`BroadcastReply` 返回 nil 只表示某个节点 ack 了请求，但 ack 是在 executor 调用 `StopPull` 后立即发出的（`mqtt/broadcast.go:50-63`）。如果 `StopPull` 返回 true（找到进程），但实际 `processes.Stop` 超时（3s `StopWaitTimeout`），ack 已经发出，调用方认为停止成功。

**风险**: 低。Manager.Stop 的 3s 超时后进程会被强制清理。

### 3. DistributedRelay.Start (`internal/relay/distributed.go:37-96`)

#### 问题 5: 获取锁失败时静默返回 target (line 48-51)

```go
if !ok {
    clog.Infof("lock held by another node, skip start: target=%s", target)
    return target, nil
}
```

调用方 `StartRelayPullLogic` 会认为启动成功（err==nil），返回 `RelayId`。但实际上可能没有任何节点在运行此 relay。

**场景**: 并发请求同一 target 时，第二个请求拿到锁失败，返回成功但 relay 可能尚未启动完成。

**风险**: 低。第一个请求持有锁时会完成启动流程。如果第一个请求失败，会 `releaseAndEnqueue` 入队补拉。

#### 问题 6: Start 中 "同源同目标 = no-op" 但不检查 source 是否一致 (line 54-69)

```go
previous, err := d.store.GetState(ctx, target)
if previous != nil {
    deadline = previous.DeadlineAtUnix
} else if maxDurationSeconds > 0 {
    deadline = time.Now().Add(time.Duration(maxDurationSeconds) * time.Second).Unix()
}
if err := d.store.SaveState(ctx, &RelayState{
    Source: source, Target: target, RelayURL: relayURL, DeadlineAtUnix: deadline,
}); err != nil {
```

如果已有状态但 source 不同，会覆盖写入新 source + 旧 deadline。注释说"幂等：同源同目标 = no-op"，但实际代码对不同 source 也是覆盖写入。

**风险**: 中。如果调用方期望"已存在时返回错误"，当前行为不符合预期。但作为"幂等创建"语义，覆盖写入是合理的。

### 4. DistributedRelay.Stop (`internal/relay/distributed.go:99-114`)

#### 问题 7: Stop 中 DeleteState/DeleteLease 错误被忽略 (line 111-112)

```go
_ = d.store.DeleteState(ctx, target)
_ = d.store.DeleteLease(ctx, target)
```

与 StopRelayPullLogic 中相同的问题。如果 Redis 删除失败，Reconcile 可能补拉。

**风险**: 同问题 2。

#### 问题 8: Stop 获取锁失败返回 false (line 105-108)

```go
if !ok {
    logx.WithContext(ctx).Infof("[relay] lock held by another node, skip stop: target=%s", target)
    return false
}
```

`DistributedRelay.Stop` 返回 false，但 `StopRelayPullLogic` 没有使用这个返回值（它直接调用 `StateStore.DeleteState/DeleteLease` + `RelayPulls.StopByTarget`，不经过 `DistRelay.Stop`）。

**实际上**: `DistributedRelay.Stop` 当前没有被任何地方调用！`StopRelayPullLogic` 直接操作 StateStore 和 RelayPulls，绕过了分布式协调器。

### 5. DistributedRelay.Reconcile (`internal/relay/distributed.go:117-170`)

**无问题。** 流程严谨：锁 → 读状态 → 检查过期 → 抢租约 → 复查状态 → 启动。双检（claim 前后各查一次）防止了与 Stop 的竞态。

#### 细节注意: Reconcile 中 GetState 错误返回 err (line 129-131)

```go
st, err := d.store.GetState(ctx, target)
if err != nil {
    return err
}
```

返回 err 会让 Asynq 认为任务失败并重试，这是正确的。

### 6. DistributedRelay.OnProcessExit (`internal/relay/distributed.go:203-213`)

#### 问题 9: DeadlineExceeded 时只删状态不释放租约 (line 204-208)

```go
if result.ContextErr == context.DeadlineExceeded {
    _ = d.store.DeleteState(ctx, target)
    _ = d.store.DeleteLease(ctx, target)
    return
}
```

实际上这里调用了 `DeleteLease`，所以没问题。但 `DeleteLease` 是无条件删除（不校验 nodeID），而 `Release` 是校验 nodeID 后删除。这里用 `DeleteLease` 是正确的，因为 deadline 到期是正常结束，不需要入队补拉。

**无问题。**

### 7. NormalizeTarget (`internal/relay/state.go:189-208`)

**无问题。** 覆盖了多种输入格式：
- 有 scheme: `url.Parse` 提取 host+path
- 无 scheme: 直接清理 query 参数和尾部 `/`

#### 细节: `url.Parse` 对 `rtmp://` scheme 的处理

Go 的 `url.Parse` 支持任意 scheme，`rtmp://host:1935/live/stream` 会被正确解析为 `Host=host:1935`, `Path=/live/stream`。

### 8. ParseTarget (`internal/relay/state.go:213-235`)

**无问题。** 校验严格：
- 必须有 `/`
- host 和 path 都不能为空
- path 必须恰好两段 (`/{app}/{stream}`)
- 每段不能为空

测试覆盖完整（`registry_test.go:244-292`）。

## Files Found

| File Path | Description |
|---|---|
| `app/oryxserver/internal/logic/startrelaypulllogic.go` | 启动 relay 的 gRPC handler |
| `app/oryxserver/internal/logic/stoprelaypulllogic.go` | 停止 relay 的 gRPC handler |
| `app/oryxserver/internal/relay/distributed.go` | 分布式协调器（Start/Stop/Reconcile） |
| `app/oryxserver/internal/relay/state.go` | Redis 状态/租约存储 + NormalizeTarget/ParseTarget |
| `app/oryxserver/internal/relay/registry.go` | PullRegistry 进程管理 |

## Summary of Issues

| # | 文件 | 行号 | 严重性 | 描述 |
|---|---|---|---|---|
| 1 | stoprelaypulllogic.go | 52 | 低 | GetState error 被忽略 |
| 2 | stoprelaypulllogic.go | 63-64 | 中 | DeleteState/DeleteLease 错误被忽略，可能导致 Reconcile 补拉 |
| 3 | stoprelaypulllogic.go | 91 | 中 | 广播非超时错误无补停兜底 |
| 4 | stoprelaypulllogic.go | 85-88 | 低 | 广播 ack 不代表进程已停止 |
| 5 | distributed.go | 48-51 | 低 | 锁竞争失败静默返回成功 |
| 6 | distributed.go | 54-69 | 中 | 不同 source 覆盖写入，与注释"幂等"语义不完全一致 |
| 7 | distributed.go | 111-112 | 中 | Stop 中 DeleteState/DeleteLease 错误被忽略 |
| 8 | distributed.go | 99-114 | 信息 | DistributedRelay.Stop 未被调用 |
| 9 | distributed.go | 204-208 | 无 | DeadlineExceeded 处理正确 |

## Caveats / Not Found

- `DistributedRelay.Stop` 方法存在但未被调用。`StopRelayPullLogic` 直接操作 StateStore 和 RelayPulls，绕过了分布式协调器的锁保护。
