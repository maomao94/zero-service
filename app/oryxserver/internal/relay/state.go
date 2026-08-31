package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	// RelayQueue 隔离队列名（与 trigger 的 critical/default/low 完全隔离）
	RelayQueue = "oryx-relay"
	// RelayTaskPrefix 所有 relay 任务类型前缀
	RelayTaskPrefix = "oryx:relay:"
	// RelayReconcileTask 中继补偿/补拉任务类型
	RelayReconcileTask = RelayTaskPrefix + "reconcile"
	// RelayStopTask 中继补停任务类型（广播超时后重试广播停止）
	RelayStopTask = RelayTaskPrefix + "stop"
	// stateKeyPrefix 状态 key 前缀（存在 ⟺ 应该运行）
	stateKeyPrefix = RelayTaskPrefix + "state:"
	// leaseKeyPrefix 租约 key 前缀（值 = nodeID，TTL 到期即失效）
	leaseKeyPrefix = RelayTaskPrefix + "lease:"
	// leaseTTL 租约时长：ffmpeg progress 每帧续一次，进程死则断租
	leaseTTL = 30 * time.Second
	// lockKeyPrefix 分布式锁前缀：保护 Start/Stop/Reconcile 的 read-check-act 关键段
	lockKeyPrefix = RelayTaskPrefix + "lock:"
	// lockTTL 分布式锁最大持有时间（秒），覆盖 state 写入 + lease 抢占 + 本地启动
	lockTTL = 15
	// defaultRelayDuration max_duration_seconds=0 时的默认中继时长（1天）
	defaultRelayDuration = 24 * time.Hour
	// stateTTLOverhead deadline 之外额外保留时间（1天兜底，防止进程崩溃后 key 永久残留）
	stateTTLOverhead = 24 * time.Hour
	// registryKey relay 索引 Sorted Set：score = 最后更新时间（Unix 秒），member = app:stream
	registryKey = RelayTaskPrefix + "registry"
	// staleThreshold 扫描阈值：超过此时间未更新的条目视为孤儿候选
	staleThreshold = 60 * time.Second
	// PendingStaleThreshold pending=true 超过此时间仍未被消费（Asynq 任务可能丢失），最终补偿重入队
	PendingStaleThreshold = 5 * time.Minute
)

// StaleEntry 扫描结果：state + Sorted Set score（score = 最后更新时间 Unix 秒）
type StaleEntry struct {
	State *RelayState
	Score int64 // Sorted Set score（Unix 秒，即最后 RenewRegistry 时间）
}

// RelayState 中继期望状态（Redis string，存在 ⟺ 应该运行）
type RelayState struct {
	UUID             string `json:"uuid"`                        // 中继会话 UUID（每次开启中继生成）
	App              string `json:"app,omitempty"`                // 目标应用名
	Stream           string `json:"stream,omitempty"`             // 目标流名
	Host             string `json:"host,omitempty"`               // SRS 主机地址（日志用）
	Port             int    `json:"port,omitempty"`               // SRS RTMP 端口（日志用）
	Source           string `json:"source"`                       // 源流地址
	RelayURL         string `json:"relay_url"`                    // 完整推流地址（含鉴权参数，ffmpeg 用）
	DeadlineAtUnix   int64  `json:"deadline_at_unix,omitempty"`
	DeadlineAtStr    string `json:"deadline_at_str,omitempty"`    // yyyy-MM-dd HH:mm:ss
	CreatedAtStr     string `json:"created_at_str,omitempty"`     // yyyy-MM-dd HH:mm:ss
	RetryCount       int    `json:"retry_count,omitempty"`        // 当前补拉重试次数
	PendingReconcile bool   `json:"pending_reconcile,omitempty"`  // 防扫描器重复入队
}

// ReconcilePayload 补拉任务 payload
type ReconcilePayload struct {
	App        string `json:"app"`
	Stream     string `json:"stream"`
	UUID       string `json:"uuid"` // 校验用，不匹配则跳过
	RetryCount int    `json:"retry_count,omitempty"`
}

// StopPayload 补停任务 payload
type StopPayload struct {
	App    string `json:"app"`
	Stream string `json:"stream"`
	UUID   string `json:"uuid"`            // 校验用，不匹配则跳过
}

// leaseValue 租约值：存 nodeID
type leaseValue struct {
	NodeID string `json:"node_id"`
}

// Store 中继状态/租约存储：全部 Redis 原生命令，无 Lua。
// state key = app:stream；lease key = app:stream。
type Store struct {
	r *redis.Redis
}

// NewStore 创建状态/租约存储
func NewStore(r *redis.Redis) *Store {
	return &Store{r: r}
}

