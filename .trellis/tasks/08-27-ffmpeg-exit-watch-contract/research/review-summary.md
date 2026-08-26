# Research: Code Review Summary

- **Query**: app/oryxserver/ 服务全面审查报告
- **Scope**: internal
- **Date**: 2026-08-27

## 审查结论

整体代码质量较高，架构设计合理。发现 **3 个中等问题** 和 **4 个低风险问题**，无阻塞性问题。

---

## 关键问题（需关注）

### ⚠️ 问题 1: StopRelayPullLogic 绕过分布式锁保护

**文件**: `stoprelaypulllogic.go` 全文
**严重性**: 中

`StopRelayPullLogic` 直接操作 `StateStore.DeleteState/DeleteLease` + `RelayPulls.StopByTarget`，未调用 `DistributedRelay.Stop`（后者会获取分布式锁）。

**竞态场景**:
1. StopRelayPullLogic: GetState → 存在
2. Reconcile (Asynq): 获取锁 → GetState → 存在 → 抢租约 → 启动进程
3. StopRelayPullLogic: DeleteState + DeleteLease（删掉了 Reconcile 的状态和租约）
4. Reconcile: StartPull 成功，进程在运行
5. StopRelayPullLogic: StopByTarget → 如果在同一节点，停止进程

**影响**: 小概率出现 Reconcile 补拉的进程被 Stop 删掉状态后无人管理。

**修复方向**: 让 StopRelayPullLogic 使用 `DistributedRelay.Stop`（带锁保护），或在 StopRelayPullLogic 中手动获取锁。

---

### ⚠️ 问题 2: 广播非超时错误无补停兜底

**文件**: `stoprelaypulllogic.go:91`
**严重性**: 中

```go
if errors.Is(broadcastErr, antsx.ErrReplyExpired) {
    // 只有超时才入队补停
    l.svcCtx.DistRelay.EnqueueStop(...)
}
// 其他错误直接返回 gRPC 错误
```

如果广播返回非超时错误（如 MQTT publish 失败），目标节点的 relay 进程可能仍在运行，但没有补停机制。

**修复方向**: 所有广播失败都入队补停，或至少区分可重试/不可重试错误。

---

### ⚠️ 问题 3: StopHandler 广播超时触发无意义重试

**文件**: `task/stop.go:59-62`
**严重性**: 中

```go
if broadcastErr != nil {
    return broadcastErr  // 触发 Asynq 重试
}
```

如果目标 relay 已停止（所有节点都没有该进程），BroadcastReply 超时返回 `ErrReplyExpired`，StopHandler 返回 error 触发 Asynq 重试。重试 25 次后才放弃。

**修复方向**: 检查 `errors.Is(broadcastErr, antsx.ErrReplyExpired)`，超时返回 nil 或 `asynq.SkipRetry`。

---

## 低风险问题

### 问题 4: GetState error 被忽略

**文件**: `stoprelaypulllogic.go:52`
**严重性**: 低

```go
st, _ := l.svcCtx.StateStore.GetState(l.ctx, target)
```

Redis 不可用时 `st` 为 nil，可能误判 relay 不存在。但 Redis 不可用时 relay 本身也无法正常运行。

---

### 问题 5: DeleteState/DeleteLease 错误被忽略

**文件**: `stoprelaypulllogic.go:63-64`, `distributed.go:111-112`
**严重性**: 低

删除失败时 Redis 中残留状态，可能导致 Reconcile 补拉。但 Stop 流程是"尽力而为"，有广播+补停兜底。

---

### 问题 6: Renew/Release 非原子操作

**文件**: `state.go:135-149`, `state.go:154-166`
**严重性**: 低

GET+EXPIRE / GET+DEL 非原子，小概率给别人的租约续期或误删新持有者的租约。竞态窗口毫秒级，且有 Reconcile 兜底。

---

### 问题 7: Start 中不同 source 覆盖写入

**文件**: `distributed.go:54-69`
**严重性**: 低（或设计如此）

注释说"幂等：同源同目标 = no-op"，但实际对不同 source 也是覆盖写入。如果业务语义是"不允许覆盖"，需要加 source 校验。

---

## 无问题的模块

| 模块 | 结论 |
|---|---|
| NormalizeTarget / ParseTarget | 正确，测试覆盖完整 |
| Reconcile 流程 | 严谨，双检防止竞态 |
| Broadcast executor | 不会递归调用，设计明确 |
| 分布式锁 TTL (15s) | 合理，覆盖关键段操作 |
| 租约 TTL (30s) | 合理，progress 帧每 5s 续一次 |
| PullRegistry 并发安全 | lifecycleMu → metaMu 顺序一致，无死锁 |
| Manager 进程生命周期 | metadata 先删、进程后停，handleExit 用指针比较防误删 |
| Asynq 任务幂等 | TaskID 去重，重复入队安全 |

---

## 文件清单

| 文件 | 行数 | 审查结论 |
|---|---|---|
| `internal/logic/startrelaypulllogic.go` | 91 | 无问题 |
| `internal/logic/stoprelaypulllogic.go` | 101 | 3 个中等问题 |
| `internal/relay/registry.go` | 232 | 无问题 |
| `internal/relay/distributed.go` | 247 | 1 个中等问题 + 1 个低风险 |
| `internal/relay/state.go` | 235 | 2 个低风险 |
| `internal/task/routes.go` | 22 | 无问题 |
| `internal/task/reconcile.go` | 44 | 无问题 |
| `internal/task/stop.go` | 65 | 1 个中等问题 |
| `mqtt/broadcast.go` | 64 | 无问题 |
