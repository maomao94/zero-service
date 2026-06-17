# DRC Manager 并发规范

## 核心设计

### 并发模型

```
┌─────────────────────────────────────────────────────────────┐
│                         Manager                              │
│  mu (sync.RWMutex) ─── 保护 session map                      │
│    Write: Enable（插入）、cleanLoop（删除）                     │
│    RLock: Disable、OnDeviceHeartbeat、GetNextSeq、GetStatus   │
│                                                              │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ DeviceSession                                            │ │
│  │  mu (sync.Mutex) ─── 保护状态机字段                        │ │
│  │  seq (atomic.Int64) ─── 无需 mu                          │ │
│  │  lastHeartbeat (atomic.Int64) ─── 无需 mu                │ │
│  │  heartbeatCancel (context.CancelFunc) ─── 需 mu          │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### 会话清除：Mark-and-Sweep

**单一删除点**：只有 `cleanupExpiredStates`（在 cleanLoop 中调用）执行 `delete(m.session, key)`。

其他方法（Disable、OnDeviceHeartbeat、expireSession）**仅标记** `Enabled=false` 并停止心跳 goroutine。

```
Disable / OnDeviceHeartbeat / expireSession
    │
    ▼  标记（mark）
┌────────────────────┐
│ session.Enabled=false │
│ cancelHeartbeat()    │
└────────────────────┘
    │
    │  cleanLoop 定期扫描
    ▼  清扫（sweep）
