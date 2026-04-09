package gormx

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/syncx"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config gormx配置
type Config struct {
	DataSource    string          `json:",optional"`
	MaxIdleConns  int             `json:",optional,default=10"`
	MaxOpenConns  int             `json:",optional,default=100"`
	SlowThreshold time.Duration   `json:",optional,default=200ms"`
	Cache         cache.CacheConf `json:",optional"`
	LogLevel      string          `json:",optional,default=error"` // silent | error | warn | info
	QueryFields   bool            `json:",optional,default=false"`
}

func parseLogLevel(level string) logger.LogLevel {
	switch level {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return logger.Error
	}
}

// DB GORM 连接封装，继承gorm.DB所有能力，扩展缓存、链路追踪等go-zero生态集成
//
// 设计说明：
// - 完全兼容GORM原生API，学习成本为0
// - 自动集成go-zero缓存、日志、链路追踪能力
// - 内置多租户、审计、软删除、分页、批量操作等通用能力
type DB struct {
	*gorm.DB
	cache cache.Cache
}

// WithContext 带上下文的数据库操作，自动传递链路追踪信息
func (db *DB) WithContext(ctx context.Context) *DB {
	traceID := trace.SpanContextFromContext(ctx).TraceID().String()
	gormDB := db.DB.WithContext(ctx)
	if traceID != "" {
		gormDB = gormDB.Set("trace_id", traceID)
	}
	return &DB{
		DB:    gormDB,
		cache: db.cache,
	}
}

// GetCache 获取缓存
func (db *DB) GetCache(key string, v any) error {
	if db.cache == nil {
		return errors.New("cache not configured")
	}
	return db.cache.Get(key, v)
}

// SetCache 设置缓存
func (db *DB) SetCache(key string, v any) error {
	if db.cache == nil {
		return errors.New("cache not configured")
	}
	return db.cache.Set(key, v)
}

// SetCacheWithExpire 设置带过期时间的缓存
func (db *DB) SetCacheWithExpire(key string, v any, expire time.Duration) error {
	if db.cache == nil {
		return errors.New("cache not configured")
	}
	return db.cache.SetWithExpire(key, v, expire)
}

// DelCache 删除缓存
func (db *DB) DelCache(keys ...string) error {
	if db.cache == nil {
		return errors.New("cache not configured")
	}
	return db.cache.Del(keys...)
}

// TakeCache 缓存不存在时执行查询并缓存
func (db *DB) TakeCache(v any, key string, queryFn func(val any) error) error {
	if db.cache == nil {
		return queryFn(v)
	}
	return db.cache.Take(v, key, queryFn)
}

// Transaction 事务（别名，兼容gorm原生命名）
func (db *DB) Transaction(fn func(tx *gorm.DB) error, opts ...*sql.TxOptions) error {
	return db.DB.Transaction(fn, opts...)
}

// Transact 事务（别名，符合go-zero命名风格）
func (db *DB) Transact(fn func(tx *DB) error) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		return fn(&DB{
			DB:    tx,
			cache: db.cache,
		})
	})
}

// ============ 分页查询方法 ============

// Paginate 分页 Scope，链式调用
//
// 使用示例：
//
//	var users []User
//	db.Where("status = ?", 1).Paginate(1, 10).Find(&users)
func (db *DB) Paginate(page, pageSize int) *DB {
	offset := (page - 1) * pageSize
	return &DB{
		DB:    db.Offset(offset).Limit(pageSize),
		cache: db.cache,
	}
}

// QueryPage 分页查询，返回完整分页结果
//
// 使用示例：
//
//	var users []User
//	pageResult, err := gormx.QueryPage(db, 1, 10, &users)
func (db *DB) QueryPage(page, pageSize int, result any) (*PageResult[any], error) {
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if err := db.Paginate(page, pageSize).Find(result).Error; err != nil {
		return nil, err
	}

	return &PageResult[any]{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Data:       result,
	}, nil
}

