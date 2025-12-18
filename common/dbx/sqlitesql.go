package dbx

import (
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	_ "modernc.org/sqlite"
)

const sqliteDriverName = "sqlite"

func NewSqlite(datasource string, opts ...sqlx.SqlOption) sqlx.SqlConn {
	return sqlx.NewSqlConn(sqliteDriverName, datasource, opts...)
}
