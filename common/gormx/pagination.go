package gormx

import (
	"context"
	"math"
	"strconv"
	"unsafe"

	"gorm.io/gorm"
)

// PageResult[T] 分页查询结果（泛型）
type PageResult[T any] struct {
	Total      int64 `json:"total"`       // 总记录数
	Page       int   `json:"page"`        // 当前页
	PageSize   int   `json:"page_size"`   // 每页大小
	TotalPages int   `json:"total_pages"` // 总页数
	Data       T     `json:"data"`        // 查询结果
}

// QueryPage 分页查询
//
// 使用示例：
//
//	var users []User
//	result, err := gormx.QueryPage(conn.DB, 1, 10, &users)
func QueryPage[T any](db *gorm.DB, page, pageSize int, result *T) (*PageResult[T], error) {
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if err := db.Offset(offset).Limit(pageSize).Find(result).Error; err != nil {
		return nil, err
	}

	return &PageResult[T]{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Data:       *result,
	}, nil
}

// QueryPageWithTotal 指定总数查询
func QueryPageWithTotal[T any](db *gorm.DB, page, pageSize int, total int64, result *T) (*PageResult[T], error) {
	offset := (page - 1) * pageSize
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if err := db.Offset(offset).Limit(pageSize).Find(result).Error; err != nil {
		return nil, err
	}

	return &PageResult[T]{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Data:       *result,
	}, nil
}

func buildPageCacheKey(prefix string, page, pageSize int) string {
	return prefix + ":p" + strconv.Itoa(page) + ":s" + strconv.Itoa(pageSize)
}

// QueryPageCtx 带 Context 的分页查询
func QueryPageCtx[T any](ctx context.Context, db *gorm.DB, page, pageSize int, result *T, whereFn func(db *gorm.DB) *gorm.DB) (*PageResult[T], error) {
	return QueryPage(whereFn(db.WithContext(ctx)), page, pageSize, result)
}

// QueryPageWithTenant 带租户过滤的分页查询
func QueryPageWithTenant[T any](ctx context.Context, db *gorm.DB, page, pageSize int, result *T, whereFn func(db *gorm.DB) *gorm.DB) (*PageResult[T], error) {
	db = db.WithContext(ctx).Scopes(TenantScope(ctx))
	return QueryPage(whereFn(db), page, pageSize, result)
}

// NewPageResult 创建分页结果
func NewPageResult[T any](total int64, page, pageSize int, data T) *PageResult[T] {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	return &PageResult[T]{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Data:       data,
	}
}

// EmptyPageResult 创建空分页结果
func EmptyPageResult[T any](page, pageSize int) *PageResult[T] {
	return &PageResult[T]{
		Total:      0,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: 0,
		Data:       *new(T),
	}
}

// ============ 游标分页 ============

// CursorPageResult 游标分页结果
type CursorPageResult[T any] struct {
	Cursor  string `json:"cursor"`   // 下一页游标
	HasMore bool   `json:"has_more"` // 是否有更多
	Data    T      `json:"data"`     // 查询结果
}

// CursorPage 游标分页查询（通用版）
//
// 使用示例（基于 ID 游标）：
//
//	var users []User
//	result, err := gormx.CursorPage(db, 10, "", &users,
//	    func(db *gorm.DB, cursor string) *gorm.DB {
//	        if cursor != "" {
//	            id, _ := strconv.ParseInt(cursor, 10, 64)
//	            return db.Where("id > ?", id)
//	        }
//	        return db
//	    },
//	    func(row any) string {
//	        // 从每行提取游标（最后一条的 ID）
//	        if u, ok := row.(User); ok {
//	            return strconv.FormatInt(int64(u.Id), 10)
//	        }
//	        return ""
//	    },
//	)
func CursorPage[T any](
	db *gorm.DB,
	pageSize int,
	cursor string,
	result *T,
	whereFn func(db *gorm.DB, cursor string) *gorm.DB,
	cursorFn func(row any) string,
) (*CursorPageResult[T], error) {
	if pageSize <= 0 {
		pageSize = 10
	}

	// 查询 pageSize + 1 条，用于判断是否有更多
	query := whereFn(db, cursor).Limit(pageSize + 1)
	if err := query.Find(result).Error; err != nil {
		return nil, err
	}

	// 获取实际数据
	data := *result
	dataSlice, isSlice := any(data).(sliceHeader)
	if !isSlice || dataSlice.Len < pageSize {
		// 数据不足 pageSize 条，没有更多
		return &CursorPageResult[T]{
			Cursor:  "",
			HasMore: false,
			Data:    data,
		}, nil
	}

	// 有更多数据
	hasMore := true
	nextCursor := ""
	if lastItem := dataSlice.get(dataSlice.Len - 1); lastItem != nil {
		nextCursor = cursorFn(lastItem)
	}

	return &CursorPageResult[T]{
		Cursor:  nextCursor,
		HasMore: hasMore,
		Data:    data,
	}, nil
}

// sliceHeader 反射获取切片长度和元素
type sliceHeader struct {
	Addr unsafe.Pointer
	Len  int
	Cap  int
}

func (s sliceHeader) get(i int) any {
	if i < 0 || i >= s.Len {
		return nil
	}
	typ := (*unsafe.Pointer)(unsafe.Add(s.Addr, uintptr(i)*ptrSize))
	return *(*any)(unsafe.Pointer(typ))
}

var ptrSize = unsafe.Sizeof((*any)(nil))

// CursorPageCtx 带 Context 的游标分页
func CursorPageCtx[T any](
	ctx context.Context,
	db *gorm.DB,
	pageSize int,
	cursor string,
	result *T,
	whereFn func(db *gorm.DB, cursor string) *gorm.DB,
	cursorFn func(row any) string,
) (*CursorPageResult[T], error) {
	return CursorPage(db.WithContext(ctx), pageSize, cursor, result, whereFn, cursorFn)
}

// CursorPageByID 基于 ID 的简化游标分页
//
// 使用示例：
//
//	var users []User
//	result, err := gormx.CursorPageByID(db, 10, "", &users, "id")
func CursorPageByID(db *gorm.DB, pageSize int, cursor string, result any, idField string) (*CursorPageResult[any], error) {
	var total int64
	query := db.Model(result)
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if pageSize <= 0 {
		pageSize = 10
	}

	var q *gorm.DB
	if cursor != "" {
		id, err := strconv.ParseInt(cursor, 10, 64)
		if err != nil {
			return nil, err
		}
		q = db.Where(idField+" > ?", id)
	} else {
		q = db
	}

	// 查询 pageSize + 1 条，用于判断是否有更多
	var data []map[string]any
	if err := q.Limit(pageSize + 1).Find(&data).Error; err != nil {
		return nil, err
	}

	hasMore := len(data) > pageSize
	nextCursor := ""

	if hasMore && len(data) > 0 {
		if id, ok := data[len(data)-1][idField]; ok {
			if idInt, ok := id.(int64); ok {
				nextCursor = strconv.FormatInt(idInt, 10)
			} else if idFloat, ok := id.(float64); ok {
				nextCursor = strconv.FormatInt(int64(idFloat), 10)
			}
		}
	}

	// 返回原始结果和游标信息
	return &CursorPageResult[any]{
		Cursor:  nextCursor,
		HasMore: hasMore,
		Data:    result,
	}, nil
}

// NewCursorResult 创建游标分页结果
func NewCursorResult[T any](cursor string, hasMore bool, data T) *CursorPageResult[T] {
	return &CursorPageResult[T]{
		Cursor:  cursor,
		HasMore: hasMore,
		Data:    data,
	}
}