// QueryPageWithCache 带缓存的分页查询
func (db *DB) QueryPageWithCache(page, pageSize int, result any, cacheKey string) (*PageResult[any], error) {
	if db.cache == nil {
		return db.QueryPage(page, pageSize, result)
	}

	key := buildPageCacheKey(cacheKey, page, pageSize)
	var pageResult PageResult[any]
	if err := db.cache.Get(key, &pageResult); err == nil {
		// 缓存命中，查询数据
		if err := db.Paginate(page, pageSize).Find(result).Error; err != nil {
			return nil, err
		}
		pageResult.Data = result
		return &pageResult, nil
	}

	// 缓存未命中，查询并缓存
	resultPage, err := db.QueryPage(page, pageSize, result)
	if err != nil {
		return nil, err
	}
	_ = db.cache.SetWithExpire(key, resultPage, time.Minute*5) // 缓存5分钟
	return resultPage, nil
}

// ============ 批量操作方法 ============

// BatchInsert 批量插入
func (db *DB) BatchInsert(values any, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 100
	}
	return db.CreateInBatches(values, batchSize).Error
}

// BatchUpdateByIds 根据ID批量更新
func (db *DB) BatchUpdateByIds(updates []Ups) error {
	if len(updates) == 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		txDB := &DB{DB: tx, cache: db.cache}
		for _, up := range updates {
			id, ok := up["id"]
			if !ok {
				continue
			}
			updateData := make(map[string]any, len(up))
			for k, v := range up {
				if k != "id" {
					updateData[k] = v
				}
			}
			if err := txDB.Where("id = ?", id).Updates(updateData).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// BatchDeleteByIds 根据ID批量删除
func (db *DB) BatchDeleteByIds(model any, ids ...int64) error {
	if len(ids) == 0 {
		return nil
	}
	return db.Delete(model, ids).Error
}

// ============ 连接创建方法 ============

// Option 数据库连接配置选项
type Option func(*dbOptions)

type dbOptions struct {
	dialector       *gorm.Dialector
	rawDB           *sql.DB
	maxIdleConns    int
	maxOpenConns    int
	connMaxLifetime time.Duration
	logger          logger.Interface
	cache           cache.Cache
	queryFields     bool
}

func WithRawDB(pool *sql.DB) Option {
	return func(o *dbOptions) {
		o.rawDB = pool
	}
}

// WithMaxIdleConns 设置最大空闲连接数
func WithMaxIdleConns(n int) Option {
	return func(o *dbOptions) {
		o.maxIdleConns = n
	}
}

// WithMaxOpenConns 设置最大打开连接数
func WithMaxOpenConns(n int) Option {
	return func(o *dbOptions) {
		o.maxOpenConns = n
	}
}

// WithConnMaxLifetime 设置连接最大生命周期
func WithConnMaxLifetime(d time.Duration) Option {
	return func(o *dbOptions) {
		o.connMaxLifetime = d
	}
}

// WithLogger 设置日志
func WithLogger(log logger.Interface) Option {
	return func(o *dbOptions) {
		o.logger = log
	}
}

// WithCache 设置缓存
func WithCache(c cache.Cache) Option {
	return func(o *dbOptions) {
		o.cache = c
	}
}

// WithQueryFields 设置是否查询所有字段（默认 true）
func WithQueryFields(b bool) Option {
	return func(o *dbOptions) {
		o.queryFields = b
	}
}

// Open 根据 DSN 创建数据库连接
//
// 使用示例：
//
//	// 最简用法（使用默认连接池配置）
//	db, err := gormx.Open("root:password@tcp(localhost:3306)/test")
//
//	// 带连接池配置
//	db, err := gormx.Open(dsn,
//	    gormx.WithMaxIdleConns(10),
//	    gormx.WithMaxOpenConns(100),
//	)
func Open(dsn string, opts ...Option) (*DB, error) {
	// 默认配置
	options := &dbOptions{
		maxIdleConns:    10,
		maxOpenConns:    100,
		connMaxLifetime: time.Hour,
		logger:          DefaultGormLogger(),
		queryFields:     false,
	}

	// 应用选项
	for _, opt := range opts {
		opt(options)
	}

	return openWithOptions(dsn, options)
}

// openWithOptions 内部方法，使用已解析的 options
func openWithOptions(dsn string, options *dbOptions) (*DB, error) {
	// 获取 Dialector
	var dialector gorm.Dialector
	var err error
	if options.dialector != nil {
		dialector = *options.dialector
	} else if dsn != "" {
		dbType := ParseDatabaseType(dsn)
		dialector, err = GetDialector(dbType, dsn)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("either dsn or WithDialector option is required")
	}

	// 构建 GORM 配置
	gormConfig := &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   options.logger,
		QueryFields:                              options.queryFields,
	}
	if options.rawDB != nil {
		gormConfig.ConnPool = options.rawDB
	}

	// 打开连接
	gormDB, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, err
	}

	// 配置连接池
	if options.rawDB == nil {
		sqlDB, err := gormDB.DB()
		if err != nil {
			return nil, err
		}
		sqlDB.SetMaxIdleConns(options.maxIdleConns)
		sqlDB.SetMaxOpenConns(options.maxOpenConns)
		sqlDB.SetConnMaxLifetime(options.connMaxLifetime)
	}

	// 注册回调
	RegisterCallbacks(gormDB)

	return &DB{
		DB:    gormDB,
		cache: options.cache,
	}, nil
}

