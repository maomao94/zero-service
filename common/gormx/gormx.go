package gormx

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/syncx"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	DataSource    string          `json:",optional"`
	MaxIdleConns  int             `json:",optional,default=10"`
	MaxOpenConns  int             `json:",optional,default=100"`
	SlowThreshold time.Duration   `json:",optional,default=200ms"`
	Cache         cache.CacheConf `json:",optional"`
	LogLevel      string          `json:",optional,default=error"`
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

type DB struct {
	*gorm.DB
	cache cache.Cache
}

func (db *DB) WithContext(ctx context.Context) *DB {
	gormDB := db.DB.WithContext(ctx)
	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		gormDB = gormDB.Set("trace_id", spanCtx.TraceID().String())
	}
	return &DB{
		DB:    gormDB,
		cache: db.cache,
	}
}

func (db *DB) GetCache(key string, v any) error {
	if db.cache == nil {
		return errors.New("cache not configured")
	}
	return db.cache.Get(key, v)
}

func (db *DB) SetCache(key string, v any) error {
	if db.cache == nil {
		return errors.New("cache not configured")
	}
	return db.cache.Set(key, v)
}

func (db *DB) SetCacheWithExpire(key string, v any, expire time.Duration) error {
	if db.cache == nil {
		return errors.New("cache not configured")
	}
	return db.cache.SetWithExpire(key, v, expire)
}

func (db *DB) DelCache(keys ...string) error {
	if db.cache == nil {
		return errors.New("cache not configured")
	}
	return db.cache.Del(keys...)
}

func (db *DB) TakeCache(v any, key string, queryFn func(val any) error) error {
	if db.cache == nil {
		return queryFn(v)
	}
	return db.cache.Take(v, key, queryFn)
}

func (db *DB) Transaction(fn func(tx *gorm.DB) error, opts ...*sql.TxOptions) error {
	return db.DB.Transaction(fn, opts...)
}

func (db *DB) Transact(fn func(tx *DB) error) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		return fn(&DB{
			DB:    tx,
			cache: db.cache,
		})
	})
}

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

func WithMaxIdleConns(n int) Option {
	return func(o *dbOptions) {
		o.maxIdleConns = n
	}
}

func WithMaxOpenConns(n int) Option {
	return func(o *dbOptions) {
		o.maxOpenConns = n
	}
}

func WithConnMaxLifetime(d time.Duration) Option {
	return func(o *dbOptions) {
		o.connMaxLifetime = d
	}
}

func WithLogger(log logger.Interface) Option {
	return func(o *dbOptions) {
		o.logger = log
	}
}

func WithCache(c cache.Cache) Option {
	return func(o *dbOptions) {
		o.cache = c
	}
}

func WithQueryFields(b bool) Option {
	return func(o *dbOptions) {
		o.queryFields = b
	}
}

func Open(dsn string, opts ...Option) (*DB, error) {
	options := &dbOptions{
		maxIdleConns:    10,
		maxOpenConns:    100,
		connMaxLifetime: time.Hour,
		logger:          DefaultGormLogger(),
		queryFields:     false,
	}

	for _, opt := range opts {
		opt(options)
	}

	return openWithOptions(dsn, options)
}

func openWithOptions(dsn string, options *dbOptions) (*DB, error) {
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
		Logger:                                   options.logger,
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

	return &DB{
		DB:    gormDB,
		cache: options.cache,
	}, nil
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

	dbType := ParseDatabaseType(conf.DataSource)
	dialector, err := GetDialector(dbType, conf.DataSource)
	if err != nil {
		return nil, err
	}

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

	if len(conf.Cache) > 0 {
		exclusiveCalls := syncx.NewSingleFlight()
		stats := cache.NewStat("gorm_model")
		options.cache = cache.New(conf.Cache, exclusiveCalls, stats, gorm.ErrRecordNotFound)
	}

	return openWithOptions("", options)
}

func MustOpenWithConf(conf Config) *DB {
	db, err := OpenWithConf(conf)
	logx.Must(err)
	return db
}

func OpenWithDialector(dialector *gorm.Dialector, opts ...Option) (*DB, error) {
	opts = append(opts, func(o *dbOptions) {
		o.dialector = dialector
	})
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

func (db *DB) MustAutoMigrate(dst ...any) {
	if err := db.AutoMigrate(dst...); err != nil {
		wrapperErr := errors.Errorf("auto migrate failed: %v", err)
		logx.Must(wrapperErr)
	}
}
