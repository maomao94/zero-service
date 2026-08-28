# 合并 RelayRegistry 和 DistributedRelay，去掉 hook 回调

## Goal

将 `RelayRegistry`（本地进程管理）和 `DistributedRelay`（分布式协调）合并为一个 struct，消除 `onExit`/`onProgress` hook 回调的间接层。

## Background

当前两个 struct 双向耦合：
- `RelayRegistry` 持有 `onExit`/`onProgress` hook
- `DistributedRelay` 通过 `SetExitHandler`/`SetProgressHandler` 注入回调
- 调用链：ffmpeg process → `handleExit` → `onExit` → `DistributedRelay.OnProcessExit`，套了两层

## Requirements

1. 合并为单一 struct（暂定名 `RelayManager`），包含两个 struct 的所有字段和方法
2. 删除 `onExit`/`onProgress` hook 字段和 `SetExitHandler`/`SetProgressHandler` 方法
3. `handleExit` 直接调用内部 `onProcessExit()` 方法，`handleOutput` 直接调用 `onProgress()`
4. 保持所有现有公共方法签名不变（`StartRelay`, `StopRelay`, `StopRelayByAppStream`, `StopAll`, `HasTarget`, `Reconcile`, `EnqueueReconcile`, `EnqueueStop`）
5. 保持 `StopRelayByAppStream` 和 `StopAll` 的语义：只做本地停止（清 meta + 停进程），不触发分布式协调
6. 更新所有调用方：`servicecontext.go`, `broadcast.go`, `stoprelaypulllogic.go`, `reconcile.go`, `task/stop.go`
7. 更新 `registry_test.go` 中的 hook 相关测试

## Acceptance Criteria

- [ ] `go build ./app/oryxserver/...` 通过
- [ ] `go test ./app/oryxserver/internal/relay/... -count=1` 通过
- [ ] `go vet ./app/oryxserver/...` 无新增警告
- [ ] `ServiceContext` 只保留一个 relay 字段（不再分别持有 `RelayRegistry` 和 `DistRelay`）
- [ ] `broadcast.go` 仍然只做本地停止（不触发分布式协调）
- [ ] `stoprelaypulllogic.go` 简化调用（用合并后的分布式 `StopRelay`）

## Constraints

- 不改变分布式锁、租约、补偿队列的现有行为
- 不改变 Redis state/lease key 结构
- `Broadcast` 只依赖本地停止能力（不暴露分布式方法给 mqtt）
