# Implement: 合并 RelayRegistry 和 DistributedRelay

## 执行计划

### Step 1: 合并 struct 定义（registry.go）

1. 在 `RelayRegistry` struct 中添加 `store`, `nodeID`, `asynq` 字段
2. 修改 `NewRelayRegistry` 构造函数，接收 `store`, `nodeID`, `asynq` 参数
3. 删除 `onExit`/`onProgress` 字段
4. 删除 `SetExitHandler`/`SetProgressHandler` 方法

### Step 2: 合并方法（distributed.go → registry.go）

1. 将 `DistributedRelay` 的所有方法移到 `registry.go`，receiver 改为 `*RelayRegistry`
2. `DistributedRelay.StartRelay` → `RelayRegistry.StartRelay`（分布式入口）
   - 内部调用本地启动逻辑（原 `RelayRegistry.StartRelay` 改为 `startLocalRelay`）
3. `DistributedRelay.StopRelay` → `RelayRegistry.StopRelay`（分布式停止）
   - 内部调用 `stopLocalRelay`（原 `RelayRegistry.StopRelay`）
4. `DistributedRelay.OnProgress` → `RelayRegistry.onProgress`（内部方法）
5. `DistributedRelay.OnProcessExit` → `RelayRegistry.onProcessExit`（内部方法）
6. `DistributedRelay.Reconcile` → `RelayRegistry.Reconcile`
7. `DistributedRelay.EnqueueReconcile` → `RelayRegistry.EnqueueReconcile`
8. `DistributedRelay.EnqueueStop` → `RelayRegistry.EnqueueStop`
9. `DistributedRelay.releaseAndEnqueue` → `RelayRegistry.releaseAndEnqueue`

### Step 3: 修改 hook 调用（registry.go）

1. `handleOutput`: 直接调 `r.onProgress(ctx, md.Target)` 替代 `fn(ctx, md.Target)`
2. `handleExit`: 直接调 `r.onProcessExit(ctx, md.Target, result)` 替代 `fn(ctx, md.Target, result)`

### Step 4: 调整公共方法语义

1. `StopRelayByAppStream` — 保持只做本地停止（清 meta + 停进程）
2. `StopAll` — 保持只做本地停止
3. `HasTarget` — 不变
4. 新增 `stopLocalRelay` 内部方法（原 `RelayRegistry.StopRelay` 的逻辑）
5. 新增 `startLocalRelay` 内部方法（原 `RelayRegistry.StartRelay` 的逻辑）

### Step 5: 删除 distributed.go

1. 删除 `distributed.go` 文件

### Step 6: 更新 servicecontext.go

1. 删除 `DistRelay` 字段
2. 修改 `NewRelayRegistry` 调用，传入 `store`, `nodeID`, `asynqClient`
3. 删除 `SetProgressHandler`/`SetExitHandler` 注册代码
4. 更新所有 `svcCtx.DistRelay.XXX` → `svcCtx.RelayRegistry.XXX`

### Step 7: 更新调用方

1. `broadcast.go`: `*relay.RelayRegistry` 类型不变（已是）
2. `stoprelaypulllogic.go`: `l.svcCtx.DistRelay.EnqueueStop` → `l.svcCtx.RelayRegistry.EnqueueStop`
3. `reconcile.go`: `h.svcCtx.DistRelay.Reconcile` → `h.svcCtx.RelayRegistry.Reconcile`
4. `task/stop.go`: `h.svcCtx.DistRelay.StopRelay` → `h.svcCtx.RelayRegistry.StopRelay`

### Step 8: 更新测试

1. `registry_test.go`: 删除 hook 相关测试，或重写为直接验证内部方法调用
2. 如果有 `distributed_test.go`，合并到 `registry_test.go`

### Step 9: 验证

```bash
go build ./app/oryxserver/...
go test ./app/oryxserver/internal/relay/... -count=1
go vet ./app/oryxserver/...
```

## 回滚点

- Step 1-4 完成后如果编译失败，检查字段/方法签名冲突
- Step 6-7 完成后如果调用方报错，检查 ServiceContext 字段名
- 每个 Step 完成后可独立回滚（git stash / checkpoint）
