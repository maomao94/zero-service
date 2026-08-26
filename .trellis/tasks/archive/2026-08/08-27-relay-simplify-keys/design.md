# 技术设计：简化 Relay ID 设计

## 变更概览

| 组件 | 当前 | 变更后 |
|------|------|--------|
| Redis state key | `oryx:relay:state:{SHA256(target)}` | `oryx:relay:state:{target}` |
| Redis 反向索引 | `oryx:relay:idx:{target} → SHA256(target)` | **删除** |
| Redis lease key | `oryx:relay:lease:{target}` | 不变 |
| Manager meta key | `relayID (SHA256)` | `target` (明文) |
| Proto relay_id | `SHA256(target)` | `MD5(target)` |

## 数据流

### Start 流程
```
StartRelayPull(app, stream, source)
  │
  ├─ 构造 target = srsRtmpAddr + "/" + app + "/" + stream
  ├─ 构造 relayURL = target + "?" + authQuery
  │
  ├─ DistributedRelay.Start(source, target, relayURL)
  │     ├─ SaveState(target, {source, target, relayURL})  ← key = target
  │     ├─ HasLease(target)                                ← 不变
  │     ├─ TryClaim(target, nodeID, target)                ← relayID = target
  │     └─ Manager.StartPull(source, target, relayURL)
  │           ├─ meta[target] = {source, target, app, stream}  ← key = target
  │           └─ proc.Start(target, cmd, ctx, cancel)          ← id = target
  │
  └─ 返回 {relay_id: MD5(target), app, stream}
```

### Stop 流程
```
StopRelayPull(app, stream)
  │
  ├─ 构造 target = srsRtmpAddr + "/" + app + "/" + stream
  │
  ├─ 检查存在性
  │     ├─ RelayManager.HasTarget(target)     ← O(1) 直接查找 meta[target]
  │     └─ StateStore.GetState(target)         ← 直接用 target 作为 key
  │
  ├─ 删除 Redis
  │     ├─ DeleteState(target)                 ← 直接用 target，无需反向索引
  │     └─ DeleteLease(target)                 ← 不变
  │
  ├─ 停止本地进程
  │     └─ RelayManager.StopByTarget(target)   ← O(1) 直接查找 meta[target]
  │
  └─ 集群广播（可选）
```

## 文件变更

### 1. state.go
- 删除 `targetIndexKeyPrefix` 常量
- 删除 `targetIndexKey()` 函数
- 删除 `GetStateByTarget()` 方法
- 删除 `GetLeaseRelayID()` 方法
- 删除未使用的 `progressInterval` 常量
- `SaveState()`: 删除反向索引写入
- `DeleteState()`: 删除反向索引删除，参数改为 `target string`
- `GetState()`: 参数改为 `target string`（直接用 target 作为 key）

### 2. manager.go
- `meta` 类型从 `map[string]*relayMeta` 改为 key 是 target
- `StartPull()`: 使用 `target` 作为 key 存入 meta
- `HasTarget()`: 从遍历改为 `meta[target]` 直接查找
- `StopByTarget()`: 从遍历改为 `meta[target]` 直接查找
- `SetExitHandler/SetProgressHandler`: 回调参数从 relayID 改为 target

### 3. distributed.go
- `Start()`: `SaveState` 使用 target 作为 key
- `Start()`: `TryClaim` 的 relayID 参数改为 target
- `Reconcile()`: `GetState` 使用 target 作为 key
- `Reconcile()`: `TryClaim` 的 relayID 参数改为 target
- `Stop()`: `DeleteState` 使用 target（无需先查 relayID）
- 删除 `RelayID()` 调用（只在 proto 返回时使用）

### 4. startrelaypulllogic.go
- 修复日志格式：`target` 参数传 `target` 而不是 `relayURL`
- 删除重复注释
- 返回 `relay_id = MD5(target)`

### 5. stoprelaypulllogic.go
- 简化 state 操作：直接用 target 作为 key

### 6. proto (oryxserver.proto)
- 添加注释说明 relay_id = MD5(target)

## 兼容性

- **Redis 数据迁移**: 需要清空旧的 state key（SHA256 编码）和反向索引
- **Proto 兼容**: relay_id 格式变更，调用方需要适配
- **内部兼容**: 所有内部逻辑使用明文 target，无需适配

## 回滚方案

如果出现问题，可以：
1. 恢复 `RelayID()` 函数
2. 恢复反向索引
3. 恢复 `GetStateByTarget()` 等函数
4. 清空新的 Redis key，恢复旧 key
