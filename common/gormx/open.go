package gormx

import (
	"database/sql"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

func Open(dsn string, opts ...Option) (*DB, error) {
	options := defaultDBOptions()
	for _, opt := range opts {
		opt(options)
	}

	dialector, err := openDialector(dsn, options.dialector)
	if err != nil {
		return nil, err
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
		closeOpenedDB(gormDB, options.rawDB)
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
	if dialector == nil {
		return nil, errors.New("dialector is required")
	}
	opts = append(opts, func(o *dbOptions) { o.dialector = dialector })
	return Open("", opts...)
}

func MustOpenWithDialector(dialector *gorm.Dialector, opts ...Option) *DB {
	db, err := OpenWithDialector(dialector, opts...)
	logx.Must(err)
	return db
}

func OpenWithRawDB(sqlDB *sql.DB, dbType DatabaseType, opts ...Option) (*DB, error) {
	if sqlDB == nil {
		return nil, errors.New("raw sql db is required")
	}
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

func openDialector(dsn string, dialector *gorm.Dialector) (gorm.Dialector, error) {
	if dialector != nil {
		return *dialector, nil
	}
	if dsn == "" {
		return nil, errors.New("either dsn or WithDialector option is required")
	}
	dbType := ParseDatabaseType(dsn)
	return GetDialector(dbType, dsn)
}

func closeOpenedDB(gormDB *gorm.DB, rawDB *sql.DB) {
	if gormDB == nil || rawDB != nil {
		return
	}
	sqlDB, err := gormDB.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}
