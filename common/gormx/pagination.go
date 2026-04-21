package gormx

import (
	"math"
	"reflect"
	"strconv"

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

func QueryPageWithTenant[T any](db *gorm.DB, page, pageSize int, result *T, whereFn func(db *gorm.DB) *gorm.DB) (*PageResult[T], error) {
	return QueryPage(whereFn(db), page, pageSize, result)
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

func CursorPageByID(db *gorm.DB, pageSize int, cursor string, result any, idField string) (*CursorPageResult[any], error) {
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

	if err := q.Limit(pageSize + 1).Find(result).Error; err != nil {
		return nil, err
	}

	rv := reflect.ValueOf(result)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	length := 0
	if rv.Kind() == reflect.Slice {
		length = rv.Len()
	}

	hasMore := length > pageSize
	nextCursor := ""

	if hasMore && length > 0 {
		lastIdx := pageSize - 1
		last := rv.Index(lastIdx)
		nextCursor = extractIDCursor(last, idField)
		rv.Set(rv.Slice(0, pageSize))
	} else if length > 0 {
		last := rv.Index(length - 1)
		nextCursor = extractIDCursor(last, idField)
	}

	return &CursorPageResult[any]{
		Cursor:  nextCursor,
		HasMore: hasMore,
		Data:    result,
	}, nil
}

func extractIDCursor(v reflect.Value, idField string) string {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		f := v.FieldByName("ID")
		if !f.IsValid() {
			f = v.FieldByName("Id")
		}
		if f.IsValid() {
			switch f.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return strconv.FormatInt(f.Int(), 10)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return strconv.FormatUint(f.Uint(), 10)
			case reflect.Float32, reflect.Float64:
				return strconv.FormatInt(int64(f.Float()), 10)
			case reflect.String:
				return f.String()
			}
		}
	}
	if v.Kind() == reflect.Map {
		key := v.MapIndex(reflect.ValueOf(idField))
		if key.IsValid() {
			iface := key.Interface()
			switch id := iface.(type) {
			case int64:
				return strconv.FormatInt(id, 10)
			case float64:
				return strconv.FormatInt(int64(id), 10)
			case uint:
				return strconv.FormatUint(uint64(id), 10)
			case string:
				return id
			}
		}
	}
	return ""
}

func NewCursorResult[T any](cursor string, hasMore bool, data T) *CursorPageResult[T] {
	return &CursorPageResult[T]{
		Cursor:  cursor,
		HasMore: hasMore,
		Data:    data,
	}
}
