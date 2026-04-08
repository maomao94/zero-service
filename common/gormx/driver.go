package gormx

import (
	"strings"

	"github.com/zeromicro/go-zero/core/stores/cache"
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

// DsnConf DSN 配置（用于构建 DSN）
type DsnConf struct {
	Host     string // 主机地址
	Port     int    // 端口
	User     string // 用户名
	Password string // 密码
	DBName   string // 数据库名
	SSLMode  string // SSL 模式（postgres 专用）
}

// NewMySQLWithDsn 使用 DSN 配置创建 MySQL 连接
func NewMySQLWithDsn(conf DsnConf) (*gorm.DB, error) {
	dsn := conf.User + ":" + conf.Password + "@tcp(" + conf.Host + ":" + itoa(conf.Port) + ")/" + conf.DBName + "?charset=utf8mb4&parseTime=True&loc=Local"
	return NewMySQL(dsn)
}

// NewPostgres 使用 PostgreSQL DSN 创建连接
//
// DSN 格式：host=127.0.0.1 user=gorm password=gorm dbname=gorm port=5432 sslmode=disable
func NewPostgres(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
}

// NewPostgresWithDsn 使用配置创建 PostgreSQL 连接
func NewPostgresWithDsn(conf DsnConf) (*gorm.DB, error) {
	sslMode := conf.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	dsn := "host=" + conf.Host + " user=" + conf.User + " password=" + conf.Password + " dbname=" + conf.DBName + " port=" + itoa(conf.Port) + " sslmode=" + sslMode
	return NewPostgres(dsn)
}

// NewSQLite 使用 SQLite DSN 创建连接
//
// DSN 格式：file:app.db?cache=shared
func NewSQLite(dsn string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
}

// ParseDatabaseType 自动识别 DSN 的数据库类型
func ParseDatabaseType(dsn string) DatabaseType {
	dsn = strings.TrimSpace(dsn)

	if strings.HasPrefix(dsn, "file:") || strings.Contains(dsn, ".db") || strings.Contains(dsn, ".sqlite") {
		return DatabaseSQLite
	}
	if strings.HasPrefix(dsn, "postgres") || strings.Contains(dsn, "pg ") || strings.Contains(dsn, "sslmode=") {
		return DatabasePostgres
	}
	if strings.Contains(dsn, "@tcp(") || strings.Contains(dsn, "charset=") {
		return DatabaseMySQL
	}

	return DatabaseMySQL
}

// NewByDSN 根据 DSN 自动识别数据库类型并创建连接
func NewByDSN(dsn string) (*gorm.DB, error) {
	dbType := ParseDatabaseType(dsn)
	switch dbType {
	case DatabaseSQLite:
		return NewSQLite(dsn)
	case DatabasePostgres:
		return NewPostgres(dsn)
	default:
		return NewMySQL(dsn)
	}
}

// NewCachedConnByDSN 根据 DSN 创建带缓存的连接
func NewCachedConnByDSN(dsn string, cacheConf cache.CacheConf) (*CachedConn, error) {
	db, err := NewByDSN(dsn)
	if err != nil {
		return nil, err
	}
	return &CachedConn{
		Cache: cache.New(cacheConf, exclusiveCalls, stats, gorm.ErrRecordNotFound),
		DB:    db,
	}, nil
}

// itoa 简单的 int 转 string
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var result []byte
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}
	if negative {
		result = append([]byte{'-'}, result...)
	}
	return string(result)
}
