package gormx

import (
	"context"
	"fmt"
	"math"
	"reflect"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	if !isSafeCursorOrderColumn(orderColumn) {
		return nil, fmt.Errorf("invalid cursor order column: %s", orderColumn)
	}
	if limit <= 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	column := clause.Column{Name: orderColumn}
	if cursor != "" {
		db = db.Where(clause.Gt{Column: column, Value: cursor})
	}
	if err := db.Order(clause.OrderByColumn{Column: column}).Limit(limit + 1).Find(dest).Error; err != nil {
		return nil, err
	}
	hasMore := len(*dest) > limit
	if hasMore {
		*dest = (*dest)[:limit]
	}

	return &CursorPageResult[T]{
		Data:       *dest,
		NextCursor: nextCursorValue(db, orderColumn, *dest),
		HasMore:    hasMore,
	}, nil
}

func isSafeCursorOrderColumn(column string) bool {
	if column == "" {
		return false
	}
	for i, r := range column {
		if r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func nextCursorValue[T any](db *gorm.DB, orderColumn string, data []T) string {
	if len(data) == 0 || db == nil {
		return ""
	}
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(new(T)); err != nil || stmt.Schema == nil {
		return ""
	}
	field := stmt.Schema.FieldsByDBName[orderColumn]
	if field == nil {
		return ""
	}
	value, zero := field.ValueOf(context.Background(), reflect.ValueOf(data[len(data)-1]))
	if zero {
		return ""
	}
	return fmt.Sprint(value)
}
