package gormx

import (
	"math"

	"gorm.io/gorm"
)

type PageResult[T any] struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
	Data       T     `json:"data"`
}

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

func EmptyPageResult[T any](page, pageSize int) *PageResult[T] {
	return &PageResult[T]{
		Total:      0,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: 0,
		Data:       *new(T),
	}
}

type CursorPageResult[T any] struct {
	Cursor  string `json:"cursor"`
	HasMore bool   `json:"has_more"`
	Data    T      `json:"data"`
}

func CursorPage[T any](
	db *gorm.DB,
	pageSize int,
	cursor string,
	result *[]T,
	whereFn func(db *gorm.DB, cursor string) *gorm.DB,
	cursorFn func(row T) string,
) (*CursorPageResult[[]T], error) {
	if pageSize <= 0 {
		pageSize = 10
	}

	query := whereFn(db, cursor).Limit(pageSize + 1)
	if err := query.Find(result).Error; err != nil {
		return nil, err
	}

	data := *result
	hasMore := len(data) > pageSize
	nextCursor := ""

	if hasMore {
		nextCursor = cursorFn(data[pageSize-1])
		data = data[:pageSize]
		*result = data
	} else if len(data) > 0 {
		nextCursor = cursorFn(data[len(data)-1])
	}

	return &CursorPageResult[[]T]{
		Cursor:  nextCursor,
		HasMore: hasMore,
		Data:    data,
	}, nil
}

func NewCursorResult[T any](cursor string, hasMore bool, data T) *CursorPageResult[T] {
	return &CursorPageResult[T]{
		Cursor:  cursor,
		HasMore: hasMore,
		Data:    data,
	}
}
