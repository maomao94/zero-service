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
	DataSource         string        `json:",optional"`
	MaxIdleConns       int           `json:",optional,default=10"`
	MaxOpenConns       int           `json:",optional,default=100"`
	SlowThreshold      time.Duration `json:",optional,default=200ms"`
	LogLevel           string        `json:",optional,default=error"`
	LogQueryParameters bool          `json:",optional,default=false"`
	QueryFields        bool          `json:",optional,default=false"`
	Trace              TraceConfig   `json:",optional"`
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
	dialector       *gorm.Dialector
	rawDB           *sql.DB
	maxIdleConns    int
	maxOpenConns    int
	connMaxLifetime time.Duration
	gormLogger      logger.Interface
	queryFields     bool
	openTelemetry   TraceConfig
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

func WithLogger(l logger.Interface) Option {
	return func(o *dbOptions) { o.gormLogger = l }
}

func WithQueryFields(b bool) Option {
	return func(o *dbOptions) { o.queryFields = b }
}

func Open(dsn string, opts ...Option) (*DB, error) {
	options := &dbOptions{
		maxIdleConns:    10,
		maxOpenConns:    100,
		connMaxLifetime: time.Hour,
		gormLogger:      DefaultGormLogger(),
		openTelemetry:   defaultOpenTelemetryConfig(),
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
		WithLogger(NewGormLogger(LoggerConfig{
			LogLevel:           parseLogLevel(conf.LogLevel),
			SlowThreshold:      conf.SlowThreshold,
			LogQueryParameters: conf.LogQueryParameters,
		})),
		WithQueryFields(conf.QueryFields),
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
