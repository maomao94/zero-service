package gormx

import (
	"context"
	"math"
	"strconv"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"gorm.io/gorm"
)

// PageResult 分页查询结果
type PageResult struct {
	Total      int64       `json:"total"`       // 总记录数
	Page       int         `json:"page"`        // 当前页
	PageSize   int         `json:"page_size"`   // 每页大小
	TotalPages int         `json:"total_pages"` // 总页数
	Data       interface{} `json:"data"`        // 查询结果
}

// Paginate 分页 Scope
//
// 使用示例：
//
//	var users []User
//	db.Scopes(gormx.Paginate(1, 10)).Find(&users)
func Paginate(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

// QueryPage 分页查询
//
// 设计说明：
// - 先 COUNT 获取总数，再分页查询
// - 返回 PageResult 包含分页元信息
//
// 使用示例：
//
//	var users []User
//	result, err := conn.DB.Model(&User{}).Where("status = ?", 1).Session(&gorm.Session{})
//	pageResult, err := gormx.QueryPage(result, 1, 10, &users)
func QueryPage(db *gorm.DB, page, pageSize int, result interface{}) (*PageResult, error) {
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if err := db.Scopes(Paginate(page, pageSize)).Find(result).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Data:       result,
	}, nil
}

// QueryPageWithTotal 指定总数查询
//
// 当总数已知时使用，避免额外的 COUNT 查询
func QueryPageWithTotal(db *gorm.DB, page, pageSize int, total int64, result interface{}) (*PageResult, error) {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if err := db.Scopes(Paginate(page, pageSize)).Find(result).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Data:       result,
	}, nil
}

// QueryPageWithCache 分页查询（带缓存）
//
// 使用示例：
//
//	var users []User
//	pageResult, err := gormx.QueryPageWithCache(conn.DB, conn.Cache, &users,
//	    "users", 1, 10, func(db *gorm.DB) *gorm.DB {
//	        return db.Model(&User{}).Where("status = ?", 1)
//	    })
func QueryPageWithCache(db *gorm.DB, cache cache.Cache, result interface{}, cacheKey string, page, pageSize int, query func(db *gorm.DB) *gorm.DB) (*PageResult, error) {
	key := buildPageCacheKey(cacheKey, page, pageSize)

	// 尝试从缓存获取
	var pageResult PageResult
	if err := cache.Get(key, &pageResult); err == nil {
		// 缓存命中，重新查询数据
		if err := query(db.Session(&gorm.Session{})).Scopes(Paginate(page, pageSize)).Find(result).Error; err != nil {
			return nil, err
		}
		pageResult.Data = result
		return &pageResult, nil
	}

	// 缓存未命中
	var total int64
	if err := query(db.Session(&gorm.Session{})).Count(&total).Error; err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if err := query(db.Session(&gorm.Session{})).Scopes(Paginate(page, pageSize)).Find(result).Error; err != nil {
		return nil, err
	}

	pageResult = PageResult{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Data:       result,
	}

	// 写入缓存
	cache.Set(key, &pageResult)

	return &pageResult, nil
}

func buildPageCacheKey(prefix string, page, pageSize int) string {
	return prefix + ":p" + strconv.Itoa(page) + ":s" + strconv.Itoa(pageSize)
}

// ============ 多租户分页查询 ============

// QueryPageCtx 带 Context 的分页查询
//
// 使用示例：
//
//	var users []User
//	result, err := gormx.QueryPageCtx(ctx, conn.DB, 1, 10, &users, func(db *gorm.DB) *gorm.DB {
//	    return db.Where("status = ?", 1)
//	})
func QueryPageCtx(ctx context.Context, db *gorm.DB, page, pageSize int, result interface{}, whereFn func(db *gorm.DB) *gorm.DB) (*PageResult, error) {
	return QueryPage(whereFn(db.WithContext(ctx)), page, pageSize, result)
}

// QueryPageWithTenant 带租户过滤的分页查询
//
// 自动从 Context 中提取租户ID，添加租户过滤条件。
//
// 使用示例：
//
//	var users []User
//	result, err := gormx.QueryPageWithTenant(ctx, conn.DB, 1, 10, &users, func(db *gorm.DB) *gorm.DB {
//	    return db.Where("status = ?", 1)
//	})
func QueryPageWithTenant(ctx context.Context, db *gorm.DB, page, pageSize int, result interface{}, whereFn func(db *gorm.DB) *gorm.DB) (*PageResult, error) {
	db = db.WithContext(ctx).Scopes(TenantScope(ctx))
	return QueryPage(whereFn(db), page, pageSize, result)
}