┌────────────────────┐
│ delete(m.session, key) │
└────────────────────┘
```

**好处**：
- 热路径（OnDeviceHeartbeat）只需 `m.mu.RLock()`，不与 cleanLoop 的写锁竞争
- 单一删除点降低竞态和逻辑分散风险
- 锁持有时间更短

---

## 锁顺序规则

### 原则：无交叉加锁

**禁止**在持有 `m.mu`（无论读写）的同时获取 `session.mu`（`Enable` 和 `cleanupExpiredStates` 除外）。

**正确模式（读路径）**：先释放 `m.mu.RLock()`，再获取 `session.mu.Lock()`。

```go
// 正确 — 无交叉加锁
func (m *Manager) Disable(ctx context.Context, gatewaySn string) error {
    m.mu.RLock()
    session, ok := m.session[gatewaySn]
    m.mu.RUnlock()  // 先释放
    if !ok { return nil }

    session.mu.Lock()  // 再获取
    defer session.mu.Unlock()
    if !session.Enabled { return nil }  // TOCTOU 守卫
    session.Enabled = false
    m.cancelHeartbeat(session)
    return nil
}
```

**反模式：手递手加锁（Hand-over-hand）**：

```go
// 错误 — 持有 RLock 的同时获取 session.mu
m.mu.RLock()
session := m.session[key]
session.mu.Lock()   // 如果此处阻塞，RLock 卡住所有 Writer（Enable/cleanLoop）
m.mu.RUnlock()      // 才释放 → 优先级反转
```

此模式会导致：当 session.mu 被其他 goroutine 持有时，RLock 阻塞在 session.mu 上，间接卡住所有需要 Write lock 的操作（Enable、cleanLoop），造成级联延迟甚至活锁。

### 写路径（Enable / cleanupExpiredStates）

这两个方法需要修改 map，必须持有 `m.mu.Lock()`，内部嵌套 `session.mu.Lock()` 是安全的，因为：
1. 同一时刻只有一个 Writer 能持有 `m.mu.Lock()`
2. 它们之间不会互相阻塞

```go
// Enable — 嵌套加锁安全
func (m *Manager) Enable(...) {
    m.mu.Lock()
    defer m.mu.Unlock()
    session := m.loadOrInitSession(gatewaySn)
    session.mu.Lock()
    defer session.mu.Unlock()
    // ...
}
```

### TOCTOU 处理

释放 `m.mu.RLock()` 后、获取 `session.mu.Lock()` 前存在窗口期：
- session 可能被 cleanLoop 从 map 删除 → 但我们已持有对象引用，内存仍有效
- session 可能被其他 goroutine 标记为 disabled → `if !session.Enabled` 检查兜底

**关键**：每个读路径获取 `session.mu` 后必须检查 `session.Enabled`。

---

## 锁持有矩阵

| 方法 | m.mu | session.mu | 交叉持有 | 修改 map |
|------|------|------------|---------|---------|
| `Enable` | Write (全程) | Lock (全程) | 是（安全：唯一 writer） | 是（插入） |
| `Disable` | RLock → 立即释放 | Lock | 否 | 否 |
| `OnDeviceHeartbeat` | RLock → 立即释放 | Lock | 否 | 否 |
| `expireSession` | RLock → 立即释放 | Lock | 否 | 否 |
| `GetNextSeq` | RLock → 立即释放 | Lock | 否 | 否 |
| `GetStatus` | RLock → 立即释放 | Lock | 否 | 否 |
| `isCurrentSessionAlive` | RLock → 立即释放 | Lock | 否 | 否 |
| `cleanupExpiredStates` | Write (全程) | Lock (逐个短持) | 是（安全：唯一 writer） | 是（删除） |

---

## 字段保护分类

| 字段 | 保护方式 | 读操作 | 写操作 |
|------|----------|--------|--------|
| `seq` | `atomic.Int64` | `seq.Load()` | `seq.Add(1)` |
| `lastHeartbeat` | `atomic.Int64` | `lastHeartbeat.Load()` | `lastHeartbeat.Store(...)` |
| `Enabled` | `session.mu` | 需要加锁 | 需要加锁 |
| `SessionID` | `session.mu` | 需要加锁 | 需要加锁 |
| `MaxDeadline` | `session.mu` | 需要加锁 | 需要加锁 |
| `heartbeatCancel` | `session.mu` | 需要加锁 | 需要加锁 |
| `session map` | `m.mu` | `RLock` | `Lock` |

---

## heartbeatCancel 所有权规则

| 方法 | 职责 | 调用上下文 |
|------|------|-----------|
| `startHeartbeat` | 创建 ctx/cancel，写入 `session.heartbeatCancel` | 持有 m.mu + session.mu |
| `cancelHeartbeat` | 读取 cancel、调用、置 nil | 持有 session.mu 或 session 已从 map 移除 |

`expireSession` 中的 `session.heartbeatCancel = nil` 是唯一例外：heartbeat goroutine 的 ctx 已 DeadlineExceeded，goroutine 正在退出，置 nil 仅为清理。

**`cleanupExpiredStates` 锁外调用 `cancelHeartbeat` 的安全性**：
- 此时 session 已从 map 删除且 `Enabled=false` 已在 `session.mu` 内设好
- 其他拿到旧引用的 goroutine 必然在 `!session.Enabled` 检查处退出
- 不会竞态访问 `heartbeatCancel`

---

## SessionID 版本隔离

SessionID 是防止旧 goroutine 误操作新会话的核心机制：

- Enable 每次生成新的 SessionID
- heartbeatLoop 启动时捕获当前 SessionID
- `expireSession` 通过 SessionID 比对防止误清新会话
- `isCurrentSessionAlive` 同时校验 SessionID 和存活状态

两层保护机制：
1. cancel() → ctx.Err() 是 `context.Canceled` → 不进 DeadlineExceeded 分支 → 不调 expireSession
2. 即使是 DeadlineExceeded → expireSession 校验 sessionID → 不匹配则 return

---

## cleanLoop 自适应间隔

cleanLoop 扫描间隔根据 HeartbeatTimeout 自适应计算：`clamp(HeartbeatTimeout/2, 5s, 15s)`。

保证设备断线后在合理时间内被清理，同时避免过于频繁的扫描。

---

## DeviceSession 生命周期

```
Enable()                     Disable()
    │                            │
    ▼                            ▼  标记
┌────────────────┐         ┌────────────────┐
│ Enabled=true   │         │ Enabled=false  │
│ startHeartbeat │         │ cancelHeartbeat│
│ → set cancel   │ ──────► │ → call cancel  │
│                │         │ → set nil      │
└────────────────┘         └────────────────┘
        │                          │
        │  MaxDeadline / timeout   │
        └──────────────────────────┘
                    │
                    ▼  cleanLoop 清扫
            ┌────────────────┐
            │ delete from map│
            │ 通知 Hook       │
            └────────────────┘
