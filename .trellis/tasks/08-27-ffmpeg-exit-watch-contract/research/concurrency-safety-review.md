# Research: Concurrency Safety Review

- **Query**: 审查 PullRegistry 的并发安全性（lifecycleMu/metaMu 使用一致性）
- **Scope**: internal
- **Date**: 2026-08-27

## Findings

### 1. PullRegistry 锁设计 (`internal/relay/registry.go`)

```go
type PullRegistry struct {
    processes *ffmpegx.Manager
    buildCmd  relayCommandBuilder

    lifecycleMu sync.Mutex    // 保护 Start/Stop 生命周期操作
    metaMu      sync.RWMutex  // 保护 meta map 的读写
    meta        map[string]*pullMeta
    onExit      func(...)     // 在 metaMu 保护下读取
    onProgress  func(...)     // 在 metaMu 保护下读取
}
```

**两把锁的职责**:
- `lifecycleMu`: 保护 Start/Stop 操作的原子性（防止同一 target 被并发 Start/Stop）
- `metaMu`: 保护 `meta` map 的并发读写

### 2. StartPull (`registry.go:64-124`)

```go
func (r *PullRegistry) StartPull(ctx context.Context, source, target, relayURL string, maxDuration ...time.Duration) error {
    target = NormalizeTarget(target)

    r.lifecycleMu.Lock()          // ① 获取生命周期锁
    defer r.lifecycleMu.Unlock()

    r.metaMu.RLock()              // ② 读锁检查是否已存在
    if _, exists := r.meta[target]; exists && r.processes.Has(target) {
        r.metaMu.RUnlock()
        return nil                 // 幂等：已存在且进程在跑 → 直接返回
    }
    r.metaMu.RUnlock()

    // ... ParseTarget, 构建 metadata ...

    r.metaMu.Lock()               // ③ 写锁注册 metadata
    r.meta[target] = md
    r.metaMu.Unlock()

    // ... 启动进程 ...
    if err := r.processes.Start(...); err != nil {
        r.metaMu.Lock()           // ④ 失败时清理 metadata
        if r.meta[target] == md {
            delete(r.meta, target)
        }
        r.metaMu.Unlock()
        return err
    }
    return nil
}
```

#### 问题 1: lifecycleMu 和 metaMu 的获取顺序不一致

- `StartPull`: lifecycleMu → metaMu (line 72 → 74/87)
- `StopByTarget`: lifecycleMu → metaMu (line 139 → 141)
- `StopPull`: lifecycleMu → metaMu (line 156 → 158)
- `StopAll`: lifecycleMu → metaMu (line 180 → 182)
- `handleExit`: 只用 metaMu (line 211)
- `handleOutput`: 只用 metaMu (line 198)
- `SetExitHandler`: 只用 metaMu (line 50)
- `SetProgressHandler`: 只用 metaMu (line 57)

**所有获取两把锁的路径都遵循 lifecycleMu → metaMu 顺序。** 不存在死锁风险。

#### 问题 2: handleExit 和 handleOutput 不持有 lifecycleMu

```go
func (r *PullRegistry) handleExit(ctx context.Context, md *pullMeta, result ffmpegx.ExitResult) {
    r.metaMu.Lock()
    if r.meta[md.Target] != md {
        r.metaMu.Unlock()
        return
    }
    delete(r.meta, md.Target)
    fn := r.onExit
    r.metaMu.Unlock()
    if fn != nil {
        fn(ctx, md.Target, result)
    }
}
```

`handleExit` 在 `metaMu` 保护下删除 metadata，但不持有 `lifecycleMu`。

**竞态场景**:
1. 进程 A 异常退出 → `handleExit` 获取 metaMu 写锁 → 删除 metadata
2. 同时 `StopByTarget` 获取 lifecycleMu → 等待 metaMu
3. `handleExit` 释放 metaMu → `StopByTarget` 获取 metaMu → 发现 metadata 已删除 → 返回 false

**影响**: `StopByTarget` 返回 false（未找到），但进程已退出。调用方可能认为进程不存在。

**风险**: 低。这是正确的语义——进程已退出，Stop 操作无需执行。

#### 问题 3: StartPull 中 metaMu.RUnlock 和 metaMu.Lock 之间的窗口

```go
r.metaMu.RLock()
if _, exists := r.meta[target]; exists && r.processes.Has(target) {
    r.metaMu.RUnlock()
    return nil
}
r.metaMu.RUnlock()  // ⑤ 释放读锁

// ... ParseTarget, 构建 metadata ...

r.metaMu.Lock()     // ⑥ 获取写锁
r.meta[target] = md
r.metaMu.Unlock()
```

在 ⑤ 和 ⑥ 之间，其他 goroutine 可以修改 `meta`。但因为 `lifecycleMu` 被持有，只有以下操作可以并发：
- `handleExit`: 可以删除 metadata（不持有 lifecycleMu）
- `handleOutput`: 只读，不影响

**竞态场景**:
1. StartPull: 读锁检查 → 不存在 → 释放读锁
2. handleExit: 写锁删除（但目标不同，不影响）
3. StartPull: 写锁注册

**无问题。** 即使 handleExit 删除了同一个 target 的 metadata（旧进程退出），StartPull 会重新注册新 metadata。

### 3. Manager 的进程生命周期与 Registry 的 metadata 生命周期同步

#### 进程生命周期（Manager）

```
Start → watchProcess → Wait → removeProcess → onExit callback
```

#### metadata 生命周期（Registry）

```
StartPull → meta[target] = md → (进程运行中) → handleExit → delete(meta, target)
```

**同步点**:
- `StartPull`: 先注册 metadata，再启动进程。如果启动失败，清理 metadata。
- `handleExit`: Manager 的 `watchProcess` 在 `Wait()` 后调用 `onExit` 回调，Registry 在回调中删除 metadata。

