# 执行计划：简化 Relay ID 设计

## 执行步骤

### Step 1: 修改 state.go
- 删除 `targetIndexKeyPrefix` 常量
- 删除 `targetIndexKey()` 函数
- 删除 `progressInterval` 常量
- 修改 `SaveState()`: 删除反向索引写入
- 修改 `DeleteState()`: 参数改为 `target string`，直接删除 state key
- 修改 `GetState()`: 参数改为 `target string`（直接用 target 作为 key）
- 删除 `GetStateByTarget()` 方法
- 删除 `GetLeaseRelayID()` 方法

### Step 2: 修改 manager.go
- 修改 `meta` 注释：key 从 relayID 改为 target
- 修改 `StartPull()`: 使用 `target` 作为 key 存入 meta
- 修改 `HasTarget()`: 从遍历改为 `meta[target]` 直接查找
- 修改 `StopByTarget()`: 从遍历改为 `meta[target]` 直接查找
- 修改 `SetExitHandler`: 回调参数从 relayID 改为 target
- 修改 `SetProgressHandler`: 回调参数从 relayID 改为 target

### Step 3: 修改 distributed.go
- 修改 `Start()`: `SaveState` 使用 target 作为 key
- 修改 `Start()`: `TryClaim` 的 relayID 参数改为 target
- 修改 `Reconcile()`: `GetState` 使用 target 作为 key
- 修改 `Reconcile()`: `TryClaim` 的 relayID 参数改为 target
- 修改 `Stop()`: `DeleteState` 使用 target（无需先查 relayID）
- 删除所有 `RelayID()` 调用（只在 proto 返回时使用）

### Step 4: 修改 startrelaypulllogic.go
- 修复日志格式：`target` 参数传 `target` 而不是 `relayURL`
- 删除重复注释
- 导入 `crypto/md5` 和 `encoding/hex`
- 修改返回值：`relay_id = MD5(target)`

### Step 5: 修改 stoprelaypulllogic.go
- 简化 state 操作：直接用 target 作为 key
- 删除 `st.RelayID` 引用

### Step 6: 添加 proto 注释
- 在 `StartRelayPullRes` 的 `relay_id` 字段添加注释说明编码逻辑

### Step 7: 验证
- `go build ./...`
- `go vet ./...`
- `go test ./common/ffmpegx/...`
- `git diff --check`

## 验证命令

```bash
# 构建验证
go build ./app/oryxserver/... ./common/ffmpegx/...

# 静态检查
go vet ./app/oryxserver/... ./common/ffmpegx/...

# 测试验证
go test ./common/ffmpegx/... -v -count=1

# Git 检查
git diff --check
```

## 回滚点

如果任何步骤失败，可以：
1. `git checkout -- .` 恢复所有文件
2. 重新评估设计方案
