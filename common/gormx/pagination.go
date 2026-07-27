package gormx

import (
	"context"
	"fmt"
	"reflect"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PageResult[T any] struct {
	Data       []T   `json:"data"`
	Total      int64 `json:"total"`
	Page       int64 `json:"page"`
	PageSize   int64 `json:"page_size"`
	TotalPages int64 `json:"total_pages"`
}

// QueryPageData 分页查询数据，不计算总数。
func QueryPageData[T any](db *gorm.DB, page, pageSize int64) ([]T, error) {
	return queryPageData[T](db, NewPageParams(page, pageSize))
}

func queryPageData[T any](db *gorm.DB, params PageParams) ([]T, error) {
	offset, ok := params.Offset()
	if !ok {
		return []T{}, nil
	}
	var dest []T
	if err := db.Offset(offset).Limit(int(params.PageSize())).Find(&dest).Error; err != nil {
		return nil, err
	}
	return dest, nil
}

func QueryPage[T any](db *gorm.DB, page, pageSize int64, dest *[]T) (*PageResult[T], error) {
	params := NewPageParams(page, pageSize)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	if !params.Contains(total) {
		data := []T{}
		*dest = data
		return newPageResult(data, total, params), nil
	}

	data, err := queryPageData[T](db, params)
	if err != nil {
		return nil, err
	}
	*dest = data
	return newPageResult(data, total, params), nil
}

func NewPageResult[T any](data []T, total int64, page, pageSize int64) *PageResult[T] {
	return newPageResult(data, total, NewPageParams(page, pageSize))
}

func newPageResult[T any](data []T, total int64, params PageParams) *PageResult[T] {
	return &PageResult[T]{
		Data:       data,
		Total:      total,
		Page:       params.Page(),
		PageSize:   params.PageSize(),
		TotalPages: params.TotalPages(total),
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
