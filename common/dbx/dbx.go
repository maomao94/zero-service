package dbx

import (
	"strings"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlserver"
	_ "github.com/go-sql-driver/mysql" // MySQL驱动
	_ "github.com/lib/pq"              // PostgreSQL驱动
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/postgres"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type DatabaseType string

const (
	DatabaseTypeMySQL    DatabaseType = "mysql"
	DatabaseTypePostgres DatabaseType = "postgres"
	DatabaseTypeSQLite   DatabaseType = "sqlite"
	DatabaseTypeTAOS     DatabaseType = "taos"
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
		return DatabaseTypePostgres
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
	case DatabaseTypePostgres:
		return postgres.New(datasource, opts...)
	default: // DatabaseTypeMySQL and others
		return sqlx.NewMysql(datasource, opts...)
	}
}

func NewQoqu(datasource string, opts ...sqlx.SqlOption) *goqu.Database {
	var conn sqlx.SqlConn
	dbType := ParseDatabaseType(datasource)
	switch dbType {
	case DatabaseTypeSQLite:
		conn = NewSqlite(datasource, opts...)
	case DatabaseTypeTAOS:
		conn = NewTaos(datasource, opts...)
	case DatabaseTypePostgres:
		conn = postgres.New(datasource, opts...)
	default: // DatabaseTypeMySQL and others
		conn = sqlx.NewMysql(datasource, opts...)
	}
	db, _ := conn.RawDB()
	database := goqu.New(string(dbType), db)
	database.Logger(&QoquLog{})
	return database
}

type QoquLog struct {
}

func (log *QoquLog) Printf(format string, v ...any) {
	logx.Infof(format, v...)
}
