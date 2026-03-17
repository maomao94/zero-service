package model

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// ============================
// 错误变量
// ============================
var (
	ErrNotFound     = sqlx.ErrNotFound
	ErrNoRowsUpdate = errors.New("update db no rows change")
)

// ============================
// 通用缓存结构
// ============================
type CacheEntry[T any] struct {
	Data  T
	Valid bool
}

// ============================
// 数据库执行结果适配（Postgres）
// ============================
type postgresResult struct {
	id int64
}

func (r *postgresResult) LastInsertId() (int64, error) {
	return r.id, nil
}

func (r *postgresResult) RowsAffected() (int64, error) {
	return 1, nil
}

// ============================
// 数据库类型
// ============================
type DatabaseType string

const (
	DatabaseTypeMySQL    DatabaseType = "mysql"
	DatabaseTypePostgres DatabaseType = "postgres"
	DatabaseTypeSQLite   DatabaseType = "sqlite"
	DatabaseTypeTAOS     DatabaseType = "taos"
)

const dbTag = "db"

// ============================
// 默认自动字段（插入时忽略）
// ============================
var defaultAutoFields = map[string]struct{}{
	"id": {},

	"create_at":   {},
	"create_time": {},
	"created_at":  {},

	"update_at":   {},
	"update_time": {},
	"updated_at":  {},
}

// ============================
// 反射缓存（解析 struct 字段）
// ============================
type fieldMeta struct {
	column string
	index  int
}

var structMetaCache sync.Map // map[reflect.Type][]fieldMeta

// ============================
// 正则缓存
// ============================
var pgPlaceholderRegexp = regexp.MustCompile(`\$\d+`)

// ============================
// 计划状态枚举
// ============================
const (
	//PlanStatusDisabled   int = 0 // 计划禁用
	PlanStatusEnabled    int = 1
	PlanStatusPaused     int = 2
	PlanStatusTerminated int = 3
)

// ============================
// 执行项调度状态枚举
// ============================
const (
	StatusWaiting    int = 0   // 等待调度，可扫表触发
	StatusDelayed    int = 10  // 延期等待（业务失败重试或业务延期），可扫表触发
	StatusRunning    int = 100 // 已下发/执行中，等待业务回调
	StatusPaused     int = 150 // 执行项暂停（不扫表、不触发）
	StatusCompleted  int = 200 // 执行完成，不再触发
	StatusTerminated int = 300 // 人工/策略/超过重试次数终止
)

// ============================
// 执行业务结果枚举
// ============================
const (
	ResultCompleted  string = "completed"  // 业务执行完成
	ResultTerminated string = "terminated" // 业务终止
	ResultFailed     string = "failed"     // 业务执行失败
	ResultDelayed    string = "delayed"    // 业务执行延期
	ResultOngoing    string = "ongoing"    // 业务正在执行（未回调或部分异步）
)

func getFieldMetas(t reflect.Type) []fieldMeta {
	if v, ok := structMetaCache.Load(t); ok {
		return v.([]fieldMeta)
	}

	var metas []fieldMeta
	for i := 0; i < t.NumField(); i++ {
		fi := t.Field(i)
		tagv := fi.Tag.Get(dbTag)
		if tagv == "-" {
			continue
		}
		if tagv == "" {
			tagv = fi.Name
		} else if strings.Contains(tagv, ",") {
			tagv = strings.Split(tagv, ",")[0]
		}
		tagv = strings.TrimSpace(tagv)
		tagv = strings.ToLower(tagv)
		metas = append(metas, fieldMeta{
			column: tagv,
			index:  i,
		})
	}

	structMetaCache.Store(t, metas)
	return metas
}

// ============================
// 列名包装
// ============================
func wrapColumn(col string, dbType DatabaseType) string {
	switch dbType {
	case DatabaseTypeMySQL:
		return "`" + col + "`"
	case DatabaseTypePostgres:
		return `"` + col + `"`
	default:
		return col
	}
}

// ============================
// SQL 占位符适配
// ============================
func adaptSQLPlaceholders(sqlTemplate string, dbType DatabaseType) string {
	if !strings.EqualFold(string(dbType), string(DatabaseTypePostgres)) {
		return pgPlaceholderRegexp.ReplaceAllString(sqlTemplate, "?")
	}
	return sqlTemplate
}

// ============================
// 生成列名和值
// ============================
func generateColumnsAndValues(in any, excludeFields []string) ([]string, []any) {
	v := reflect.ValueOf(in)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		panic(fmt.Errorf("InsertColumnsAndValues only accepts struct; got %T", in))
	}

	t := v.Type()
	metas := getFieldMetas(t)

	excludeMap := make(map[string]struct{}, len(defaultAutoFields)+len(excludeFields))
	for k := range defaultAutoFields {
		excludeMap[k] = struct{}{}
	}
	for _, f := range excludeFields {
		excludeMap[strings.ToLower(f)] = struct{}{}
	}

	columns := make([]string, 0, len(metas))
	values := make([]any, 0, len(metas))

	for _, m := range metas {
		if _, ok := excludeMap[m.column]; ok {
			continue
		}

		fv := v.Field(m.index)
		if fv.Kind() == reflect.Ptr {
			if fv.IsNil() {
				values = append(values, nil)
			} else {
				values = append(values, fv.Elem().Interface())
			}
		} else {
			values = append(values, fv.Interface())
		}

		columns = append(columns, m.column)
	}

	return columns, values
}

// ============================
// 生成 insert 列名和占位符
// ============================
func insertColumnsAndPlaceholders(in any, excludeFields []string, pg ...bool) ([]string, []string) {
	v := reflect.ValueOf(in)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		panic(fmt.Errorf("InsertColumnsAndPlaceholders only accepts struct; got %T", in))
	}

	usePG := len(pg) > 0 && pg[0]

	t := v.Type()
	metas := getFieldMetas(t)

	excludeMap := make(map[string]struct{}, len(defaultAutoFields)+len(excludeFields))
	for k := range defaultAutoFields {
		excludeMap[k] = struct{}{}
	}
	for _, f := range excludeFields {
		excludeMap[strings.ToLower(f)] = struct{}{}
	}

	columns := make([]string, 0, len(metas))
	placeholders := make([]string, 0, len(metas))

	idx := 1
	for _, m := range metas {
		if _, ok := excludeMap[m.column]; ok {
			continue
		}
		if usePG {
			columns = append(columns, m.column)
			placeholders = append(placeholders, "$"+strconv.Itoa(idx))
			idx++
		} else {
			columns = append(columns, "`"+m.column+"`")
			placeholders = append(placeholders, "?")
		}
	}

	return columns, placeholders
}

// ============================
// 生成 update 占位符
// ============================
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
