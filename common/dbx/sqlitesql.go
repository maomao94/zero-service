package dbx

import (
	_ "github.com/mattn/go-sqlite3"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

const sqliteDriverName = "sqlite3"

func NewSqlite(datasource string, opts ...sqlx.SqlOption) sqlx.SqlConn {
	return sqlx.NewSqlConn(sqliteDriverName, datasource, opts...)
}
