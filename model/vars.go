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

// ============================
// 计划状态枚举
// ============================
const (
	PlanStatusDisabled   int = 0 // 计划禁用
	PlanStatusEnabled    int = 1 // 计划启用，可调度
	PlanStatusPaused     int = 2 // 计划暂停（不触发计划项）
	PlanStatusTerminated int = 3 // 计划终止（人工/策略终止）
)

// ============================
// 执行项调度状态枚举
// ============================
const (
	StatusWaiting int = 0   // 初始等待调度，可扫表触发
	StatusDelayed int = 10  // 延期等待（业务失败重试或业务延期），可扫表触发
	StatusRunning int = 100 // 已下发，等待业务回调，扫表时需判断超时

	// 暂停态
	StatusPaused int = 150 // 执行项暂停（不扫表、不触发）

	// 终态
	StatusCompleted  int = 200 // 执行完成终态，不再触发
	StatusTerminated int = 300 // 人工/策略/超过重试次数终止，终态
)

// 状态名称映射
var statusNames = map[int64]string{
	int64(StatusWaiting):    "等待调度",
	int64(StatusDelayed):    "延期等待",
	int64(StatusRunning):    "执行中",
	int64(StatusPaused):     "暂停",
	int64(StatusCompleted):  "完成",
	int64(StatusTerminated): "终止",
}

// ============================
// 执行业务结果枚举
// ============================
const (
	ResultCompleted string = "completed" // 业务执行完成
	ResultFailed    string = "failed"    // 业务执行失败
	ResultDelayed   string = "delayed"   // 业务执行延期
	ResultRunning   string = "runnging"  // 业务正在执行（未回调或部分异步）
)

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
