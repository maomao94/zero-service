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
	if strings.HasPrefix(lower, "sqlite://") || strings.HasPrefix(lower, "sqlite3://") || strings.HasPrefix(lower, "file:") || strings.HasPrefix(lower, ":memory:") {
		return DatabaseSQLite
	}
	if strings.HasPrefix(lower, "gaussdb://") {
		// Temporarily disabled: gorm.io/driver/gaussdb has timestamp compatibility
		// issues in PG-compatible mode. Use postgres:// DSNs for GaussDB instead.
		return DatabaseType("gaussdb")
	}
	if strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://") {
		return DatabasePostgres
	}
	if strings.HasPrefix(lower, "mysql://") {
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
