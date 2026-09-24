package gormx

import (
	"strings"

	dameng "github.com/godoes/gorm-dameng"
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
	DatabaseDM       DatabaseType = "dm"
	DatabaseKingbase DatabaseType = "kingbase"
	DatabaseH3       DatabaseType = "h3"
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
	if strings.HasPrefix(lower, "dm://") {
		return DatabaseDM
	}
	if strings.HasPrefix(lower, "kingbase://") {
		return DatabaseKingbase
	}
	if strings.HasPrefix(lower, "h3://") {
		return DatabaseH3
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
	case DatabaseDM:
		return dameng.Open(dsn), nil
	case DatabaseKingbase:
		return newKingbaseDialector(dsn)
	case DatabaseH3:
		return newH3Dialector(dsn)
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
	case *dameng.Dialector:
		return DatabaseDM
	default:
		return DatabaseMySQL
	}
}