#### 问题 4: handleExit 中 metadata 可能已被 StopByTarget 删除

```go
func (r *PullRegistry) handleExit(ctx context.Context, md *pullMeta, result ffmpegx.ExitResult) {
    r.metaMu.Lock()
    if r.meta[md.Target] != md {  // ⑦ 检查是否是当前 metadata
        r.metaMu.Unlock()
        return
    }
    delete(r.meta, md.Target)
    // ...
}
```

⑦ 使用指针比较 `r.meta[md.Target] != md`。如果 `StopByTarget` 已删除 metadata 并启动了新进程（新 metadata），`handleExit` 会跳过旧进程的退出处理。

**测试覆盖**: `TestPullRegistryOldExitDoesNotDeleteReplacement` (registry_test.go:144-164)

**无问题。**

#### 问题 5: Manager.Stop 和 Registry.StopByTarget 的交互

```go
// Registry.StopByTarget
func (r *PullRegistry) StopByTarget(target string) bool {
    target = NormalizeTarget(target)
    r.lifecycleMu.Lock()
    defer r.lifecycleMu.Unlock()
    r.metaMu.Lock()
    if _, exists := r.meta[target]; !exists {
        r.metaMu.Unlock()
        return false
    }
    delete(r.meta, target)      // ⑧ 先删 metadata
    r.metaMu.Unlock()
    return r.processes.Stop(target)  // ⑨ 再停进程
}
```

⑧ 在 ⑨ 之前删除 metadata。如果 ⑨ 超时（Manager.Stop 有 3s StopWaitTimeout），metadata 已删除但进程可能仍在运行。

**测试覆盖**: `TestPullRegistryStopByTargetDoesNotCallExitHook` (registry_test.go:96-117)

**影响**: 进程最终会被 Manager 的 `waitProcess` 超时清理。metadata 已删除，后续的 `handleExit` 会跳过。

**无问题。**

### 4. ffmpegx.Manager 的并发安全性 (`common/ffmpegx/process.go`)

```go
type Manager struct {
    mu        sync.RWMutex
    processes map[string]*process
}
```

- `Start`: 先检查存在性（读锁），再注册（写锁）
- `Stop`: 获取写锁删除，再 cancel + wait
- `Has`: 读锁检查

#### 问题 6: Manager.Start 中检查和注册不是原子的

```go
m.mu.Lock()
_, exists := m.processes[id]
m.mu.Unlock()           // ⑩ 释放锁
if exists {
    return fmt.Errorf("process %q: %w", id, ErrProcessExists)
}
// ... build command ...
m.mu.Lock()
m.processes[id] = proc  // ⑪ 注册
m.mu.Unlock()
```

在 ⑩ 和 ⑪ 之间，另一个 goroutine 可以 Start 同一个 id。

**但**: Registry 的 `StartPull` 持有 `lifecycleMu`，保证了同一个 target 不会被并发 Start。Manager 的并发安全由调用方保证。

**无问题。**

### 5. SetExitHandler / SetProgressHandler 的线程安全性

```go
func (r *PullRegistry) SetExitHandler(fn func(ctx context.Context, target string, result ffmpegx.ExitResult)) {
    r.metaMu.Lock()
    r.onExit = fn
    r.metaMu.Unlock()
}
```

在 `metaMu` 写锁下修改 `onExit`。`handleExit` 和 `handleOutput` 在 `metaMu` 读锁下读取 `onExit`/`onProgress`。

**无问题。**

### 6. ServiceContext 中回调的并发安全性 (`internal/svc/servicecontext.go:108-113`)

```go
svcCtx.RelayPulls.SetProgressHandler(func(ctx context.Context, target string) {
    svcCtx.DistRelay.OnProgress(ctx, target)
})
svcCtx.RelayPulls.SetExitHandler(func(ctx context.Context, target string, result ffmpegx.ExitResult) {
    svcCtx.DistRelay.OnProcessExit(ctx, target, result)
})
```

回调在 ffmpeg 的 `watchProcess` goroutine 中同步调用。`OnProgress` 和 `OnProcessExit` 内部会操作 Redis（续租、释放、入队），这些操作是并发安全的（Redis 客户端是线程安全的）。

**无问题。**

## Files Found

| File Path | Description |
|---|---|
| `app/oryxserver/internal/relay/registry.go` | PullRegistry 并发控制 |
| `common/ffmpegx/process.go` | Manager 进程管理 |
| `app/oryxserver/internal/svc/servicecontext.go` | 回调注册 |

## Summary of Issues

| # | 文件 | 行号 | 严重性 | 描述 |
|---|---|---|---|---|
| 1 | registry.go | 72-88 | 无 | lifecycleMu → metaMu 顺序一致，无死锁 |
| 2 | registry.go | 210-222 | 无 | handleExit 不持有 lifecycleMu，但语义正确 |
| 3 | registry.go | 74-87 | 无 | 读锁释放到写锁获取之间的窗口，有 lifecycleMu 保护 |
| 4 | registry.go | 211-216 | 无 | 指针比较防止旧进程退出删除新 metadata |
| 5 | registry.go | 145-148 | 无 | metadata 先删、进程后停，超时时 handleExit 跳过 |
| 6 | process.go | 102-149 | 无 | 检查和注册非原子，但调用方有 lifecycleMu 保护 |

## Caveats / Not Found

- PullRegistry 的并发安全性依赖于调用方（StartPull/StopByTarget/StopPull 持有 lifecycleMu）。
- Manager 的并发安全性也依赖于调用方（Registry 保证同一 target 不会被并发 Start）。
- 整体设计是"外层锁 + 内层锁"的分层并发控制，逻辑一致。
