# 技术设计：中继逻辑 UUID 标识 + app:stream key

## Architecture

### 核心变更点

```
┌─────────────────────────────────────────────────────────────┐
│                      Redis Key 结构变更                       │
├─────────────────────────────────────────────────────────────┤
│  旧: oryx:relay:state:{host_port/app/stream}                │
│  新: oryx:relay:state:{app}:{stream}                        │
│                                                             │
│  旧: oryx:relay:lease:{host_port/app/stream}                │
│  新: oryx:relay:lease:{app}:{stream}                        │
│                                                             │
│  旧: oryx:relay:lock:{host_port/app/stream}                 │
│  新: oryx:relay:lock:{app}:{stream}                         │
│                                                             │
│  Registry Sorted Set member:                                │
│  旧: {host_port/app/stream}                                 │
│  新: {app}:{stream}                                         │
└─────────────────────────────────────────────────────────────┘
```

### 数据流

```
StartRelayPull (gRPC)
    │
    ├─ 1. 解析 app, stream, source
    │
    ├─ 2. 构造 key = app + ":" + stream
    │
    ├─ 3. 检查 state:{key} 是否存在
    │     └─ 存在 → 拒绝，返回错误
    │
    ├─ 4. 生成 UUID
    │
    ├─ 5. 写入 state (含 UUID)
    │
    ├─ 6. 抢租约
    │
    └─ 7. 启动本地 ffmpeg
```

### Asynq 回调校验

```
Reconcile/Stop (Asynq)
    │
    ├─ 1. 读取 state:{app}:{stream}
    │
    ├─ 2. 比较 state.UUID == payload.UUID
    │     └─ 不匹配 → 跳过（已被新中继覆盖）
    │
    └─ 3. 执行原有逻辑
```

## Contracts

### RelayState 结构体变更

```go
type RelayState struct {
    UUID             string `json:"uuid"`                       // 中继会话 UUID
    App              string `json:"app,omitempty"`
    Stream           string `json:"stream,omitempty"`
    Host             string `json:"host,omitempty"`               // SRS 主机地址（保留，日志用）
    Port             int    `json:"port,omitempty"`               // SRS RTMP 端口（保留，日志用）
    Source           string `json:"source"`
    RelayURL         string `json:"relay_url"`
    DeadlineAtUnix   int64  `json:"deadline_at_unix,omitempty"`
    DeadlineAtStr    string `json:"deadline_at_str,omitempty"`
    CreatedAtStr     string `json:"created_at_str,omitempty"`
    RetryCount       int    `json:"retry_count,omitempty"`
    PendingReconcile bool   `json:"pending_reconcile,omitempty"`
}
```

### Payload 结构体变更

```go
type ReconcilePayload struct {
    App        string `json:"app"`
    Stream     string `json:"stream"`
    UUID       string `json:"uuid"`        // 校验用
    Source     string `json:"source,omitempty"`
    RetryCount int    `json:"retry_count,omitempty"`
}

type StopPayload struct {
    App    string `json:"app"`
    Stream string `json:"stream"`
    UUID   string `json:"uuid"`            // 校验用
}
```

### Store 方法签名变更

```go
// key 构造函数
func stateKey(app, stream string) string {
    return stateKeyPrefix + app + ":" + stream
}

func leaseKey(app, stream string) string {
    return leaseKeyPrefix + app + ":" + stream
}

func lockKey(app, stream string) string {
    return lockKeyPrefix + app + ":" + stream
}

// 方法签名调整
func (s *Store) GetState(ctx context.Context, app, stream string) (*RelayState, error)
func (s *Store) SaveState(ctx context.Context, st *RelayState) error  // 从 st.App, st.Stream 提取 key
func (s *Store) DeleteState(ctx context.Context, app, stream string) error
func (s *Store) HasLease(ctx context.Context, app, stream string) (bool, error)
func (s *Store) TryClaim(ctx context.Context, app, stream, nodeID string) (bool, error)
func (s *Store) Renew(ctx context.Context, app, stream, nodeID string) (bool, error)
func (s *Store) Release(ctx context.Context, app, stream, nodeID string) (bool, error)
func (s *Store) DeleteLease(ctx context.Context, app, stream string) error
func (s *Store) AddToRegistry(ctx context.Context, app, stream string) error
func (s *Store) RemoveFromRegistry(ctx context.Context, app, stream string) error
func (s *Store) RenewRegistry(ctx context.Context, app, stream string) error
func (s *Store) Lock(ctx context.Context, app, stream string) (*redis.RedisLock, bool, error)
```

## Compatibility

### 向后兼容
- 旧格式的 Redis key（`host_port/app/stream`）在部署后自动过期（有 TTL）
- 新旧中继任务不互通，部署时需确保无正在运行的中继任务

### 回滚方案
- 回滚代码版本即可
- 旧 key 有 TTL，会自动清理

## Trade-offs

1. **UUID 校验开销**: 每次 Asynq 回调多一次 Redis GET，可接受
2. **key 格式变更**: 旧数据需等 TTL 过期，部署窗口期需注意
3. **保留 Host/Port**: 增加字段但方便日志追溯
