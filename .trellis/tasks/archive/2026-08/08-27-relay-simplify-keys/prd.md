# 简化 Relay ID 设计：明文 target 作为 Redis key，删除反向索引

## Goal

简化 Redis key 设计和内部 ID 管理，消除不必要的编码和反向索引，降低代码复杂度。

## Requirements

### 1. Redis key 简化
- State key 从 `oryx:relay:state:{SHA256(target)}` 改为 `oryx:relay:state:{target}`（明文）
- 删除反向索引 `oryx:relay:idx:{target}` — 不再需要
- Lease key `oryx:relay:lease:{target}` 保持不变

### 2. 内部 ID 简化
- Manager meta key 从 `relayID (SHA256)` 改为 `target`（明文）
- `HasTarget` 从 O(n) 遍历改为 O(1) 直接查找
- `StopByTarget` 从 O(n) 遍历改为 O(1) 直接查找

### 3. Proto relay_id
- relay_id 改为 `MD5(target)` — 给调用方的短标识
- 内部存储使用明文 target — 方便解析业务 key

### 4. 代码清理
- 删除 `RelayID()` 在内部逻辑中的调用（只保留给 proto 返回用）
- 删除 `GetStateByTarget()` — 直接用 `GetState(target)`
- 删除 `GetLeaseRelayID()` — 不再需要
- 删除 `targetIndexKey` 及相关函数
- 删除未使用的 `progressInterval` 常量

## Acceptance Criteria

- [ ] Redis state key 使用明文 target
- [ ] 反向索引已删除
- [ ] Manager meta key 使用明文 target
- [ ] HasTarget/StopByTarget 为 O(1) 查找
- [ ] Proto relay_id = MD5(target)
- [ ] 所有现有测试通过
- [ ] go build ./... 通过
- [ ] go vet ./... 通过

## Notes

- 这是一个重构任务，不改变业务逻辑
- 所有现有的 Start/Stop/Reconcile 流程保持不变
- 只是简化内部 ID 管理和 Redis key 设计