```

---

## 常见错误与教训

### 1. 交叉加锁导致优先级反转

**症状**：高并发下 Enable/cleanLoop 长时间等待写锁，系统延迟飙升

**原因**：
```go
// 反模式 — RLock 持有期间阻塞在 session.mu 上
m.mu.RLock()
session := m.session[key]
session.mu.Lock()   // 阻塞！卡住所有 Writer
m.mu.RUnlock()
```

**修复**：先释放 `m.mu.RUnlock()`，再获取 `session.mu.Lock()`

### 2. 散布式 delete 增加维护复杂度

**症状**：多处 `delete(m.session, key)` 各自需要写锁，逻辑分散，新增功能时容易遗漏

**原因**：每个方法自行负责删除

**修复**：Mark-and-sweep 模式，单一删除点在 cleanLoop

### 3. Cancel 丢失（heartbeatCancel 提前置 nil）

**症状**：Disable 后心跳 goroutine 不立即退出

**原因**：在调用 `cancelHeartbeat` 前手动设置 `session.heartbeatCancel = nil`

**修复**：由 `cancelHeartbeat` 独占 cancel + nil 操作

### 4. 读取受 mu 保护的字段未加锁

**症状**：`go test -race` 报 data race

**原因**：
```go
// 错误 — Enabled 受 session.mu 保护
if !session.Enabled { ... }  // data race!
```

**修复**：所有 `Enabled` 读取必须在 `session.mu.Lock()` 之后

---

## 设计决策记录

### 为什么不用 channel/event 替代锁？

设备数量有限（小规模场景），锁 + mark-and-sweep 已足够简单且可验证。Channel 方案需要：
- 管理 channel 缓冲区/关闭
- Disable 同步确认需额外 ack 机制
- 增加排查复杂度

当设备数量或心跳频率增长到锁竞争成为瓶颈时，再考虑升级为 event-driven 架构。

### 为什么 cleanLoop 不立即通知？

cleanLoop 发现 `needNotify=false`（已被 Disable/OnDeviceHeartbeat 处理过）时跳过通知，避免：
- 重复触发 expired hook
- 对已主动 Disable 的设备误报 "heartbeat_timeout"

---

## 并发测试要求

| 测试类型 | 覆盖场景 | 命令 |
|----------|----------|------|
| Race 检测 | 所有路径 | `go test -race -count=1` |
| 并发 Disable + OnDeviceHeartbeat | 同一设备 | 多 goroutine 并发调用 |
| 三路压测 | Enable/Disable/Heartbeat 同设备并发 | 验证无死锁无 race |
| Cancel 立即生效 | Disable 后 heartbeat ctx 取消 | 断言 heartbeatCancel 为 nil |
| Mark-and-sweep | Disable 后 session 仍在 map 但 !IsAlive | 断言状态而非 map 存在性 |

---

## 验证清单

- [ ] 写路径仅 `Enable`（插入）和 `cleanupExpiredStates`（删除）使用 `m.mu.Lock()`
- [ ] 读路径使用 `m.mu.RLock()` → 立即 `RUnlock()` → 再 `session.mu.Lock()`
- [ ] 无交叉加锁（不在持有 m.mu 的同时等待 session.mu，Enable/cleanLoop 除外）
- [ ] `seq` 和 `lastHeartbeat` 使用 atomic
- [ ] 仅 `cleanupExpiredStates` 执行 `delete(m.session, ...)`
- [ ] 清理操作在锁外执行 hook 和 cancelHeartbeat
- [ ] 读路径获取 session.mu 后必须检查 `!session.Enabled`
- [ ] `heartbeatCancel` 只由 `startHeartbeat` / `cancelHeartbeat` 写入
- [ ] 并发测试覆盖 Disable + Heartbeat 交叉
- [ ] `go test -race -timeout 30s` 通过