// MustOpen 根据 DSN 创建数据库连接，失败则 panic
func MustOpen(dsn string, opts ...Option) *DB {
	db, err := Open(dsn, opts...)
	logx.Must(err)
	return db
}

// OpenWithConf 根据配置创建数据库连接
//
// 使用示例：
//
//	db, err := gormx.OpenWithConf(c.DB)
func OpenWithConf(conf Config) (*DB, error) {
	if conf.DataSource == "" {
		return nil, errors.New("data source is required")
	}

	dbType := ParseDatabaseType(conf.DataSource)
	dialector, err := GetDialector(dbType, conf.DataSource)
	if err != nil {
		return nil, err
	}

	// 构建选项
	options := &dbOptions{
		dialector:       &dialector,
		maxIdleConns:    conf.MaxIdleConns,
		maxOpenConns:    conf.MaxOpenConns,
		connMaxLifetime: time.Hour,
		logger: NewGormLogger(LoggerConfig{
			LogLevel:      parseLogLevel(conf.LogLevel),
			SlowThreshold: conf.SlowThreshold,
		}),
		queryFields: conf.QueryFields,
	}

	// 配置缓存
	if len(conf.Cache) > 0 {
		exclusiveCalls := syncx.NewSingleFlight()
		stats := cache.NewStat("gorm_model")
		options.cache = cache.New(conf.Cache, exclusiveCalls, stats, gorm.ErrRecordNotFound)
	}

	return openWithOptions("", options)
}

// MustOpenWithConf 根据配置创建数据库连接，失败则 panic
func MustOpenWithConf(conf Config) *DB {
	db, err := OpenWithConf(conf)
	logx.Must(err)
	return db
}

// OpenWithDialector 使用 Dialector 创建数据库连接
func OpenWithDialector(dialector *gorm.Dialector, opts ...Option) (*DB, error) {
	opts = append(opts, func(o *dbOptions) {
		o.dialector = dialector
	})
	return Open("", opts...)
}

// MustOpenWithDialector 使用 Dialector 创建数据库连接，失败则 panic
func MustOpenWithDialector(dialector *gorm.Dialector, opts ...Option) *DB {
	db, err := OpenWithDialector(dialector, opts...)
	logx.Must(err)
	return db
}

// OpenWithRawDB 基于已有的 *sql.DB 实例创建 gormx.DB
//
// 使用示例：
//
//	db, err := gormx.OpenWithRawDB(sqlDB, gormx.TypeMySQL)
func OpenWithRawDB(sqlDB *sql.DB, dbType DatabaseType, opts ...Option) (*DB, error) {
	dialector, err := GetDialector(dbType, "")
	if err != nil {
		return nil, err
	}
	return OpenWithDialector(&dialector, append(opts, WithRawDB(sqlDB))...)
}

// MustOpenWithRawDB 基于已有的 *sql.DB 实例创建 gormx.DB，失败则 panic
func MustOpenWithRawDB(sqlDB *sql.DB, dbType DatabaseType, opts ...Option) *DB {
	db, err := OpenWithRawDB(sqlDB, dbType, opts...)
	logx.Must(err)
	return db
}

// ============ 表结构自动迁移 ============

