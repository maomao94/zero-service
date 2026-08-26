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
	// maxReconcileRetries 补拉最大重试次数，超过后停止补拉
	maxReconcileRetries = 10
)

// RelayState 中继期望状态（Redis string，存在 ⟺ 应该运行）
type RelayState struct {
	Source         string `json:"source"`
	Target         string `json:"target"`    // 基础地址（状态/租约标识，不含鉴权参数）
	RelayURL       string `json:"relay_url"` // 完整推流地址（含鉴权参数，ffmpeg 用）
	DeadlineAtUnix int64  `json:"deadline_at_unix,omitempty"`
	DeadlineAtStr  string `json:"deadline_at_str,omitempty"` // 可视化截止时间 yyyy-MM-dd HH:mm:ss
	CreatedAtStr   string `json:"created_at_str,omitempty"`  // 创建时间 yyyy-MM-dd HH:mm:ss
	RetryCount     int    `json:"retry_count,omitempty"`     // 当前补拉重试次数
}

// ReconcilePayload 补拉任务 payload
type ReconcilePayload struct {
	Target     string `json:"target"`
	RetryCount int    `json:"retry_count,omitempty"`
}

// StopPayload 补停任务 payload
type StopPayload struct {
	Target string `json:"target"`
	App    string `json:"app"`
	Stream string `json:"stream"`
}

// leaseValue 租约值：存 nodeID
type leaseValue struct {
	NodeID string `json:"node_id"`
}

// Store 中继状态/租约存储：全部 Redis 原生命令，无 Lua。
// state key = target（明文目标地址）；lease key = target（目标地址）。
type Store struct {
	r *redis.Redis
}

// NewStore 创建状态/租约存储
func NewStore(r *redis.Redis) *Store {
	return &Store{r: r}
}

func stateKey(target string) string {
	return stateKeyPrefix + target
}

func leaseKey(target string) string {
	return leaseKeyPrefix + target
}

// GetState 读取期望状态；不存在返回 nil。
func (s *Store) GetState(ctx context.Context, target string) (*RelayState, error) {
	val, err := s.r.GetCtx(ctx, stateKey(target))
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

// SaveState 写入期望状态（覆盖写，不设 TTL）
func (s *Store) SaveState(ctx context.Context, st *RelayState) error {
	data, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("marshal relay state: %w", err)
	}
	return s.r.SetCtx(ctx, stateKey(st.Target), string(data))
}

// DeleteState 删除期望状态
func (s *Store) DeleteState(ctx context.Context, target string) error {
	_, err := s.r.DelCtx(ctx, stateKey(target))
	return err
}

// HasLease 目标地址是否被任意节点持有有效租约
func (s *Store) HasLease(ctx context.Context, target string) (bool, error) {
	val, err := s.r.GetCtx(ctx, leaseKey(target))
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val != "", nil
}

// TryClaim 抢占租约：SET NX EX（原子），成功即成为该目标的唯一运行节点
func (s *Store) TryClaim(ctx context.Context, target, nodeID string) (bool, error) {
	data, _ := json.Marshal(leaseValue{NodeID: nodeID})
	return s.r.SetnxExCtx(ctx, leaseKey(target), string(data), int(leaseTTL/time.Second))
}

// Renew 续租：GET 校验自己仍是持有者 → EXPIRE。两步非原子：
// 中间被抢占只相当于给活着的持有者续了几秒（无害）；
// 中间被删除则 EXPIRE 失败返回 false，持有方据此停止本地进程。
func (s *Store) Renew(ctx context.Context, target, nodeID string) (bool, error) {
	val, err := s.r.GetCtx(ctx, leaseKey(target))
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
	if err := s.r.ExpireCtx(ctx, leaseKey(target), int(leaseTTL/time.Second)); err != nil {
		return false, err
	}
	return true, nil
}

// Release 释放租约：GET 校验持有者是自己 → DEL。仅释放自己的租约，避免误删新持有者。
func (s *Store) Release(ctx context.Context, target, nodeID string) (bool, error) {
	val, err := s.r.GetCtx(ctx, leaseKey(target))
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
	_, err = s.r.DelCtx(ctx, leaseKey(target))
	return err == nil, err
}

// DeleteLease 无条件删除租约（StopRelayPull 语义：无论谁持有都清掉）
func (s *Store) DeleteLease(ctx context.Context, target string) error {
	_, err := s.r.DelCtx(ctx, leaseKey(target))
	return err
}

// Lock 获取 per-target 分布式锁，保护 Start/Stop/Reconcile 的关键段互斥。
func (s *Store) Lock(ctx context.Context, target string) (*redis.RedisLock, bool, error) {
	lock := redis.NewRedisLock(s.r, lockKeyPrefix+target)
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
