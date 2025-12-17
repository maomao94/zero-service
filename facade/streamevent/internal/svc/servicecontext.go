package svc

import (
	"zero-service/common/dbx"
	"zero-service/facade/streamevent/internal/config"

	_ "github.com/taosdata/driver-go/v3/taosWS"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config     config.Config
	TaosConn   sqlx.SqlConn
	SqliteConn sqlx.SqlConn
}

func NewServiceContext(c config.Config) *ServiceContext {
	if c.DisableStmtLog {
		sqlx.DisableStmtLog()
	}
	svcCtx := &ServiceContext{
		Config: c,
	}
	if len(c.TaosDSN) > 0 {
		svcCtx.TaosConn = dbx.NewTaos(c.TaosDSN)
	}
	if len(c.Sqlite) > 0 {
		svcCtx.SqliteConn = dbx.NewSqlite(c.Sqlite)
	}
	return svcCtx
}
