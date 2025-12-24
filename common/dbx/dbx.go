package dbx

import (
	"strings"

	_ "github.com/go-sql-driver/mysql" // MySQL驱动
	_ "github.com/lib/pq"              // PostgreSQL驱动
	"github.com/zeromicro/go-zero/core/stores/postgres"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// New 根据数据源URL自动判断数据库类型并创建连接
// 支持的数据库类型：
// - SQLite: 数据源以"file:"开头或包含".db"后缀
// - TAOS: 数据源以"http://"或"https://"开头
// - MySQL: 数据源包含":"和"@tcp("
// - PostgreSQL: 数据源以"postgres://"开头
func New(datasource string, opts ...sqlx.SqlOption) sqlx.SqlConn {
	datasource = strings.TrimSpace(datasource)
	if strings.HasPrefix(datasource, "file:") || strings.Contains(datasource, ".db") {
		return NewSqlite(datasource, opts...)
	} else if strings.HasPrefix(datasource, "http://") || strings.HasPrefix(datasource, "https://") {
		return NewTaos(datasource, opts...)
	} else if strings.Contains(datasource, "@tcp(") {
		return sqlx.NewMysql(datasource, opts...)
	} else if strings.HasPrefix(strings.ToLower(datasource), "postgres") {
		return postgres.New(datasource, opts...)
	} else {
		return sqlx.NewMysql(datasource, opts...)
	}
}
