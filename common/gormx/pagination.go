package gormx

import (
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
