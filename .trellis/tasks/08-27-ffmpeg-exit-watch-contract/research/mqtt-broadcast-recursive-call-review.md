# Research: MQTT Broadcast Recursive Call Review

- **Query**: 确认 broadcast executor 不会递归调用 StopRelayPullLogic
- **Scope**: internal
- **Date**: 2026-08-27

## Findings

### 1. Broadcast Executor (`mqtt/broadcast.go`)

**不会递归调用。** 设计明确避免了这个问题。

#### 关键设计 (line 19-20, 38-41)

```go
// 注意：executor 内不调用带「未命中再广播」分支的 Logic，避免集群消息风暴；
// 仅注入最小依赖（PullRegistry），不注入 ServiceContext。
type Broadcast struct {
    pulls *relay.PullRegistry
}
```

executor 只持有 `PullRegistry`，不持有 `ServiceContext`、`Broadcaster`、`MqttClient` 等依赖。物理上不可能调用 `StopRelayPullLogic` 或发起广播。

#### executor 调用链

```
mqtt/broadcast.go:stopRelayPull
  → PullRegistry.StopPull(app, stream)  // 只停本地进程
  → 返回 (result, nil) 或 (nil, ErrSkipAck)
```

`PullRegistry.StopPull` (registry.go:151-175) 只做：
1. 遍历 `r.meta` 匹配 app+stream
2. 删除 metadata
3. 调用 `r.processes.Stop(target)` 停止 ffmpeg 进程

不涉及任何网络调用、广播、Redis 操作。

### 2. 防回环机制 (`common/mqttx/broadcast/dispatcher.go`)

除了 executor 层面的隔离，广播框架本身也有防回环：

```go
// line 34-41
if body.AckTopic == b.ackTopic() {
    logx.WithContext(ctx).Debugw("mqtt broadcast loopback ignored", ...)
    return nil
}
```

如果广播消息的 `AckTopic` 指向本实例，直接忽略。这意味着即使 executor 误发广播，本实例也不会消费自己的消息。

### 3. Ack 语义 (`mqtt/broadcast.go:50-63`)

```go
found := b.pulls.StopPull(in.App, in.Stream)
if !found {
    return nil, broadcast.ErrSkipAck  // 不回 ack，让其他节点处理
}
// 找到并停止 → 回 ack 成功
resJson, _ := protojson.Marshal(&oryxserver.StopRelayPullRes{})
return resJson, nil
```

- **找到进程**: 停止后回 ack 成功
- **未找到进程**: 返回 `ErrSkipAck`，不回 ack

`ErrSkipAck` 在 dispatcher 中被捕获 (line 58-65)，不发 ack。这意味着：
- 如果某个节点持有进程，它会回 ack
- 如果所有节点都没有进程，没有节点回 ack，调用方超时

## Files Found

| File Path | Description |
|---|---|
| `app/oryxserver/mqtt/broadcast.go` | 集群广播 executor |
| `common/mqttx/broadcast/dispatcher.go` | 广播消费分发（防回环 + ack） |
| `common/mqttx/broadcast/client.go` | 广播客户端（BroadcastReply 等待 ack） |

## Summary of Issues

**无问题。** broadcast executor 不会递归调用 StopRelayPullLogic。设计上有两层保护：
1. executor 只持有 PullRegistry 最小依赖，物理上不可能发起广播
2. 广播框架有 AckTopic 防回环机制

## Caveats / Not Found

- 所有节点都没有目标进程时，BroadcastReply 超时返回 ErrReplyExpired。调用方（StopRelayPullLogic）将此视为需要入队补停的信号，但实际上进程已不存在。StopHandler 重试也会遇到同样问题。
