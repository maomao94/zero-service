package gormx

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// GetCacheCtx 获取缓存
func (cc *CachedConn) GetCacheCtx(ctx context.Context, key string, v interface{}) error {
	return cc.Cache.Get(key, v)
}

// SetCacheCtx 设置缓存
func (cc *CachedConn) SetCacheCtx(ctx context.Context, key string, v interface{}) error {
	return cc.Cache.Set(key, v)
}

// SetCacheWithExpireCtx 设置带过期时间的缓存
func (cc *CachedConn) SetCacheWithExpireCtx(ctx context.Context, key string, v interface{}, expire time.Duration) error {
	return cc.Cache.SetWithExpire(key, v, expire)
}

// DelCacheCtx 删除缓存
func (cc *CachedConn) DelCacheCtx(ctx context.Context, keys ...string) error {
	return cc.Cache.Del(keys...)
}

// TakeCacheCtx 缓存不存在时执行查询并缓存
func (cc *CachedConn) TakeCacheCtx(ctx context.Context, v interface{}, key string, queryFn func(val interface{}) error) error {
	return cc.Cache.Take(v, key, queryFn)
}

// TransactCtx 带 Context 的事务
func (cc *CachedConn) TransactCtx(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return cc.DB.WithContext(ctx).Transaction(fn)
}

// PingCtx 检查数据库连接
func (cc *CachedConn) PingCtx(ctx context.Context) error {
	sqlDB, err := cc.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// CloseCtx 关闭数据库连接
func (cc *CachedConn) CloseCtx(ctx context.Context) error {
	sqlDB, err := cc.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// QueryPageCtx 带 Context 的分页查询
func (cc *CachedConn) QueryPageCtx(ctx context.Context, db *gorm.DB, page, pageSize int, result interface{}) (*PageResult, error) {
	return QueryPage(db.WithContext(ctx), page, pageSize, result)
}

// QueryPageWithCacheCtx 带 Context 和缓存的分页查询
func (cc *CachedConn) QueryPageWithCacheCtx(ctx context.Context, result interface{}, cacheKey string, page, pageSize int, query func(db *gorm.DB) *gorm.DB) (*PageResult, error) {
	return QueryPageWithCache(cc.DB.WithContext(ctx), cc.Cache, result, cacheKey, page, pageSize, query)
}
