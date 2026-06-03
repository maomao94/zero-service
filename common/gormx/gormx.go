package gormx

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	// 数据库连接地址，支持 MySQL/PostgreSQL/SQLite 自动识别。
	// MySQL:      user:pass@tcp(host:port)/db?charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai
	// PostgreSQL: postgres://user:pass@host:port/db?sslmode=disable&TimeZone=Asia/Shanghai
	// SQLite:     file:./data.db?cache=shared
	DataSource string `json:",optional"`
	// 最大空闲连接数，默认 100。建议与 MaxOpenConns 一致，避免连接抖动。
	MaxIdleConns int `json:",optional,default=100"`
	// 最大打开连接数，默认 100。根据 DB 最大连接数和实例数调整，公式: (db_max / 实例数) * 0.8。
	MaxOpenConns int `json:",optional,default=100"`
	// 连接最大生命周期，默认 1h。有负载均衡时建议缩短到 5-30min。
	ConnMaxLifetime time.Duration `json:",optional,default=1h"`
	// 空闲连接最大存活时间，默认 5min。低流量时自动清理闲置连接，防止被 DB 服务端断开。
	ConnMaxIdleTime time.Duration `json:",optional,default=5m"`
	// 慢 SQL 阈值，默认 200ms。超过此时间的查询会被记录为慢查询。
	SlowThreshold time.Duration `json:",optional,default=200ms"`
	// 日志级别，默认 error。可选: silent / error / warn / info。
	// 生产建议 error 或 warn（warn 会额外记录慢查询）。
	LogLevel string `json:",optional,default=error"`
	// 是否脱敏 SQL 参数，默认 true。开启后日志中不打印查询参数值，防止泄露敏感数据（手机号、密码等）。
	ParameterizedQueries bool `json:",optional,default=true"`
	// 是否忽略 record not found 错误日志，默认 false。
	IgnoreRecordNotFoundError bool `json:",optional,default=false"`
	// 是否按字段名显式查询（SELECT col1, col2 而非 SELECT *），默认 false。
	QueryFields bool `json:",optional,default=false"`
	// 是否跳过默认事务包裹，默认 true。单条写操作不再自动 BEGIN/COMMIT，性能提升约 10-30%。
	// 需要多条操作原子性时，使用 db.Transact() 手动包裹。
	SkipDefaultTransaction bool `json:",optional,default=true"`
	// 是否缓存预编译语句，默认 false。开启后重复执行相同 SQL 时跳过解析阶段，降低延迟。
	// 注意：连接池切换或数据库重启后缓存会失效，某些驱动可能存在兼容性问题。
	PrepareStmt bool `json:",optional,default=false"`
	// OpenTelemetry 链路追踪配置。
	Trace TraceConfig `json:",optional"`
}

type DB struct {
	*gorm.DB
}

func (db *DB) WithContext(ctx context.Context) *DB {
	return &DB{DB: db.DB.WithContext(ctx)}
}

func (db *DB) ExplainSQL(queryFn func(tx *gorm.DB) *gorm.DB) string {
	return db.DB.ToSQL(queryFn)
}

func (db *DB) Transact(fn func(tx *DB) error) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		return fn(&DB{DB: tx})
	})
}

func (db *DB) WithTenant(ctx context.Context) *gorm.DB {
	return db.WithContext(ctx).DB.Scopes(TenantScope(ctx))
}

func (db *DB) WithTenantStrict(ctx context.Context) *gorm.DB {
	return db.WithContext(ctx).DB.Scopes(TenantScopeStrict(ctx))
}

func (db *DB) WithDeleted(ctx context.Context) *gorm.DB {
	return db.WithContext(ctx).DB.Unscoped()
}

func (db *DB) WithTenantDeleted(ctx context.Context) *gorm.DB {
	return db.WithContext(ctx).DB.Scopes(TenantScopeWithDelete(ctx))
}

func (db *DB) AutoMigrate(dst ...any) error {
	if len(dst) == 0 {
		return nil
	}
	if err := db.DB.Session(&gorm.Session{Logger: QuietGormLogger()}).AutoMigrate(dst...); err != nil {
		return err
	}
	logx.Infof("auto migrate %d tables success", len(dst))
	return nil
}

func (db *DB) MustAutoMigrate(dst ...any) {
	if err := db.AutoMigrate(dst...); err != nil {
		logx.Must(errors.Errorf("auto migrate failed: %v", err))
	}
}

type Option func(*dbOptions)

type dbOptions struct {
	dialector              *gorm.Dialector
	rawDB                  *sql.DB
	maxIdleConns           int
	maxOpenConns           int
	connMaxLifetime        time.Duration
	connMaxIdleTime        time.Duration
	gormLogger             logger.Interface
	queryFields            bool
	skipDefaultTransaction bool
	prepareStmt            bool
	openTelemetry          TraceConfig
}

