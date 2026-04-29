package gormx

import (
	"math"

	"gorm.io/gorm"
)

const (
	DefaultPage     = 1
	DefaultPageSize = 10
	MaxPageSize     = 500
)

type PageResult[T any] struct {
	Data       []T   `json:"data"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

func NormalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = DefaultPage
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return page, pageSize
}

func QueryPage[T any](db *gorm.DB, page, pageSize int, dest *[]T) (*PageResult[T], error) {
	page, pageSize = NormalizePage(page, pageSize)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Find(dest).Error; err != nil {
		return nil, err
	}

	return NewPageResult(*dest, total, page, pageSize), nil
}

func NewPageResult[T any](data []T, total int64, page, pageSize int) *PageResult[T] {
	page, pageSize = NormalizePage(page, pageSize)
	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}
	return &PageResult[T]{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

type CursorPageResult[T any] struct {
	Data       []T    `json:"data"`
	NextCursor string `json:"next_cursor"`
	HasMore    bool   `json:"has_more"`
}

func CursorPage[T any](db *gorm.DB, cursor string, limit int, orderColumn string, dest *[]T) (*CursorPageResult[T], error) {
	if limit <= 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	if cursor != "" {
		db = db.Where(orderColumn+" > ?", cursor)
	}
	if err := db.Order(orderColumn + " ASC").Limit(limit + 1).Find(dest).Error; err != nil {
		return nil, err
	}
	hasMore := len(*dest) > limit
	if hasMore {
		*dest = (*dest)[:limit]
	}

	return &CursorPageResult[T]{
		Data:    *dest,
		HasMore: hasMore,
	}, nil
}
