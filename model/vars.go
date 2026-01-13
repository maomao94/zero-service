package model

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var ErrNotFound = sqlx.ErrNotFound
var ErrNoRowsUpdate = errors.New("update db no rows change")

type CacheEntry[T any] struct {
	Data  T
	Valid bool
}

type postgresResult struct {
	id int64
}

func (r *postgresResult) LastInsertId() (int64, error) {
	return r.id, nil
}

func (r *postgresResult) RowsAffected() (int64, error) {
	// Assuming one row is inserted
	return 1, nil
}

type DatabaseType string

const (
	dbTag                               = "db"
	DatabaseTypeMySQL      DatabaseType = "mysql"
	DatabaseTypePostgreSQL DatabaseType = "postgresql"
	DatabaseTypeSQLite     DatabaseType = "sqlite"
	DatabaseTypeTAOS       DatabaseType = "taos"
)

var defaultAutoFields = []string{
	"id",
	"create_at", "create_time", "created_at",
	"update_at", "update_time", "updated_at",
}

func insertColumnsAndPlaceholders(in any, excludeFields []string, pg ...bool) (columns []string, placeholders []string) {
	v := reflect.ValueOf(in)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		panic(fmt.Errorf("InsertColumnsAndPlaceholders only accepts struct; got %T", in))
	}
	usePG := false
	if len(pg) > 0 {
		usePG = pg[0]
	}
	excludeMap := make(map[string]struct{}, len(defaultAutoFields)+len(excludeFields))
	for _, f := range defaultAutoFields {
		excludeMap[f] = struct{}{}
	}
	for _, f := range excludeFields {
		excludeMap[f] = struct{}{}
	}
	typ := v.Type()
	placeholderIdx := 1
	for i := 0; i < v.NumField(); i++ {
		fi := typ.Field(i)
		tagv := fi.Tag.Get(dbTag)
		if tagv == "-" {
			continue
		}
		if tagv == "" {
			tagv = fi.Name
		} else if strings.Contains(tagv, ",") {
			tagv = strings.TrimSpace(strings.Split(tagv, ",")[0])
		}
		if tagv == "" {
			tagv = fi.Name
		}
		if _, ok := excludeMap[tagv]; ok {
			continue
		}
		if usePG {
			columns = append(columns, tagv)
			placeholders = append(placeholders, "$"+strconv.Itoa(placeholderIdx))
		} else {
			columns = append(columns, fmt.Sprintf("`%s`", tagv))
			placeholders = append(placeholders, "?")
		}
		placeholderIdx++
	}
	return
}

func generateColumnsAndValues(in any, excludeFields []string) (columns []string, values []any) {
	v := reflect.ValueOf(in)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		panic(fmt.Errorf("InsertColumnsAndValues only accepts struct; got %T", in))
	}
	excludeMap := make(map[string]struct{}, len(defaultAutoFields)+len(excludeFields))
	for _, f := range defaultAutoFields {
		excludeMap[f] = struct{}{}
	}
	for _, f := range excludeFields {
		excludeMap[f] = struct{}{}
	}
	typ := v.Type()
	for i := 0; i < v.NumField(); i++ {
		fi := typ.Field(i)
		tagv := fi.Tag.Get(dbTag)
		if tagv == "-" {
			continue
		}
		if tagv == "" {
			tagv = fi.Name
		} else if strings.Contains(tagv, ",") {
			tagv = strings.TrimSpace(strings.Split(tagv, ",")[0])
		}
		if tagv == "" {
			tagv = fi.Name
		}
		if _, ok := excludeMap[tagv]; ok {
			continue
		}
		columns = append(columns, tagv)
		values = append(values, v.Field(i).Interface())
	}
	return
}

func generateUpdatePlaceholders(fieldNames []string, isPostgreSQL bool) string {
	var placeholders []string
	for i, field := range fieldNames {
		var placeholder string
		if isPostgreSQL {
			placeholder = fmt.Sprintf("%s = $%d", field, i+1)
		} else {
			placeholder = fmt.Sprintf("%s = ?", field)
		}
		placeholders = append(placeholders, placeholder)
	}
	return strings.Join(placeholders, ", ")
}
