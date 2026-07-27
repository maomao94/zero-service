package gormx

import (
	"reflect"

	"gorm.io/gorm"
)

const (
	DefaultPage     = 1
	DefaultPageSize = 10
	MaxPageSize     = 500
)

// PageParams 保存归一化后的页码和页大小，并负责安全计算分页边界。
type PageParams struct {
	page     int64
	pageSize int64
}

// NewPageParams 应用默认页码、默认页大小和最大页大小。
func NewPageParams(page, pageSize int64) PageParams {
	if page <= 0 {
		page = DefaultPage
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return PageParams{page: page, pageSize: pageSize}
}

func (p PageParams) normalized() PageParams {
	return NewPageParams(p.page, p.pageSize)
}

// Page 返回归一化后的页码。
func (p PageParams) Page() int64 {
	return p.normalized().page
}

// PageSize 返回归一化后的页大小。
func (p PageParams) PageSize() int64 {
	return p.normalized().pageSize
}

// Offset 返回 GORM 可接受的 int Offset；超出当前平台 int 范围时返回 false。
func (p PageParams) Offset() (int, bool) {
	p = p.normalized()
	maxInt := int64(^uint(0) >> 1)
	if p.page-1 > maxInt/p.pageSize {
		return 0, false
	}
	return int((p.page - 1) * p.pageSize), true
}

// TotalPages 根据总记录数计算总页数。
func (p PageParams) TotalPages(total int64) int64 {
	p = p.normalized()
	if total <= 0 {
		return 0
	}
	pages := total / p.pageSize
	if total%p.pageSize != 0 {
		pages++
	}
	return pages
}

// Contains 判断当前页是否在总记录数对应的页码范围内。
func (p PageParams) Contains(total int64) bool {
	p = p.normalized()
	return total > 0 && p.page <= p.TotalPages(total)
}

func setSchemaColumn(db *gorm.DB, column string, value any) {
	if db.Statement.Schema == nil {
		return
	}
	if _, ok := db.Statement.Schema.FieldsByDBName[column]; !ok {
		return
	}
	db.Statement.SetColumn(column, value)
}

func mapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

func zeroValue(fieldType reflect.Type) any {
	return reflect.Zero(fieldType).Interface()
}
