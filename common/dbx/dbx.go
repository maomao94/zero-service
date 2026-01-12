package dbx

import (
	"strings"

	_ "github.com/go-sql-driver/mysql" // MySQL驱动
	_ "github.com/lib/pq"              // PostgreSQL驱动
	"github.com/zeromicro/go-zero/core/stores/postgres"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type DatabaseType string

const (
	DatabaseTypeMySQL      DatabaseType = "mysql"
	DatabaseTypePostgreSQL DatabaseType = "postgresql"
	DatabaseTypeSQLite     DatabaseType = "sqlite"
	DatabaseTypeTAOS       DatabaseType = "taos"
)

func ParseDatabaseType(datasource string) DatabaseType {
	datasource = strings.TrimSpace(datasource)
	if strings.HasPrefix(datasource, "file:") || strings.Contains(datasource, ".db") {
		return DatabaseTypeSQLite
	} else if strings.Contains(datasource, "http") || strings.Contains(datasource, "https") {
		return DatabaseTypeTAOS
	} else if strings.Contains(datasource, "@tcp(") {
		return DatabaseTypeMySQL
	} else if strings.HasPrefix(strings.ToLower(datasource), "postgres") {
		return DatabaseTypePostgreSQL
	} else {
		return DatabaseTypeMySQL
	}
}

// New 根据数据源URL自动判断数据库类型并创建连接
// 支持的数据库类型：
// - SQLite: 数据源以"file:"开头或包含".db"后缀
// - TAOS: 数据源包含"http"或"https"
// - MySQL: 数据源包含":"和"@tcp("
// - PostgreSQL: 数据源以"postgres://"开头
func New(datasource string, opts ...sqlx.SqlOption) sqlx.SqlConn {
	dbType := ParseDatabaseType(datasource)
	switch dbType {
	case DatabaseTypeSQLite:
		return NewSqlite(datasource, opts...)
	case DatabaseTypeTAOS:
		return NewTaos(datasource, opts...)
	case DatabaseTypePostgreSQL:
		return postgres.New(datasource, opts...)
	default: // DatabaseTypeMySQL and others
		return sqlx.NewMysql(datasource, opts...)
	}
}
