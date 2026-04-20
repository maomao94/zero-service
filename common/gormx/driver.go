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
	if strings.HasPrefix(dsn, "file:") || strings.Contains(dsn, ".db") || strings.Contains(dsn, ".sqlite") {
		return DatabaseSQLite
	}
	if strings.HasPrefix(dsn, "postgres") || strings.Contains(dsn, "pg ") || strings.Contains(dsn, "sslmode=") {
		return DatabasePostgres
	}
	if strings.Contains(dsn, "@tcp(") || strings.Contains(dsn, "charset=") || strings.Contains(dsn, "root:") {
		return DatabaseMySQL
	}
	lower := strings.ToLower(dsn)
	if strings.HasPrefix(lower, "mysql") || strings.Contains(lower, ":3306") {
		return DatabaseMySQL
	}
	if strings.HasPrefix(lower, "postgresql") || strings.Contains(lower, ":5432") {
		return DatabasePostgres
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
	switch db.Statement.Dialector.(type) {
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
