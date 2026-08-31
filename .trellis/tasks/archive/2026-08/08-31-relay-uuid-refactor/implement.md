# 实施计划：中继逻辑 UUID 标识 + app:stream key

## 实施步骤

### Phase 1: 修改 state.go 核心结构

- [ ] 1.1 `RelayState` 新增 `UUID` 字段
- [ ] 1.2 `ReconcilePayload` 改为 `App + Stream + UUID`
- [ ] 1.3 `StopPayload` 新增 `UUID` 字段
- [ ] 1.4 修改 `stateKey()` 签名为 `stateKey(app, stream string)`
- [ ] 1.5 修改 `leaseKey()` 签名为 `leaseKey(app, stream string)`
- [ ] 1.6 修改 `lockKey()` 签名为 `lockKey(app, stream string)`
- [ ] 1.7 修改 `GetState()` 签名
- [ ] 1.8 修改 `SaveState()` 从 st.App, st.Stream 提取 key
- [ ] 1.9 修改 `DeleteState()` 签名
- [ ] 1.10 修改 `HasLease()` 签名
- [ ] 1.11 修改 `TryClaim()` 签名
- [ ] 1.12 修改 `Renew()` 签名
- [ ] 1.13 修改 `Release()` 签名
- [ ] 1.14 修改 `DeleteLease()` 签名
- [ ] 1.15 修改 `AddToRegistry()` 签名
- [ ] 1.16 修改 `RemoveFromRegistry()` 签名
- [ ] 1.17 修改 `RenewRegistry()` 签名
- [ ] 1.18 修改 `Lock()` 签名
- [ ] 1.19 修改 `ScanStale()` 使用 app:stream 作为 key

### Phase 2: 修改 registry.go 业务逻辑

- [ ] 2.1 `StartRelay()` - 生成 UUID，使用 app:stream key，校验已存在
- [ ] 2.2 `StopRelay()` - 使用 app:stream key
- [ ] 2.3 `Reconcile()` - 使用 app:stream key，校验 UUID
- [ ] 2.4 `releaseAndRetry()` - 使用 app:stream key
- [ ] 2.5 `defaultOnProgress()` - 使用 app:stream key
- [ ] 2.6 `defaultOnProcessExit()` - 使用 app:stream key
- [ ] 2.7 `cleanupTarget()` - 使用 app:stream key
- [ ] 2.8 `EnqueueReconcile()` - payload 使用 app:stream:uuid
- [ ] 2.9 `EnqueueStop()` - payload 使用 app:stream:uuid
- [ ] 2.10 `pullMeta` 结构体新增 `UUID` 字段
- [ ] 2.11 `startLocalRelay()` - 传递 UUID
- [ ] 2.12 `StopRelayByAppStream()` - 保持不变
- [ ] 2.13 `StopAll()` - 保持不变

### Phase 3: 修改 logic 层

- [ ] 3.1 `startrelaypulllogic.go` - 使用 app:stream key，传递 UUID
- [ ] 3.2 `stoprelaypulllogic.go` - 使用 app:stream key，传递 UUID

### Phase 4: 修改 task 层

- [ ] 4.1 `reconcile.go` - payload 解析使用 app:stream:uuid
- [ ] 4.2 `stop.go` - payload 解析使用 app:stream:uuid

### Phase 5: 清理和测试

- [ ] 5.1 移除 `CanonicalUID()` 函数（不再需要）
- [ ] 5.2 更新测试用例
- [ ] 5.3 运行测试验证

## 验证命令

```bash
# 运行 relay 包测试
go test ./app/oryxserver/internal/relay/...

# 运行 task 包测试
go test ./app/oryxserver/internal/task/...

# 编译检查
go build ./app/oryxserver/...
```

## 风险点

1. **key 格式变更**: 部署时需确保无正在运行的中继任务
2. **UUID 校验**: Asynq 回调时需正确传递 UUID
3. **向后兼容**: 旧 key 有 TTL，会自动清理
