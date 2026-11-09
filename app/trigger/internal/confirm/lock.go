package confirm

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	// LockTTL 批次确认操作的 Redis 锁 TTL。
	LockTTL       = 30 * time.Second
	lockKeyPrefix = "trigger:lock:plan:confirm:"
)

// LockKey 返回批次确认的 Redis 锁 key，确认、拒绝、通知、超时共用。
func LockKey(batchPk string) string {
	return lockKeyPrefix + batchPk
}

// AcquireLock 获取批次确认锁。Redis 未配置时直接放行，便于单测和本地运行。
func AcquireLock(ctx context.Context, rds *redis.Redis, batchPk string) (release func(), ok bool, err error) {
	if rds == nil {
		return func() {}, true, nil
	}
	lock := redis.NewRedisLock(rds, LockKey(batchPk))
	lock.SetExpire(int(LockTTL / time.Second))
	acquired, err := lock.AcquireCtx(ctx)
	if err != nil {
		return nil, false, err
	}
	if !acquired {
		return nil, false, nil
	}
	return func() { _, _ = lock.Release() }, true, nil
}
