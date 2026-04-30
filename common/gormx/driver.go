package gormx

import (
	"strings"

	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DatabaseType string

const (
	DatabaseMySQL    DatabaseType = "mysql"
	DatabasePostgres DatabaseType = "postgres"
	DatabaseSQLite   DatabaseType = "sqlite"
)

func ParseDatabaseType(dsn string) DatabaseType {
	dsn = strings.TrimSpace(dsn)
	lower := strings.ToLower(dsn)
	if lower == "" {
		return DatabaseMySQL
	}
	if strings.HasPrefix(lower, "sqlite://") || strings.HasPrefix(lower, "sqlite3://") || strings.HasPrefix(lower, "file:") || strings.Contains(lower, ".db") || strings.Contains(lower, ".sqlite") || strings.Contains(lower, ":memory:") {
		return DatabaseSQLite
	}
	if strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://") || strings.HasPrefix(lower, "postgres ") || strings.Contains(lower, "sslmode=") || strings.Contains(lower, ":5432") {
		return DatabasePostgres
	}
	if strings.HasPrefix(lower, "mysql://") || strings.Contains(lower, "@tcp(") || strings.Contains(lower, "charset=") || strings.Contains(lower, ":3306") {
		return DatabaseMySQL
	}
	return DatabaseMySQL
}

func GetDialector(dbType DatabaseType, dsn string) (gorm.Dialector, error) {
	switch dbType {
	case DatabaseMySQL:
		return mysql.Open(dsn), nil
	case DatabasePostgres:
		return postgres.Open(dsn), nil
	case DatabaseSQLite:
		return sqlite.Open(dsn), nil
	default:
		return nil, errors.Errorf("unsupported database type: %s", dbType)
	}
}

func Dialector(dbType DatabaseType, dsn string) (gorm.Dialector, error) {
	return GetDialector(dbType, dsn)
}

func DatabaseTypeOf(db *gorm.DB) DatabaseType {
	return GetDatabaseTypeFromDialector(db)
}

func GetDatabaseTypeFromDialector(db *gorm.DB) DatabaseType {
	if db == nil || db.Dialector == nil {
		return DatabaseMySQL
	}
	switch db.Dialector.(type) {
	case *mysql.Dialector:
		return DatabaseMySQL
	case *postgres.Dialector:
		return DatabasePostgres
	case *sqlite.Dialector:
		return DatabaseSQLite
	default:
		return DatabaseMySQL
	}
}
