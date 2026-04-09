package gormx

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"gorm.io/gorm"
)

// TenantCachedConn 多租户连接封装
//
// 封装 CachedConn，提供租户隔离的数据库操作能力。
// 自动从 Context 中提取租户信息，或使用固定租户ID。
//
// 使用示例：
//
//	conn := gormx.NewTenantCachedConn(conf, cacheConf, "tenant_001")
//
//	// 自动注入租户上下文
//	ctx := gormx.WithTenantContext(ctx, "tenant_001")
//	err := conn.Query(ctx, &users, "status = ?", 1)
type TenantCachedConn struct {
	*CachedConn
	tenantID string // 固定租户ID
}

// NewTenantCachedConn 创建多租户连接
//
// 使用示例：
//
//	conn := gormx.NewTenantCachedConn(
//	    gormx.MysqlConf{
//	        DataSource:   "user:pass@tcp(localhost:3306)/db",
//	        MaxIdleConns:  10,
//	        MaxOpenConns: 100,
//	    },
//	    cache.CacheConf{
//	        {CacheKey: "cache:", Expire: time.Minute * 5},
//	    },
//	    "tenant_001",
//	)
func NewTenantCachedConn(conf MysqlConf, cacheConf cache.CacheConf, tenantID string) *TenantCachedConn {
	conn := NewCachedConn(conf, cacheConf)
	return &TenantCachedConn{
		CachedConn: conn,
		tenantID:   tenantID,
	}
}

// NewTenantCachedConnWithDB 使用已有的 CachedConn 创建多租户连接
func NewTenantCachedConnWithDB(db *gorm.DB, cache cache.Cache, tenantID string) *TenantCachedConn {
	return &TenantCachedConn{
		CachedConn: &CachedConn{
			DB:    db,
			Cache: cache,
		},
		tenantID: tenantID,
	}
}

// WithTenant 切换租户（返回新的连接实例）
//
// 使用示例：
//
//	// 创建连接后切换租户
//	tenantConn := conn.WithTenant("tenant_002")
func (cc *TenantCachedConn) WithTenant(tenantID string) *TenantCachedConn {
	return &TenantCachedConn{
		CachedConn: cc.CachedConn,
		tenantID:   tenantID,
	}
}

// TenantDB 获取带租户上下文的 DB
//
// 自动将固定租户ID注入到 Context 中。
//
// 使用示例：
//
//	err := cc.TenantDB(ctx).Create(&user).Error
func (cc *TenantCachedConn) TenantDB(ctx context.Context) *gorm.DB {
	// 如果 Context 中已有租户信息，优先使用
	existingTenantID := GetTenantID(ctx)
	if existingTenantID != "" {
		// Context 中有租户，使用 Context 中的
		return cc.DB.WithContext(ctx)
	}

	// 使用固定的租户ID
	if cc.tenantID != "" {
		ctx = WithTenantContext(ctx, cc.tenantID)
	}

	return cc.DB.WithContext(ctx)
}

// TenantDBWithUser 获取带用户和租户上下文的 DB
//
// 使用示例：
//
//	err := cc.TenantDBWithUser(ctx, 1, "admin").Create(&user).Error
func (cc *TenantCachedConn) TenantDBWithUser(ctx context.Context, userID uint, userName string) *gorm.DB {
	ctx = WithUserAndTenantContext(ctx, userID, userName, cc.tenantID)
	return cc.DB.WithContext(ctx)
}

// Query 查询（自动注入租户过滤）
//
// 使用示例：
//
//	var users []*User
//	err := conn.Query(ctx, &users)
func (cc *TenantCachedConn) Query(ctx context.Context, result interface{}, conds ...interface{}) error {
	db := cc.TenantDB(ctx).Scopes(TenantScope(ctx))
	return db.Find(result, conds...).Error
}

// QueryByID 根据ID查询（租户自动过滤）
//
// 使用示例：
//
//	var user User
//	err := conn.QueryByID(ctx, &user, 1)
func (cc *TenantCachedConn) QueryByID(ctx context.Context, result interface{}, id interface{}) error {
	db := cc.TenantDB(ctx).Scopes(TenantScope(ctx))
	return db.First(result, id).Error
}

// Create 创建（自动填充租户ID和审计字段）
//
// 使用示例：
//
//	err := conn.Create(ctx, &user)
func (cc *TenantCachedConn) Create(ctx context.Context, value interface{}) error {
	db := cc.TenantDB(ctx)
	return db.Create(value).Error
}

