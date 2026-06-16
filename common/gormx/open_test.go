package gormx

import (
	"testing"
	"time"

	"gorm.io/gorm/logger"
)

func TestOpenWithConfPassesThroughConfig(t *testing.T) {
	db, err := OpenWithConf(Config{
		DataSource:             "file:" + t.Name() + "?mode=memory&cache=shared",
		MaxIdleConns:           50,
		MaxOpenConns:           50,
		ConnMaxLifetime:        30 * time.Minute,
		ConnMaxIdleTime:        2 * time.Minute,
		SlowThreshold:          100 * time.Millisecond,
		LogLevel:               "error",
		ParameterizedQueries:   true,
		SkipDefaultTransaction: true,
		Trace:                  TraceConfig{Disabled: true},
	})
	if err != nil {
		t.Fatalf("open with conf error = %v", err)
	}

	sqlDB, err := db.DB.DB()
	if err != nil {
		t.Fatalf("sql db error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if got := sqlDB.Stats().MaxOpenConnections; got != 50 {
		t.Fatalf("max open conns = %d, want 50", got)
	}
	if !db.DB.Config.SkipDefaultTransaction {
		t.Fatalf("skip default transaction = false, want true")
	}
	gormLogger, ok := db.DB.Config.Logger.(*gormLogger)
	if !ok {
		t.Fatalf("logger type = %T, want *gormLogger", db.DB.Config.Logger)
	}
	if gormLogger.cfg.LogLevel != logger.Error {
		t.Fatalf("log level = %v, want error", gormLogger.cfg.LogLevel)
	}
	if !gormLogger.cfg.ParameterizedQueries {
		t.Fatalf("parameterized queries = false, want true")
	}
}

func TestOpenWithConfPreservesExplicitFalseForGoZeroLoadedConfig(t *testing.T) {
	db, err := OpenWithConf(Config{
		DataSource:             "file:" + t.Name() + "?mode=memory&cache=shared",
		LogLevel:               "error",
		SkipDefaultTransaction: false,
		Trace:                  TraceConfig{Disabled: true},
	})
	if err != nil {
		t.Fatalf("open with conf error = %v", err)
	}

	sqlDB, err := db.DB.DB()
	if err != nil {
		t.Fatalf("sql db error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if db.DB.Config.SkipDefaultTransaction {
		t.Fatalf("skip default transaction = true, want false")
	}
}

func TestOpenWithDialectorRejectsNilDialector(t *testing.T) {
	_, err := OpenWithDialector(nil)
	if err == nil {
		t.Fatalf("expected error for nil dialector")
	}
}

func TestOpenWithRawDBRejectsNilRawDB(t *testing.T) {
	_, err := OpenWithRawDB(nil, DatabaseSQLite)
	if err == nil {
		t.Fatalf("expected error for nil raw db")
	}
}
