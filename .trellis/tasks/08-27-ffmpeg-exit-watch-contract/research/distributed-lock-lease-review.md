# Research: Distributed Lock + Lease Review

- **Query**: 审查分布式锁 + 租约管理的正确性
- **Scope**: internal
- **Date**: 2026-08-27

## Findings

### 1. Store.Lock — Per-target 分布式锁 (`internal/relay/state.go:176-184`)

```go
func (s *Store) Lock(ctx context.Context, target string) (*redis.RedisLock, bool, error) {
    lock := redis.NewRedisLock(s.r, lockKeyPrefix+target)
    lock.SetExpire(lockTTL)  // 15 秒
    ok, err := lock.AcquireCtx(ctx)
    if err != nil {
        return nil, false, fmt.Errorf("acquire lock: %w", err)
    }
    return lock, ok, nil
}
```

**无问题。** 使用 go-zero 的 `redis.RedisLock`，底层是 `SET NX EX` 原子操作。

#### 锁 TTL 分析

- `lockTTL = 15` 秒
- 保护的操作：read-state → claim-lease → start-local（最多一次 ffmpeg 启动）
- ffmpeg 启动是本地 fork 进程，通常毫秒级完成
- 15 秒足够覆盖网络延迟 + Redis 操作 + 进程启动

**结论**: TTL 合理。

#### 锁释放

所有锁持有者都使用 `defer lock.ReleaseCtx(context.Background())` 释放：
- `distributed.go:52` — Start
- `distributed.go:109` — Stop
- `distributed.go:126` — Reconcile

使用 `context.Background()` 释放是正确的——即使请求 ctx 已取消，仍需释放锁。

**无问题。**

### 2. Store.TryClaim — 租约抢占 (`internal/relay/state.go:126-129`)

```go
func (s *Store) TryClaim(ctx context.Context, target, nodeID string) (bool, error) {
    data, _ := json.Marshal(leaseValue{NodeID: nodeID})
    return s.r.SetnxExCtx(ctx, leaseKey(target), string(data), int(leaseTTL/time.Second))
}
```

**无问题。** `SET NX EX` 原子操作，30 秒 TTL。

#### 租约 TTL 分析

- `leaseTTL = 30` 秒
- 续租方式：ffmpeg progress 帧回调（每 `RelayProgressInterval=5` 秒一次）
- 续租失败（租约被抢/被删）→ 停止本地进程

**场景分析**:
- ffmpeg 正常运行: 每 5 秒续一次，30 秒 TTL 内至少续 5 次
- ffmpeg 卡死: 不产生 progress 帧，30 秒后租约过期，Reconcile 可补拉
- ffmpeg 进程崩溃: `OnProcessExit` 释放租约 + 入队补拉

**结论**: TTL 合理。

### 3. Store.Renew — 续租 (`internal/relay/state.go:134-150`)

```go
func (s *Store) Renew(ctx context.Context, target, nodeID string) (bool, error) {
    val, err := s.r.GetCtx(ctx, leaseKey(target))
    // ... 校验 nodeID ...
    if err := s.r.ExpireCtx(ctx, leaseKey(target), int(leaseTTL/time.Second)); err != nil {
        return false, err
    }
    return true, nil
}
```

#### 问题 1: GET + EXPIRE 非原子操作 (line 135-149)

注释已说明：
> 中间被抢占只相当于给活着的持有者续了几秒（无害）；
> 中间被删除则 EXPIRE 失败返回 false，持有方据此停止本地进程。

**竞态场景**:
1. 节点 A: GET → 确认自己是持有者
2. 节点 A 的租约过期
3. 节点 B: SET NX → 抢占成功
4. 节点 A: EXPIRE → 给节点 B 的租约续了 30 秒

**影响**: 节点 B 的租约被多续了 30 秒。但节点 B 本身也会每 5 秒续一次，所以实际影响是节点 B 的租约 TTL 被重置为 30 秒（本来可能快过期了）。

**风险**: 低。最坏情况是节点 B 的租约延迟 30 秒过期。如果节点 B 还活着，它自己会续租；如果节点 B 已死，30 秒后租约自然过期。

**无问题。** 注释中的分析是正确的。

### 4. Store.Release — 释放租约 (`internal/relay/state.go:153-167`)

