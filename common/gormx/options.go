package gormx

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

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

func defaultDBOptions() *dbOptions {
	return &dbOptions{
		maxIdleConns:           100,
		maxOpenConns:           100,
		connMaxLifetime:        time.Hour,
		connMaxIdleTime:        5 * time.Minute,
		gormLogger:             DefaultGormLogger(),
		skipDefaultTransaction: true,
		prepareStmt:            false,
		openTelemetry:          defaultOpenTelemetryConfig(),
	}
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