// CanonicalUID 将任意 target URL 转为唯一标识：host_port/app/stream。
// Deprecated: 新逻辑使用 app:stream 作为 key，此函数保留仅用于向后兼容和测试。
func CanonicalUID(target string) (string, error) {
	normalized := NormalizeTarget(target)
	i := strings.Index(normalized, "/")
	if i < 0 {
		return "", fmt.Errorf("target must have /{app}/{stream}, got: %s", target)
	}
	host := strings.ReplaceAll(normalized[:i], ":", "_")
	path := normalized[i+1:]
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("target must have exactly /{app}/{stream}, got: %s", target)
	}
	return host + "/" + path, nil
}

func stateKey(app, stream string) string {
	return stateKeyPrefix + app + ":" + stream
}

func leaseKey(app, stream string) string {
	return leaseKeyPrefix + app + ":" + stream
}

func lockKey(app, stream string) string {
	return lockKeyPrefix + app + ":" + stream
}

// registryMember 构造 Sorted Set member: app:stream
func registryMember(app, stream string) string {
	return app + ":" + stream
}

// GetState 读取期望状态；不存在返回 nil。
func (s *Store) GetState(ctx context.Context, app, stream string) (*RelayState, error) {
	val, err := s.r.GetCtx(ctx, stateKey(app, stream))
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if val == "" {
		return nil, nil
	}
	var st RelayState
	if err := json.Unmarshal([]byte(val), &st); err != nil {
		return nil, fmt.Errorf("unmarshal relay state: %w", err)
	}
	return &st, nil
}

// SaveState 写入期望状态（覆盖写，TTL = 剩余时长 + 1天兜底，防止进程崩溃后 key 永久残留）
func (s *Store) SaveState(ctx context.Context, st *RelayState) error {
	data, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("marshal relay state: %w", err)
	}
	key := stateKey(st.App, st.Stream)
	if err := s.r.SetCtx(ctx, key, string(data)); err != nil {
		return err
	}
	// TTL: deadline 剩余时长 + 1天兜底；deadline=0 则用默认时长 + 1天
	ttl := stateTTLOverhead
	if st.DeadlineAtUnix > 0 {
		remaining := time.Until(time.Unix(st.DeadlineAtUnix, 0))
		if remaining > 0 {
			ttl = remaining + stateTTLOverhead
		}
	} else {
		ttl = defaultRelayDuration + stateTTLOverhead
	}
	return s.r.ExpireCtx(ctx, key, int(ttl/time.Second))
}

// DeleteState 删除期望状态
func (s *Store) DeleteState(ctx context.Context, app, stream string) error {
	_, err := s.r.DelCtx(ctx, stateKey(app, stream))
	return err
}

// HasLease 目标地址是否被任意节点持有有效租约
func (s *Store) HasLease(ctx context.Context, app, stream string) (bool, error) {
	val, err := s.r.GetCtx(ctx, leaseKey(app, stream))
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val != "", nil
}

// TryClaim 抢占租约：SET NX EX（原子），成功即成为该目标的唯一运行节点
func (s *Store) TryClaim(ctx context.Context, app, stream, nodeID string) (bool, error) {
	data, _ := json.Marshal(leaseValue{NodeID: nodeID})
	return s.r.SetnxExCtx(ctx, leaseKey(app, stream), string(data), int(leaseTTL/time.Second))
}

// Renew 续租：GET 校验自己仍是持有者 → EXPIRE。两步非原子：
// 中间被抢占只相当于给活着的持有者续了几秒（无害）；
// 中间被删除则 EXPIRE 失败返回 false，持有方据此停止本地进程。
func (s *Store) Renew(ctx context.Context, app, stream, nodeID string) (bool, error) {
	val, err := s.r.GetCtx(ctx, leaseKey(app, stream))
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var lv leaseValue
	if err := json.Unmarshal([]byte(val), &lv); err != nil || lv.NodeID != nodeID {
		return false, nil
	}
	if err := s.r.ExpireCtx(ctx, leaseKey(app, stream), int(leaseTTL/time.Second)); err != nil {
		return false, err
	}
	return true, nil
}

// Release 释放租约：GET 校验持有者是自己 → DEL。仅释放自己的租约，避免误删新持有者。
func (s *Store) Release(ctx context.Context, app, stream, nodeID string) (bool, error) {
	val, err := s.r.GetCtx(ctx, leaseKey(app, stream))
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var lv leaseValue
	if err := json.Unmarshal([]byte(val), &lv); err != nil || lv.NodeID != nodeID {
		return false, nil
	}
	_, err = s.r.DelCtx(ctx, leaseKey(app, stream))
	return err == nil, err
}

// DeleteLease 无条件删除租约（StopRelayPull 语义：无论谁持有都清掉）
func (s *Store) DeleteLease(ctx context.Context, app, stream string) error {
	_, err := s.r.DelCtx(ctx, leaseKey(app, stream))
	return err
}

// AddToRegistry 将 app:stream 加入 relay 索引 Sorted Set（score = 当前 Unix 秒）。
// 启动 relay 时调用，幂等（ZADD 同 member 会刷新 score）。
func (s *Store) AddToRegistry(ctx context.Context, app, stream string) error {
	_, err := s.r.ZaddCtx(ctx, registryKey, time.Now().Unix(), registryMember(app, stream))
	return err
}