// Update 更新（租户自动过滤，自动填充审计字段）
//
// 使用示例：
//
//	err := conn.Update(ctx, &user)
func (cc *TenantCachedConn) Update(ctx context.Context, value interface{}) error {
	db := cc.TenantDB(ctx).Scopes(TenantScope(ctx))
	return db.Save(value).Error
}

// Delete 删除（软删除，租户自动过滤，自动填充删除人）
//
// 使用示例：
//
//	err := conn.Delete(ctx, &user)
func (cc *TenantCachedConn) Delete(ctx context.Context, value interface{}) error {
	db := cc.TenantDB(ctx).Scopes(TenantScope(ctx))
	return db.Delete(value).Error
}

// UpdateFields 更新指定字段（租户自动过滤）
//
// 使用示例：
//
//	err := conn.UpdateFields(ctx, &User{}, map[string]interface{}{"status": 1}, "id = ?", 1)
func (cc *TenantCachedConn) UpdateFields(ctx context.Context, model interface{}, updates map[string]interface{}, conds ...interface{}) error {
	db := cc.TenantDB(ctx).Scopes(TenantScope(ctx))
	return db.Model(model).Updates(updates).Error
}

// Count 统计（租户自动过滤）
//
// 使用示例：
//
//	var count int64
//	err := conn.Count(ctx, &count, &User{})
func (cc *TenantCachedConn) Count(ctx context.Context, count *int64, model interface{}) error {
	db := cc.TenantDB(ctx).Scopes(TenantScope(ctx))
	return db.Model(model).Count(count).Error
}

// Transact 事务（自动注入租户上下文）
//
// 使用示例：
//
//	err := conn.Transact(ctx, func(tx *gorm.DB) error {
//	    if err := tx.Create(&user).Error; err != nil {
//	        return err
//	    }
//	    return tx.Create(&profile).Error
//	})
func (cc *TenantCachedConn) Transact(ctx context.Context, fn func(tx *gorm.DB) error) error {
	txDB := cc.TenantDB(ctx)
	return txDB.Transaction(fn)
}

// TakeCache 缓存查询（自动注入租户）
//
// 使用示例：
//
//	var user User
//	err := conn.TakeCache(ctx, &user, "user:1", func(v interface{}) error {
//	    return cc.TenantDB(ctx).First(v, 1).Error
//	})
func (cc *TenantCachedConn) TakeCache(ctx context.Context, v interface{}, key string, queryFn func(val interface{}) error) error {
	return cc.Cache.Take(v, key, func(val interface{}) error {
		return queryFn(val)
	})
}

// QueryPage 分页查询（自动注入租户过滤）
//
// 使用示例：
//
//	var users []*User
//	result, err := conn.QueryPage(ctx, &users, 1, 10, func(db *gorm.DB) *gorm.DB {
//	    return db.Where("status = ?", 1)
//	})
func (cc *TenantCachedConn) QueryPage(ctx context.Context, result interface{}, page, pageSize int, whereFn func(db *gorm.DB) *gorm.DB) (*PageResult, error) {
	db := cc.TenantDB(ctx).Scopes(TenantScope(ctx))
	return QueryPage(whereFn(db), page, pageSize, result)
}

// GetTenantID 获取当前租户ID
func (cc *TenantCachedConn) GetTenantID() string {
	return cc.tenantID
}

// SetTenantID 设置当前租户ID
func (cc *TenantCachedConn) SetTenantID(tenantID string) {
	cc.tenantID = tenantID
}

// IsMultiTenant 检查是否启用了多租户
func (cc *TenantCachedConn) IsMultiTenant() bool {
	return cc.tenantID != ""
}

// TenantOnlyDB 获取仅租户过滤的 DB（不注入上下文，用于特殊场景）
func (cc *TenantCachedConn) TenantOnlyDB(ctx context.Context) *gorm.DB {
	db := cc.DB.WithContext(ctx)
	if cc.tenantID != "" {
		return db.Where("tenant_id = ?", cc.tenantID)
	}
	return db
}

// SuperAdminDB 获取超级管理员 DB（不过滤租户）
//
// 使用示例：
//
//	err := conn.SuperAdminDB(ctx).Find(&users).Error  // 跨所有租户查询
func (cc *TenantCachedConn) SuperAdminDB(ctx context.Context) *gorm.DB {
	return cc.DB.WithContext(WithSuperAdmin(ctx))
}