```go
func (s *Store) Release(ctx context.Context, target, nodeID string) (bool, error) {
    val, err := s.r.GetCtx(ctx, leaseKey(target))
    // ... 校验 nodeID ...
    _, err = s.r.DelCtx(ctx, leaseKey(target))
    return err == nil, err
}
```

#### 问题 2: GET + DEL 非原子操作 (line 154-166)

**竞态场景**:
1. 节点 A: GET → 确认自己是持有者
2. 节点 A 的租约过期
3. 节点 B: SET NX → 抢占成功
4. 节点 A: DEL → 删掉了节点 B 的租约

**影响**: 节点 B 的租约被误删。节点 B 在下次续租时发现租约丢失，会停止本地进程。

**风险**: 低。节点 B 的进程会被误停，但 Reconcile 会补拉。这个竞态窗口很小（GET 和 DEL 之间通常毫秒级）。

**与注释一致**: "仅释放自己的租约，避免误删新持有者"——实际上非原子操作有小概率误删。

### 5. Store.DeleteLease — 无条件删除 (`internal/relay/state.go:170-173`)

```go
func (s *Store) DeleteLease(ctx context.Context, target string) error {
    _, err := s.r.DelCtx(ctx, leaseKey(target))
    return err
}
```

**设计正确。** 用于 `StopRelayPullLogic` 和 `DistributedRelay.Stop`，语义是"无论谁持有都清掉"。

### 6. 锁释放 + 租约续期的 TTL 关系

| 操作 | TTL | 续期方式 |
|---|---|---|
| 分布式锁 | 15s | 不续期，操作完成后立即释放 |
| 租约 | 30s | 每 5s（progress 帧）续一次 |

**锁 TTL < 租约 TTL**: 正确。锁只保护关键段（毫秒级），租约保护进程生命周期（长时运行）。

### 7. 停止流程中的锁+租约交互

`StopRelayPullLogic` 的流程：
1. 检查本地进程 + Redis 状态（无锁）
2. 删除 Redis 状态 + 租约（无锁）
3. 停止本地进程

**问题 3: Stop 流程没有获取分布式锁**

`StopRelayPullLogic` 直接操作 StateStore 和 RelayPulls，不经过 `DistributedRelay.Stop`（后者会获取锁）。

**竞态场景**:
1. StopRelayPullLogic: GetState → 存在
2. Reconcile (Asynq): 获取锁 → GetState → 存在 → 抢租约 → 启动进程
3. StopRelayPullLogic: DeleteState + DeleteLease（删掉了 Reconcile 刚写的状态和租约）
4. Reconcile: StartPull 成功，进程在运行
5. StopRelayPullLogic: StopByTarget → 停止本地进程（如果在同一节点）

**影响**: 如果 Reconcile 在 Stop 的步骤 2 和 3 之间完成，Stop 会删掉 Reconcile 的状态，但进程可能已经在运行。后续 Reconcile 入队补拉时，状态已被删除，不会补拉。

**风险**: 中。这个竞态窗口很小，但存在。使用 `DistributedRelay.Stop`（带锁）可以解决。

## Files Found

| File Path | Description |
|---|---|
| `app/oryxserver/internal/relay/state.go` | Redis 状态/租约存储 + 分布式锁 |
| `app/oryxserver/internal/relay/distributed.go` | 分布式协调器（使用锁+租约） |
| `app/oryxserver/internal/logic/stoprelaypulllogic.go` | 停止 relay（未使用锁） |

## Summary of Issues

| # | 文件 | 行号 | 严重性 | 描述 |
|---|---|---|---|---|
| 1 | state.go | 135-149 | 低 | Renew 的 GET+EXPIRE 非原子，小概率给别人的租约续期 |
| 2 | state.go | 154-166 | 低 | Release 的 GET+DEL 非原子，小概率误删新持有者的租约 |
| 3 | stoprelaypulllogic.go | 全文 | 中 | Stop 流程未获取分布式锁，与 Reconcile 有竞态窗口 |

## Caveats / Not Found

- 非原子的 Renew/Release 操作在实际场景中影响很小（竞态窗口毫秒级，且有 Reconcile 兜底）。
- `StopRelayPullLogic` 绕过 `DistributedRelay.Stop` 是最大的架构问题——停止操作没有锁保护。