// RemoveFromRegistry 将 app:stream 从 relay 索引 Sorted Set 移除。
// 主动停止 relay 时调用。
func (s *Store) RemoveFromRegistry(ctx context.Context, app, stream string) error {
	_, err := s.r.ZremCtx(ctx, registryKey, registryMember(app, stream))
	return err
}

// RenewRegistry 刷新 app:stream 在 Sorted Set 中的 score（续期语义）。
// progress 回调时调用，与续租同步。
func (s *Store) RenewRegistry(ctx context.Context, app, stream string) error {
	_, err := s.r.ZaddCtx(ctx, registryKey, time.Now().Unix(), registryMember(app, stream))
	return err
}

// ScanStale 扫描 Sorted Set 中超过 staleThreshold 未更新的条目，返回对应的 RelayState 列表。
// O(log N + M)，N = 总条目数，M = 过期条目数（通常远小于 N）。
// state 已被 cleanupTarget 删除的过期条目会被自动清理。
func (s *Store) ScanStale(ctx context.Context) ([]*StaleEntry, error) {
	threshold := time.Now().Add(-staleThreshold).Unix()
	members, err := s.r.ZrangebyscoreWithScoresCtx(ctx, registryKey, 0, threshold)
	if err != nil {
		return nil, err
	}
	var result []*StaleEntry
	for _, m := range members {
		member := m.Key // 格式: app:stream
		app, stream, err := parseRegistryMember(member)
		if err != nil {
			// 无效格式，清理
			_, _ = s.r.ZremCtx(ctx, registryKey, member)
			continue
		}
		val, err := s.r.GetCtx(ctx, stateKey(app, stream))
		if err != nil {
			if err == redis.Nil {
				_, _ = s.r.ZremCtx(ctx, registryKey, member)
				continue
			}
			return nil, err
		}
		if val == "" {
			_, _ = s.r.ZremCtx(ctx, registryKey, member)
			continue
		}
		var st RelayState
		if err := json.Unmarshal([]byte(val), &st); err != nil {
			continue
		}
		result = append(result, &StaleEntry{State: &st, Score: int64(m.Score)})
	}
	return result, nil
}

// parseRegistryMember 从 app:stream 格式解析 app 和 stream
func parseRegistryMember(member string) (app, stream string, err error) {
	parts := strings.SplitN(member, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid registry member format: %s", member)
	}
	return parts[0], parts[1], nil
}

// Lock 获取 per-app:stream 分布式锁，保护 Start/Stop/Reconcile 的 read-check-act 关键段互斥。
func (s *Store) Lock(ctx context.Context, app, stream string) (*redis.RedisLock, bool, error) {
	lock := redis.NewRedisLock(s.r, lockKey(app, stream))
	lock.SetExpire(lockTTL)
	ok, err := lock.AcquireCtx(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("acquire lock: %w", err)
	}
	return lock, ok, nil
}

// NormalizeTarget 将 target URL 标准化为 scheme 无关的唯一标识。
// "rtmp://host:1935/live/stream?secret=abc" → "host:1935/live/stream"
// 不同协议（rtmp/http）指向同一 host:port/app/stream 时产生相同 key。
func NormalizeTarget(target string) string {
	// 已经是 normalized key（无 scheme）：直接清理
	if !strings.Contains(target, "://") {
		key := strings.TrimRight(target, "/")
		if i := strings.IndexByte(key, '?'); i >= 0 {
			key = key[:i]
		}
		return key
	}
	// 有 scheme：用 url.Parse 提取 host + path
	u, err := url.Parse(target)
	if err != nil {
		return target
	}
	key := u.Host + u.Path
	if i := strings.IndexByte(key, '?'); i >= 0 {
		key = key[:i]
	}
	return strings.TrimRight(key, "/")
}

// ParseTarget 从 normalized key 解析 app 和 stream。
// normalized key 格式：host[:port]/{app}/{stream}（scheme 已移除）
// 校验：path 必须恰好两段（/{app}/{stream}），多段或 app/stream 含 / 均报错。
func ParseTarget(target string) (app, stream string, err error) {
	// 去掉查询参数（防御性）
	if i := strings.IndexByte(target, '?'); i >= 0 {
		target = target[:i]
	}
	target = strings.TrimRight(target, "/")

	firstSlash := strings.Index(target, "/")
	if firstSlash < 0 {
		return "", "", fmt.Errorf("target must have /{app}/{stream}, got: %s", target)
	}
	host := target[:firstSlash]
	path := target[firstSlash+1:]
	if host == "" || path == "" {
		return "", "", fmt.Errorf("target must have /{app}/{stream}, got: %s", target)
	}

	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("target must have exactly /{app}/{stream}, got: %s", target)
	}
	return parts[0], parts[1], nil
}
