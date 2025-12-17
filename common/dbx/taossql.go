package dbx

import (
	_ "github.com/taosdata/driver-go/v3/taosRestful"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

const taosDriverName = "taosRestful"

func NewTaos(datasource string, opts ...sqlx.SqlOption) sqlx.SqlConn {
	return sqlx.NewSqlConn(taosDriverName, datasource, opts...)
}
