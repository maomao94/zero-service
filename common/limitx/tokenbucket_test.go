package limitx

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

func newTokenBucketTestRedis(t *testing.T) *redis.Redis {
	t.Helper()
	server := miniredis.RunT(t)
	return redis.New(server.Addr())
}

func TestTokenBucketDisabled(t *testing.T) {
	bucket := NewTokenBucket(nil, WithRate(1), WithBurst(1))
	if bucket.Enabled() {
		t.Fatal("bucket without redis should be disabled")
	}
	allowed, wait, err := bucket.AllowAt(context.Background(), "key", time.Now())
	if err != nil || !allowed || wait != 0 {
		t.Fatalf("AllowAt() = %v, %v, %v; want allowed without wait", allowed, wait, err)
	}

	bucket = NewTokenBucket(newTokenBucketTestRedis(t), WithRate(0), WithBurst(0))
	if bucket.Enabled() {
		t.Fatal("bucket with rate 0 should be disabled")
	}
}

func TestTokenBucketBurstAndSpread(t *testing.T) {
	rds := newTokenBucketTestRedis(t)
	bucket := NewTokenBucket(rds, WithRate(2), WithBurst(2)) // 2 个/秒，桶容量 2
	ctx := context.Background()
	now := time.Now()

	for i := 0; i < 2; i++ {
		allowed, _, err := bucket.AllowAt(ctx, "key", now)
		if err != nil || !allowed {
			t.Fatalf("burst AllowAt() #%d = %v, %v; want allowed", i+1, allowed, err)
		}
	}

	allowed, wait, err := bucket.AllowAt(ctx, "key", now)
	if err != nil || allowed {
		t.Fatalf("third AllowAt() = %v, %v; want denied", allowed, err)
	}
	if wait != 500*time.Millisecond {
		t.Fatalf("third wait = %v, want 500ms", wait)
	}

	// 第二个排队项应排到下一个令牌窗口，而不是和上一个同一时刻重试。
	allowed, wait, err = bucket.AllowAt(ctx, "key", now)
	if err != nil || allowed {
		t.Fatalf("fourth AllowAt() = %v, %v; want denied", allowed, err)
	}
	if wait != time.Second {
		t.Fatalf("fourth wait = %v, want 1s", wait)
	}

	// 500ms 后补充一个令牌，第一个排队项可以下发。
	allowed, wait, err = bucket.AllowAt(ctx, "key", now.Add(500*time.Millisecond))
	if err != nil || !allowed || wait != 0 {
		t.Fatalf("AllowAt() after refill = %v, %v, %v; want allowed", allowed, wait, err)
	}
}

func TestTokenBucketIsolatesKeys(t *testing.T) {
	rds := newTokenBucketTestRedis(t)
	bucket := NewTokenBucket(rds, WithRate(1), WithBurst(1))
	ctx := context.Background()
	now := time.Now()

	if allowed, _, err := bucket.AllowAt(ctx, "a", now); err != nil || !allowed {
		t.Fatalf("key a first AllowAt() = %v, %v; want allowed", allowed, err)
	}
	if allowed, _, err := bucket.AllowAt(ctx, "a", now); err != nil || allowed {
		t.Fatalf("key a second AllowAt() = %v, %v; want denied", allowed, err)
	}
	if allowed, _, err := bucket.AllowAt(ctx, "b", now); err != nil || !allowed {
		t.Fatalf("key b AllowAt() = %v, %v; want allowed", allowed, err)
	}
}

// requireTokenBucketTTL 断言 Redis key 的剩余 TTL 与期望一致，容忍 1 秒的执行耗时。
func requireTokenBucketTTL(t *testing.T, server *miniredis.Miniredis, key string, want time.Duration) {
	t.Helper()
	got := server.TTL(key)
	if got > want || got <= want-time.Second {
		t.Fatalf("ttl = %v, want ~%v", got, want)
	}
}

// PEXPIRE 的单位是毫秒，这里锁死「秒 → 毫秒」的换算，避免 TTL 被写成毫秒数导致桶状态被提前清空。
func TestTokenBucketKeyTTL(t *testing.T) {
	server := miniredis.RunT(t)
	rds := redis.New(server.Addr())
	ctx := context.Background()
	now := time.Now()

	t.Run("放行路径使用最小 TTL", func(t *testing.T) {
		bucket := NewTokenBucket(rds, WithRate(10), WithBurst(10), WithKeyPrefix("ttl-allow:"), WithMinTTL(2*time.Minute))
		if allowed, _, err := bucket.AllowAt(ctx, "key", now); err != nil || !allowed {
			t.Fatalf("AllowAt() = %v, %v; want allowed", allowed, err)
		}
		requireTokenBucketTTL(t, server, "ttl-allow:key", 2*time.Minute)
	})

	t.Run("拒绝路径按排队时长延长", func(t *testing.T) {
		bucket := NewTokenBucket(rds, WithRate(1), WithBurst(1), WithKeyPrefix("ttl-deny:"), WithMinTTL(10*time.Second))
		if allowed, _, err := bucket.AllowAt(ctx, "key", now); err != nil || !allowed {
			t.Fatalf("AllowAt() = %v, %v; want allowed", allowed, err)
		}
		for i := 0; i < 100; i++ {
			if allowed, _, err := bucket.AllowAt(ctx, "key", now); err != nil || allowed {
				t.Fatalf("AllowAt() #%d = %v, %v; want denied", i+1, allowed, err)
			}
		}
		// pending=100、rate=1：TTL 应延长到 100 秒排队时间 + 60 秒缓冲。
		requireTokenBucketTTL(t, server, "ttl-deny:key", 160*time.Second)
	})
}

