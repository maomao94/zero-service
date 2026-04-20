package gormx

import (
	"strings"

	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// DatabaseType 数据库类型
type DatabaseType string

const (
	DatabaseMySQL    DatabaseType = "mysql"
	DatabasePostgres DatabaseType = "postgres"
	DatabaseSQLite   DatabaseType = "sqlite"
)

// NewMySQL 创建MySQL连接（使用 gormx 日志配置）
func NewMySQL(dsn string) (*gorm.DB, error) {
	return gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   DefaultGormLogger(),
	})
}

// NewPostgres 使用 PostgreSQL DSN 创建连接
func NewPostgres(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   DefaultGormLogger(),
	})
}

// NewSQLite 使用 SQLite DSN 创建连接
func NewSQLite(dsn string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   DefaultGormLogger(),
	})
}

// ParseDatabaseType 自动识别 DSN 的数据库类型
func ParseDatabaseType(dsn string) DatabaseType {
	dsn = strings.TrimSpace(dsn)

	// SQLite 检测
	if strings.HasPrefix(dsn, "file:") || strings.Contains(dsn, ".db") || strings.Contains(dsn, ".sqlite") {
		return DatabaseSQLite
	}
	// PostgreSQL 检测
	if strings.HasPrefix(dsn, "postgres") || strings.Contains(dsn, "pg ") || strings.Contains(dsn, "sslmode=") {
		return DatabasePostgres
	}
	// MySQL 检测（默认）
	if strings.Contains(dsn, "@tcp(") || strings.Contains(dsn, "charset=") || strings.Contains(dsn, "root:") {
		return DatabaseMySQL
	}

	// 尝试根据端口或前缀判断
	lower := strings.ToLower(dsn)
	if strings.HasPrefix(lower, "mysql") || strings.Contains(lower, ":3306") {
		return DatabaseMySQL
	}
	if strings.HasPrefix(lower, "postgresql") || strings.Contains(lower, ":5432") {
		return DatabasePostgres
	}

	return DatabaseMySQL // 默认 MySQL
}

// GetDialector 根据数据库类型和DSN返回对应的gorm驱动
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

// GetDatabaseTypeFromDialector 从 gorm.DB 获取数据库类型
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