func WithRawDB(pool *sql.DB) Option {
	return func(o *dbOptions) { o.rawDB = pool }
}

func WithMaxIdleConns(n int) Option {
	return func(o *dbOptions) { o.maxIdleConns = n }
}

func WithMaxOpenConns(n int) Option {
	return func(o *dbOptions) { o.maxOpenConns = n }
}

func WithConnMaxLifetime(d time.Duration) Option {
	return func(o *dbOptions) { o.connMaxLifetime = d }
}

func WithConnMaxIdleTime(d time.Duration) Option {
	return func(o *dbOptions) { o.connMaxIdleTime = d }
}

func WithLogger(l logger.Interface) Option {
	return func(o *dbOptions) { o.gormLogger = l }
}

func WithQueryFields(b bool) Option {
	return func(o *dbOptions) { o.queryFields = b }
}

func WithSkipDefaultTransaction(b bool) Option {
	return func(o *dbOptions) { o.skipDefaultTransaction = b }
}

func WithPrepareStmt(b bool) Option {
	return func(o *dbOptions) { o.prepareStmt = b }
}

func Open(dsn string, opts ...Option) (*DB, error) {
	options := &dbOptions{
		maxIdleConns:           100,
		maxOpenConns:           100,
		connMaxLifetime:        time.Hour,
		connMaxIdleTime:        5 * time.Minute,
		gormLogger:             DefaultGormLogger(),
		skipDefaultTransaction: true,
		prepareStmt:            false,
		openTelemetry:          defaultOpenTelemetryConfig(),
	}
	for _, opt := range opts {
		opt(options)
	}

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

	gormConfig := &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   options.skipDefaultTransaction,
		PrepareStmt:                              options.prepareStmt,
		Logger:                                   options.gormLogger,
		QueryFields:                              options.queryFields,
	}
	if options.rawDB != nil {
		gormConfig.ConnPool = options.rawDB
	}

	gormDB, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, err
	}

	if options.rawDB == nil {
		sqlDB, err := gormDB.DB()
		if err != nil {
			return nil, err
		}
		sqlDB.SetMaxIdleConns(options.maxIdleConns)
		sqlDB.SetMaxOpenConns(options.maxOpenConns)
		sqlDB.SetConnMaxLifetime(options.connMaxLifetime)
		sqlDB.SetConnMaxIdleTime(options.connMaxIdleTime)
	}

	RegisterCallbacks(gormDB)
	if err := registerOpenTelemetry(gormDB, options.openTelemetry); err != nil {
		return nil, err
	}
	return &DB{DB: gormDB}, nil
}

func MustOpen(dsn string, opts ...Option) *DB {
	db, err := Open(dsn, opts...)
	logx.Must(err)
	return db
}

func OpenWithConf(conf Config) (*DB, error) {
	if conf.DataSource == "" {
		return nil, errors.New("data source is required")
	}
	return Open(conf.DataSource,
		WithMaxIdleConns(conf.MaxIdleConns),
		WithMaxOpenConns(conf.MaxOpenConns),
		WithConnMaxLifetime(conf.ConnMaxLifetime),
		WithConnMaxIdleTime(conf.ConnMaxIdleTime),
		WithLogger(NewGormLogger(LoggerConfig{
			LogLevel:                  parseLogLevel(conf.LogLevel),
			SlowThreshold:             conf.SlowThreshold,
			ParameterizedQueries:      conf.ParameterizedQueries,
			IgnoreRecordNotFoundError: conf.IgnoreRecordNotFoundError,
		})),
		WithQueryFields(conf.QueryFields),
		WithSkipDefaultTransaction(conf.SkipDefaultTransaction),
		WithPrepareStmt(conf.PrepareStmt),
		WithOpenTelemetryConfig(conf.Trace),
	)
}

func MustOpenWithConf(conf Config) *DB {
	db, err := OpenWithConf(conf)
	logx.Must(err)
	return db
}

func OpenWithDialector(dialector *gorm.Dialector, opts ...Option) (*DB, error) {
	opts = append(opts, func(o *dbOptions) { o.dialector = dialector })
	return Open("", opts...)
}

func MustOpenWithDialector(dialector *gorm.Dialector, opts ...Option) *DB {
	db, err := OpenWithDialector(dialector, opts...)
	logx.Must(err)
	return db
}

func OpenWithRawDB(sqlDB *sql.DB, dbType DatabaseType, opts ...Option) (*DB, error) {
	dialector, err := GetDialector(dbType, "")
	if err != nil {
		return nil, err
	}
	return OpenWithDialector(&dialector, append(opts, WithRawDB(sqlDB))...)
}

func MustOpenWithRawDB(sqlDB *sql.DB, dbType DatabaseType, opts ...Option) *DB {
	db, err := OpenWithRawDB(sqlDB, dbType, opts...)
	logx.Must(err)
	return db
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
