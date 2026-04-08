package gormx

import (
	"os"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/syncx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// cacheSafeGap cache 击穿保护时间
const cacheSafeGapBetweenIndexAndPrimary = time.Second * 5

var (
	exclusiveCalls = syncx.NewSingleFlight()
	stats          = cache.NewStat("gorm_model")
)

// CachedConn GORM 连接封装，提供缓存和数据库能力
//
// 设计说明：
// - 封装 *gorm.DB 和 cache.Cache，统一提供数据库和缓存操作
// - 直接使用 GORM 链式 API，无需额外的函数类型抽象
// - 事务方法直接接收 *gorm.DB，可继续使用 GORM 链式调用
type CachedConn struct {
	Cache cache.Cache
	DB    *gorm.DB
}

// MysqlConf MySQL 配置
type MysqlConf struct {
	DataSource   string // DSN: user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
	MaxIdleConns int    // 空闲连接池最大连接数
	MaxOpenConns int    // 打开到数据库的最大连接数
}

// GetCache 获取缓存
func (cc *CachedConn) GetCache(key string, v interface{}) error {
	return cc.Cache.Get(key, v)
}

// SetCache 设置缓存
func (cc *CachedConn) SetCache(key string, v interface{}) error {
	return cc.Cache.Set(key, v)
}

// SetCacheWithExpire 设置带过期时间的缓存
func (cc *CachedConn) SetCacheWithExpire(key string, v interface{}, expire time.Duration) error {
	return cc.Cache.SetWithExpire(key, v, expire)
}

// DelCache 删除缓存
func (cc *CachedConn) DelCache(keys ...string) error {
	return cc.Cache.Del(keys...)
}

// TakeCache 缓存不存在时执行查询并缓存
//
// 使用示例：
//
//	var user User
//	err := conn.TakeCache(&user, "user:1", func(v interface{}) error {
//	    return conn.DB.Where("id = ?", 1).First(v).Error
//	})
func (cc *CachedConn) TakeCache(v interface{}, key string, queryFn func(val interface{}) error) error {
	return cc.Cache.Take(v, key, queryFn)
}

// Transact 事务
//
// 使用示例：
//
//	err := conn.Transact(func(tx *gorm.DB) error {
//	    if err := tx.Create(&user).Error; err != nil {
//	        return err
//	    }
//	    return tx.Create(&profile).Error
//	})
func (cc *CachedConn) Transact(fn func(tx *gorm.DB) error) error {
	return cc.DB.Transaction(fn)
}

// NewMySQL 创建 MySQL 连接
func NewMySQL(dsn string) (*gorm.DB, error) {
	return gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
}

// NewMySQLWithConf 使用配置创建 MySQL 连接
func NewMySQLWithConf(conf MysqlConf) (*gorm.DB, error) {
	db, err := NewMySQL(conf.DataSource)
	if err != nil {
		return nil, err
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxIdleConns(conf.MaxIdleConns)
	sqlDB.SetMaxOpenConns(conf.MaxOpenConns)
	return db, nil
}

// NewCachedConn 创建带缓存的 MySQL 连接
func NewCachedConn(conf MysqlConf, cacheConf cache.CacheConf) *CachedConn {
	db, err := NewMySQLWithConf(conf)
	if err != nil {
		logx.Must(err)
	}
	return &CachedConn{
		Cache: cache.New(cacheConf, exclusiveCalls, stats, gorm.ErrRecordNotFound),
		DB:    db,
	}
}

// AutoMigrate 自动迁移表结构
//
// 使用示例：
//
//	gormx.AutoMigrate(db, &User{}, &Order{})
func AutoMigrate(db *gorm.DB, models ...interface{}) error {
	if len(models) == 0 {
		return nil
	}
	if err := db.AutoMigrate(models...); err != nil {
		return err
	}
	logx.Infof("auto migrate %d tables success", len(models))
	return nil
}

// MustAutoMigrate 自动迁移，失败则退出
func MustAutoMigrate(db *gorm.DB, models ...interface{}) {
	if err := AutoMigrate(db, models...); err != nil {
		logx.Errorf("auto migrate failed: %v", err)
		os.Exit(1)
	}
}