// QueryPageWithCacheCtx 带 Context 和缓存的分页查询
//
// 使用示例：
//
//	var users []User
//	result, err := gormx.QueryPageWithCacheCtx(ctx, conn.DB, conn.Cache, &users,
//	    "users", 1, 10, func(db *gorm.DB) *gorm.DB {
//	        return db.Model(&User{}).Where("status = ?", 1)
//	    })
func QueryPageWithCacheCtx(ctx context.Context, db *gorm.DB, cache cache.Cache, result interface{}, cacheKey string, page, pageSize int, query func(db *gorm.DB) *gorm.DB) (*PageResult, error) {
	return QueryPageWithCache(db.WithContext(ctx), cache, result, cacheKey, page, pageSize, query)
}

// QueryPageWithTenantCache 带租户过滤的缓存分页查询
//
// 缓存 key 自动包含租户ID前缀。
//
// 使用示例：
//
//	var users []User
//	result, err := gormx.QueryPageWithTenantCache(ctx, conn.DB, conn.Cache, &users,
//	    "users", 1, 10, func(db *gorm.DB) *gorm.DB {
//	        return db.Model(&User{}).Where("status = ?", 1)
//	    })
func QueryPageWithTenantCache(ctx context.Context, db *gorm.DB, cache cache.Cache, result interface{}, cacheKey string, page, pageSize int, query func(db *gorm.DB) *gorm.DB) (*PageResult, error) {
	// 自动添加租户ID到缓存key
	tenantID := GetTenantID(ctx)
	if tenantID != "" {
		cacheKey = "tenant:" + tenantID + ":" + cacheKey
	}

	db = db.WithContext(ctx).Scopes(TenantScope(ctx))
	return QueryPageWithCache(db, cache, result, cacheKey, page, pageSize, query)
}

// CursorPageResult 游标分页结果
type CursorPageResult struct {
	Cursor   string      `json:"cursor"`    // 下一页游标
	HasMore  bool        `json:"has_more"`  // 是否有更多
	PageSize int         `json:"page_size"` // 每页大小
	Data     interface{} `json:"data"`      // 查询结果
}

// CursorPaginate 游标分页 Scope
//
// 使用场景：适合大数据量、实时性要求高的场景，避免 OFFSET 性能问题。
//
// 使用示例：
//
//	var users []User
//	result, err := gormx.CursorPaginate(ctx, conn.DB, &users,
//	    cursor, 10, "id", func(db *gorm.DB) *gorm.DB {
//	        return db.Where("status = ?", 1)
//	    })
func CursorPaginate(ctx context.Context, db *gorm.DB, cursor string, pageSize int, orderBy string, whereFn func(db *gorm.DB) *gorm.DB) (*CursorPageResult, error) {
	query := db.WithContext(ctx).Scopes(TenantScope(ctx))

	// 如果有游标，添加过滤条件
	if cursor != "" {
		query = query.Where(orderBy+" > ?", cursor)
	}

	var results []interface{}
	if err := query.Order(orderBy + " ASC").Limit(pageSize + 1).Find(&results).Error; err != nil {
		return nil, err
	}

	hasMore := len(results) > pageSize
	if hasMore {
		results = results[:pageSize]
	}

	var nextCursor string
	if hasMore && len(results) > 0 {
		// 获取最后一条记录的游标
		if v, ok := results[pageSize-1].(interface{ GetID() string }); ok {
			nextCursor = v.GetID()
		}
	}

	return &CursorPageResult{
		Cursor:   nextCursor,
		HasMore:  hasMore,
		PageSize: pageSize,
		Data:     results,
	}, nil
}

// NewPageResult 创建分页结果
//
// 便捷构造函数，支持链式调用。
//
// 使用示例：
//
//	result := gormx.NewPageResult(total, page, pageSize, users)
func NewPageResult(total int64, page, pageSize int, data interface{}) *PageResult {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	return &PageResult{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Data:       data,
	}
}

// EmptyPageResult 创建空分页结果
//
// 使用示例：
//
//	result := gormx.EmptyPageResult(page, pageSize)
func EmptyPageResult(page, pageSize int) *PageResult {
	return &PageResult{
		Total:      0,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: 0,
		Data:       []interface{}{},
	}
}
