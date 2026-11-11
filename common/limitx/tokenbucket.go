package limitx

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	// TokenBucketDefaultMinTTL 令牌桶 key 的默认最小 TTL，避免业务结束后状态长期残留。
	TokenBucketDefaultMinTTL = 300 * time.Second
)

// tokenBucketScript 通用 Redis 令牌桶。
//
// KEYS[1] 令牌桶 key；ARGV[1] 速率（个/秒）；ARGV[2] 桶容量；ARGV[3] 当前毫秒；ARGV[4] 最小 TTL 秒。
// 返回 {1, 0} 表示允许；{0, waitMs} 表示需要等待多少毫秒。
// 脚本内 TTL 统一按秒计算，写入 Redis 时换算成毫秒（PEXPIRE 的单位是毫秒）。
//
// pending 估算当前 key 已排队等待的请求数量：令牌不足时加一，并按 pending/rate 给出
// 等待时间；每次成功获取令牌时减一。这样集中到期的请求会按 1/rate 的节奏排到不同的
// 未来时间点，而不是全部在同一时刻重试造成空转。
const tokenBucketScript = `
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local minTTLSeconds = tonumber(ARGV[4])
-- ttlBufferSeconds 在按排队时长估算 TTL 时额外预留的缓冲，单位秒。
local ttlBufferSeconds = 60

local data = redis.call('HMGET', key, 'tokens', 'ts', 'pending')
local tokens = tonumber(data[1])
local ts = tonumber(data[2])
local pending = tonumber(data[3])
if tokens == nil or ts == nil then
  tokens = burst
  ts = now
end
if pending == nil then
  pending = 0
end
if now > ts then
  tokens = math.min(burst, tokens + (now - ts) * rate / 1000)
  ts = now
end

if tokens >= 1 then
  tokens = tokens - 1
  if pending > 0 then
    pending = pending - 1
  end
  redis.call('HSET', key, 'tokens', tokens, 'ts', ts, 'pending', pending)
  redis.call('PEXPIRE', key, minTTLSeconds * 1000)
  return {1, 0}
end

pending = pending + 1
local wait = math.ceil(pending * 1000 / rate)
-- pending/rate 是最后一个排队项还要等多久（秒），加缓冲后与最小 TTL 取大，
-- 保证 key 活得比 wait 更久，排队位次不会在等待期间被过期清掉。
local ttlSeconds = math.max(minTTLSeconds, math.ceil(pending / rate) + ttlBufferSeconds)
redis.call('HSET', key, 'tokens', tokens, 'ts', ts, 'pending', pending)
redis.call('PEXPIRE', key, ttlSeconds * 1000)
return {0, wait}
`

// TokenBucketOptions 表示 TokenBucket 构造配置。
type TokenBucketOptions struct {
	// Rate 令牌补充速率（个/秒），<= 0 表示限流关闭。
	Rate int
	// Burst 桶容量，<= 0 时退化为 Rate。
	Burst int
	// KeyPrefix Redis key 前缀，用于区分不同业务；最终 key 为 KeyPrefix + Allow 传入的 key。
	KeyPrefix string
	// MinTTL Redis key 的最小 TTL，<= 0 时使用 TokenBucketDefaultMinTTL。
	MinTTL time.Duration
}

// TokenBucketOption 函数式令牌桶配置选项。
type TokenBucketOption func(*TokenBucketOptions)

// WithRate 设置令牌补充速率（个/秒）。
func WithRate(rate int) TokenBucketOption {
	return func(o *TokenBucketOptions) {
		o.Rate = rate
	}
}

// WithBurst 设置桶容量。
func WithBurst(burst int) TokenBucketOption {
	return func(o *TokenBucketOptions) {
		o.Burst = burst
	}
}

// WithKeyPrefix 设置 Redis key 前缀。
func WithKeyPrefix(prefix string) TokenBucketOption {
	return func(o *TokenBucketOptions) {
		o.KeyPrefix = prefix
	}
}

// WithMinTTL 设置 Redis key 的最小 TTL。
func WithMinTTL(ttl time.Duration) TokenBucketOption {
	return func(o *TokenBucketOptions) {
		o.MinTTL = ttl
	}
}

// TokenBucket 是基于 Redis 的分布式令牌桶，多实例共享同一 key 的令牌。
// 不同 key 之间互不影响；Redis 未配置或 Rate <= 0 时限流关闭。
type TokenBucket struct {
	redis         *redis.Redis
	rate          int
	burst         int
	keyPrefix     string
	minTTLSeconds int
}

// NewTokenBucket 创建分布式令牌桶。默认关闭限流，需通过 WithRate 显式开启。
func NewTokenBucket(rds *redis.Redis, opts ...TokenBucketOption) *TokenBucket {
	o := &TokenBucketOptions{
		MinTTL: TokenBucketDefaultMinTTL,
	}
	for _, opt := range opts {
		opt(o)
	}
	if o.Rate > 0 && o.Burst <= 0 {
		o.Burst = o.Rate
	}
	if o.MinTTL <= 0 {
		o.MinTTL = TokenBucketDefaultMinTTL
	}
	minTTLSeconds := int(o.MinTTL / time.Second)
	if minTTLSeconds < 1 {
		minTTLSeconds = 1
	}
	return &TokenBucket{
		redis:         rds,
		rate:          o.Rate,
		burst:         o.Burst,
		keyPrefix:     o.KeyPrefix,
		minTTLSeconds: minTTLSeconds,
	}
}

// Enabled 返回是否启用了限流。
func (b *TokenBucket) Enabled() bool {
	return b != nil && b.redis != nil && b.rate > 0 && b.burst > 0
}

// Allow 以当前时间尝试获取一个令牌。
func (b *TokenBucket) Allow(ctx context.Context, key string) (bool, time.Duration, error) {
	return b.AllowAt(ctx, key, time.Now())
}

// AllowAt 以指定时间尝试获取一个令牌，便于测试和多步骤复用同一时钟。
// allowed=false 时 wait 表示建议等待的时间，用于把集中请求分散到后续令牌窗口。
func (b *TokenBucket) AllowAt(ctx context.Context, key string, now time.Time) (bool, time.Duration, error) {
	if !b.Enabled() {
		return true, 0, nil
	}
	res, err := b.redis.EvalCtx(ctx, tokenBucketScript,
		[]string{b.keyPrefix + key},
		b.rate, b.burst, now.UnixMilli(), b.minTTLSeconds)
	if err != nil {
		return false, 0, err
	}
	allowed, waitMs, ok := parseTokenBucketResult(res)
	if !ok {
		return false, 0, fmt.Errorf("解析令牌桶结果失败: %v", res)
	}
	if allowed == 1 {
		return true, 0, nil
	}
	wait := time.Duration(waitMs) * time.Millisecond
	if wait <= 0 {
		wait = time.Second / time.Duration(b.rate)
	}
	return false, wait, nil
}

func parseTokenBucketResult(res any) (int64, int64, bool) {
	values, ok := res.([]any)
	if !ok || len(values) < 2 {
		return 0, 0, false
	}
	allowed, ok := toInt64(values[0])
	if !ok {
		return 0, 0, false
	}
	waitMs, ok := toInt64(values[1])
	if !ok {
		return 0, 0, false
	}
	return allowed, waitMs, true
}

func toInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case uint64:
		return int64(v), true
	case float64:
		return int64(v), true
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		return n, err == nil
	case []byte:
		n, err := strconv.ParseInt(string(v), 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}
