# Design: 合并 RelayRegistry 和 DistributedRelay

## 现状

两个 struct 双向耦合：

```
RelayRegistry (本地进程 + metadata)     DistributedRelay (Redis + Asynq)
  ├─ meta map                            ├─ Store (Redis state/lease)
  ├─ ffmpegx.Manager                     ├─ asynq.Client
  ├─ onExit hook ─────(回调)───────────→ OnProcessExit
  ├─ onProgress hook ──(回调)───────────→ OnProgress
  └─ handleExit/handleOutput             └─ Reconcile/EnqueueReconcile/EnqueueStop
```

调用链：ffmpeg process → `handleExit` → `onExit` → `OnProcessExit`，套了两层回调。

## 方案：合并为 RelayManager

将两个 struct 合并为一个 `RelayManager`，去掉 hook 中间层。

### 合并后的 struct

```go
type RelayManager struct {
    // 来自 RelayRegistry（本地进程管理）
    processes *ffmpegx.Manager
    buildCmd  relayCommandBuilder
    lifecycleMu sync.Mutex
    metaMu      sync.RWMutex
    meta        map[string]*pullMeta

    // 来自 DistributedRelay（分布式协调）
    store  *Store
    nodeID string
    asynq  *asynq.Client
}
```

### 方法映射

| 原方法 | 新方法 | 变化 |
|--------|--------|------|
| `RelayRegistry.StartRelay` | `RelayManager.startLocalRelay` | 改为内部方法 |
| `DistributedRelay.StartRelay` | `RelayManager.StartRelay` | 入口方法，内部直接调 `startLocalRelay` |
| `RelayRegistry.StopRelay` | `RelayManager.stopLocalRelay` | 改为内部方法 |
| `DistributedRelay.StopRelay` | `RelayManager.StopRelay` | 分布式停止 |
| `RelayRegistry.StopRelayByAppStream` | `RelayManager.StopRelayByAppStream` | 仅本地停止（广播用） |
| `RelayRegistry.StopAll` | `RelayManager.StopAll` | 仅本地停止（退场用） |
| `RelayRegistry.HasTarget` | `RelayManager.HasTarget` | 不变 |
| `DistributedRelay.Reconcile` | `RelayManager.Reconcile` | 不变 |
| `DistributedRelay.EnqueueReconcile` | `RelayManager.EnqueueReconcile` | 不变 |
| `DistributedRelay.EnqueueStop` | `RelayManager.EnqueueStop` | 不变 |
| `DistributedRelay.OnProgress` | `RelayManager.onProgress` | 改为内部方法，handleOutput 直接调 |
| `DistributedRelay.OnProcessExit` | `RelayManager.onProcessExit` | 改为内部方法，handleExit 直接调 |

### 删除的 API

- `SetExitHandler` / `SetProgressHandler` — 不再需要
- `onExit` / `onProgress` 字段 — 不再需要

### 调用方变更

**servicecontext.go**：
```go
// Before:
svcCtx.RelayRegistry = relay.NewRelayRegistry(svcCtx.FFmpegManager)
svcCtx.DistRelay = relay.NewDistributedRelay(store, registry, nodeID, asynqClient)
svcCtx.RelayRegistry.SetProgressHandler(...)
svcCtx.RelayRegistry.SetExitHandler(...)

// After:
svcCtx.RelayManager = relay.NewRelayManager(svcCtx.FFmpegManager, store, nodeID, svcCtx.AsynqClient)
// 无 hook 设置
```

**broadcast.go**：
```go
// Before: registry *relay.RelayRegistry
// After:  mgr *relay.RelayManager
// 调用不变：mgr.StopRelayByAppStream(app, stream)
```

**stoprelaypulllogic.go**：
```go
// Before: l.svcCtx.RelayRegistry.HasTarget / l.svcCtx.StateStore.GetState / ...
// After:  l.svcCtx.RelayManager.HasTarget / l.svcCtx.StateStore.GetState / ...
// StopRelay 调用可简化为 l.svcCtx.RelayManager.StopRelay(target)
```

**reconcile.go / task/stop.go**：
```go
// Before: h.svcCtx.DistRelay.Reconcile / .StopRelay
// After:  h.svcCtx.RelayManager.Reconcile / .StopRelay
```

### Broadcast 只依赖本地停止

`broadcast.go` 只需要 `StopRelayByAppStream`（本地停止），不需要分布式方法。

方案：`RelayManager` 的 `StopRelayByAppStream` 和 `StopAll` 保持只做本地停止（清 meta + 停进程），不触发分布式协调。这与现有行为一致。

### 测试变更

- `registry_test.go` 中 hook 相关测试（`SetExitHandler`/`SetProgressHandler`）需要重写
- 改为直接验证 `onProcessExit`/`onProgress` 被调用（通过 mock store/asynq）
- 或者保持集成测试风格：验证进程退出后 Redis state 的变化

### 文件变更清单

| 文件 | 操作 |
|------|------|
| `relay/registry.go` | 吸收 distributed.go 字段和方法，删除 hook |
| `relay/distributed.go` | 删除 |
| `relay/state.go` | 不变（hashKey 已改） |
| `relay/store.go` | 不变 |
| `relay/types.go` | 不变 |
| `svc/servicecontext.go` | 合并字段，删除 hook 注册 |
| `mqtt/broadcast.go` | 更新类型引用 |
| `logic/stoprelaypulllogic.go` | 更新类型引用 |
| `task/reconcile.go` | 更新类型引用 |
| `task/stop.go` | 更新类型引用 |
| `relay/registry_test.go` | 重写 hook 测试 |
| `relay/distributed_test.go` | 如果有，合并到 registry_test.go |

### 风险点

1. `stoprelaypulllogic.go` 的 `stopRelayPullOnce` 和 `DistributedRelay.StopRelay` 逻辑重复——合并后可以简化，但需确认广播分支行为不变
2. `Broadcast` 只注入 `RelayManager` 而非完整 `ServiceContext`——需保持窄接口设计
3. 测试中 hook 相关测试较多，重写工作量不小