// AutoMigrate 自动迁移表结构
//
// 使用示例：
//
//	db.AutoMigrate(&User{}, &Order{})
func (db *DB) AutoMigrate(dst ...any) error {
	if len(dst) == 0 {
		return nil
	}
	originalLogger := db.DB.Logger
	db.DB.Logger = QuietGormLogger()
	if err := db.DB.AutoMigrate(dst...); err != nil {
		db.DB.Logger = originalLogger
		return err
	}
	db.DB.Logger = originalLogger
	logx.Infof("auto migrate %d tables success", len(dst))
	return nil
}

// MustAutoMigrate 自动迁移，失败则panic
//
// 使用示例：
//
//	db.MustAutoMigrate(&User{}, &Order{})
func (db *DB) MustAutoMigrate(dst ...any) {
	if err := db.AutoMigrate(dst...); err != nil {
		wrapperErr := errors.Errorf("auto migrate failed: %v", err)
		logx.Must(wrapperErr)
	}
}

// MigrateLegacyFields 为现有表添加老项目兼容字段（Id/DelState/Version）
func (db *DB) MigrateLegacyFields(tableName string) error {
	fields := []struct {
		name    string
		colType string
		comment string
	}{
		{"id", "bigint NOT NULL AUTO_INCREMENT", "主键ID"},
		{"del_state", "bigint NOT NULL DEFAULT 0", "删除状态"},
		{"version", "bigint NOT NULL DEFAULT 0", "乐观锁版本号"},
	}

	for _, f := range fields {
		if err := addColumnIfNotExists(db.DB, tableName, f.name, f.colType, f.comment); err != nil {
			return err
		}
	}

	// 添加索引
	if err := addIndexIfNotExists(db.DB, tableName, "idx_del_state", "del_state"); err != nil {
		return err
	}

	logx.Infof("migrate legacy fields for table %s success", tableName)
	return nil
}

// MigrateAuditFields 为现有表添加审计字段
func (db *DB) MigrateAuditFields(tableName string) error {
	fields := []struct {
		name    string
		colType string
		comment string
	}{
		{"create_user", "bigint unsigned NOT NULL DEFAULT 0", "创建人ID"},
		{"create_name", "varchar(64) NOT NULL DEFAULT ''", "创建人姓名"},
		{"update_user", "bigint unsigned NOT NULL DEFAULT 0", "更新人ID"},
		{"update_name", "varchar(64) NOT NULL DEFAULT ''", "更新人姓名"},
	}

	for _, f := range fields {
		if err := addColumnIfNotExists(db.DB, tableName, f.name, f.colType, f.comment); err != nil {
			return err
		}
	}

	// 添加索引
	indexes := []struct {
		name    string
		columns string
	}{
		{"idx_create_user", "create_user"},
		{"idx_update_user", "update_user"},
	}
	for _, idx := range indexes {
		if err := addIndexIfNotExists(db.DB, tableName, idx.name, idx.columns); err != nil {
			return err
		}
	}

	logx.Infof("migrate audit fields for table %s success", tableName)
	return nil
}

// MigrateTenantField 为现有表添加租户字段
func (db *DB) MigrateTenantField(tableName string) error {
	if err := addColumnIfNotExists(db.DB, tableName, "tenant_id", "varchar(12) NOT NULL DEFAULT '000000'", "租户ID"); err != nil {
		return err
	}
	if err := addIndexIfNotExists(db.DB, tableName, "idx_tenant_id", "tenant_id"); err != nil {
		return err
	}
	logx.Infof("migrate tenant field for table %s success", tableName)
	return nil
}

// MigrateAllFields 为现有表添加所有扩展字段
func (db *DB) MigrateAllFields(tableName string) error {
	if err := db.MigrateTenantField(tableName); err != nil {
		return err
	}
	return db.MigrateAuditFields(tableName)
}

// MigrateBatch 批量迁移多个表
func (db *DB) MigrateBatch(tableNames []string, opts ...func(table string) error) {
	for _, table := range tableNames {
		for _, opt := range opts {
			if err := opt(table); err != nil {
				logx.Errorf("migrate %s failed: %v", table, err)
			}
		}
	}
}