// 排队位次（pending）必须活过 wait 对应的等待时间，否则被限速的执行项回来时会重新从第 1 位排队，
// wait 永远停在 1/rate 附近，集中到期的请求会以极短间隔反复重试。
func TestTokenBucketPendingSurvivesWaitWindow(t *testing.T) {
	server := miniredis.RunT(t)
	rds := redis.New(server.Addr())
	bucket := NewTokenBucket(rds, WithRate(1), WithBurst(1), WithKeyPrefix("pending:"), WithMinTTL(2*time.Minute))
	ctx := context.Background()
	now := time.Now()

	if allowed, _, err := bucket.AllowAt(ctx, "key", now); err != nil || !allowed {
		t.Fatalf("first AllowAt() = %v, %v; want allowed", allowed, err)
	}
	for i := 1; i <= 3; i++ {
		allowed, wait, err := bucket.AllowAt(ctx, "key", now)
		if err != nil || allowed {
			t.Fatalf("AllowAt() #%d = %v, %v; want denied", i, allowed, err)
		}
		if want := time.Duration(i) * time.Second; wait != want {
			t.Fatalf("wait #%d = %v, want %v", i, wait, want)
		}
	}

	// 空闲 100 秒仍小于 TTL：排队位次不能被重置，下一个排到第 4 位。
	server.FastForward(100 * time.Second)
	allowed, wait, err := bucket.AllowAt(ctx, "key", now)
	if err != nil || allowed {
		t.Fatalf("AllowAt() after idle = %v, %v; want denied", allowed, err)
	}
	if want := 4 * time.Second; wait != want {
		t.Fatalf("wait after idle = %v, want %v", wait, want)
	}

	// 超过 TTL 后 key 过期，桶回到满桶、排队位次清零。
	server.FastForward(3 * time.Minute)
	allowed, wait, err = bucket.AllowAt(ctx, "key", now.Add(280*time.Second))
	if err != nil || !allowed || wait != 0 {
		t.Fatalf("AllowAt() after ttl = %v, %v, %v; want allowed without wait", allowed, wait, err)
	}
}

func TestTokenBucketKeyPrefix(t *testing.T) {
	rds := newTokenBucketTestRedis(t)
	ctx := context.Background()
	now := time.Now()
	first := NewTokenBucket(rds, WithRate(1), WithBurst(1), WithKeyPrefix("first:"))
	second := NewTokenBucket(rds, WithRate(1), WithBurst(1), WithKeyPrefix("second:"))

	if allowed, _, err := first.AllowAt(ctx, "key", now); err != nil || !allowed {
		t.Fatalf("first bucket AllowAt() = %v, %v; want allowed", allowed, err)
	}
	if allowed, _, err := first.AllowAt(ctx, "key", now); err != nil || allowed {
		t.Fatalf("first bucket second AllowAt() = %v, %v; want denied", allowed, err)
	}
	// 不同前缀使用不同的 Redis key，不应共享令牌。
	if allowed, _, err := second.AllowAt(ctx, "key", now); err != nil || !allowed {
		t.Fatalf("second bucket AllowAt() = %v, %v; want allowed", allowed, err)
	}
}

func TestTokenBucketRefillCapsAtBurst(t *testing.T) {
	rds := newTokenBucketTestRedis(t)
	bucket := NewTokenBucket(rds, WithRate(2), WithBurst(3))
	ctx := context.Background()
	now := time.Now()

	allowedCount := 0
	for i := 0; i < 3; i++ {
		allowed, _, err := bucket.AllowAt(ctx, "key", now)
		if err != nil {
			t.Fatal(err)
		}
		if allowed {
			allowedCount++
		}
	}
	if allowedCount != 3 {
		t.Fatalf("allowed count = %d, want 3", allowedCount)
	}

	// 空闲很久后令牌只补到桶容量，不能无限累计。
	idle := now.Add(time.Hour)
	allowedCount = 0
	for i := 0; i < 5; i++ {
		allowed, _, err := bucket.AllowAt(ctx, "key", idle)
		if err != nil {
			t.Fatal(err)
		}
		if allowed {
			allowedCount++
		}
	}
	if allowedCount != 3 {
		t.Fatalf("allowed count after idle = %d, want burst 3", allowedCount)
	}
}