// addColumnIfNotExists 添加列（如果不存在）
func addColumnIfNotExists(db *gorm.DB, tableName, columnName, colType, comment string) error {
	var count int64
	err := db.Raw(fmt.Sprintf(
		"SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = '%s' AND column_name = '%s'",
		tableName, columnName,
	)).Scan(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		logx.Infof("column %s.%s already exists, skip", tableName, columnName)
		return nil
	}
	sql := fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s COMMENT '%s'", tableName, columnName, colType, comment)
	if err := db.Exec(sql).Error; err != nil {
		return err
	}
	logx.Infof("add column %s.%s success", tableName, columnName)
	return nil
}

// addIndexIfNotExists 添加索引（如果不存在）
func addIndexIfNotExists(db *gorm.DB, tableName, indexName, columns string) error {
	var count int64
	err := db.Raw(fmt.Sprintf(
		"SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = '%s' AND index_name = '%s'",
		tableName, indexName,
	)).Scan(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		logx.Infof("index %s on %s already exists, skip", indexName, tableName)
		return nil
	}
	sql := fmt.Sprintf("ALTER TABLE `%s` ADD INDEX `%s` (`%s`)", tableName, indexName, columns)
	if err := db.Exec(sql).Error; err != nil {
		return err
	}
	logx.Infof("add index %s on %s success", indexName, tableName)
	return nil
}

// ============ Schema 操作方法 ============

// AlterColumn 修改列类型
func (db *DB) AlterColumn(tableName, field, fieldType string) error {
	dialect := db.getDialect()
	var sql string
	switch dialect {
	case "mysql":
		sql = fmt.Sprintf("ALTER TABLE `%s` MODIFY COLUMN `%s` %s", tableName, field, fieldType)
	case "postgres":
		sql = fmt.Sprintf("ALTER TABLE \"%s\" ALTER COLUMN \"%s\" TYPE %s", tableName, field, fieldType)
	default:
		sql = fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s", tableName, field, fieldType)
	}
	return db.Exec(sql).Error
}

// RenameColumn 重命名列
func (db *DB) RenameColumn(tableName, oldName, newName string) error {
	return db.Migrator().RenameColumn(tableName, oldName, newName)
}

// AddColumn 添加列
func (db *DB) AddColumn(tableName, field, fieldType string) error {
	if db.Migrator().HasColumn(tableName, field) {
		logx.Infof("column %s.%s already exists, skip", tableName, field)
		return nil
	}
	sql := fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s", tableName, field, fieldType)
	return db.Exec(sql).Error
}

// DropColumn 删除列
func (db *DB) DropColumn(tableName, field string) error {
	return db.Migrator().DropColumn(tableName, field)
}

// CreateIndex 创建索引
func (db *DB) CreateIndex(tableName, indexName string, fields []string) error {
	if db.Migrator().HasIndex(tableName, indexName) {
		logx.Infof("index %s on %s already exists, skip", indexName, tableName)
		return nil
	}
	sql := fmt.Sprintf("CREATE INDEX `%s` ON `%s` (%s)", indexName, tableName, strings.Join(fields, ", "))
	return db.Exec(sql).Error
}

// DropIndex 删除索引
func (db *DB) DropIndex(indexName string) error {
	return db.Exec(fmt.Sprintf("DROP INDEX `%s`", indexName)).Error
}

// CreateForeignKey 创建外键
func (db *DB) CreateForeignKey(table, field, references string) error {
	constraintName := "fk_" + table + "_" + field
	sql := fmt.Sprintf("ALTER TABLE `%s` ADD CONSTRAINT `%s` FOREIGN KEY (`%s`) REFERENCES %s",
		table, constraintName, field, references)
	return db.Exec(sql).Error
}

// DropForeignKey 删除外键
func (db *DB) DropForeignKey(table, constraintName string) error {
	sql := fmt.Sprintf("ALTER TABLE `%s` DROP FOREIGN KEY `%s`", table, constraintName)
	return db.Exec(sql).Error
}

// RenameTable 重命名表
func (db *DB) RenameTable(oldName, newName string) error {
	return db.Migrator().RenameTable(oldName, newName)
}

// DropTable 删除表
func (db *DB) DropTable(name string) error {
	return db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS `%s`", name)).Error
}

// HasTable 检查表是否存在
func (db *DB) HasTable(name string) bool {
	return db.Migrator().HasTable(name)
}

// getDialect 获取数据库类型
func (db *DB) getDialect() string {
	switch db.Statement.Dialector.(type) {
	default:
		return "mysql"
	}
}
