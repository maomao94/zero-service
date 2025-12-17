package svc

import (
	"zero-service/common/taosx"
	"zero-service/facade/streamevent/internal/config"

	_ "github.com/taosdata/driver-go/v3/taosWS"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config   config.Config
	TaosConn sqlx.SqlConn
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlx.DisableStmtLog()
	svcCtx := &ServiceContext{
		Config: c,
	}
	if len(c.TaosDSN) > 0 {
		svcCtx.TaosConn = taosx.New(c.TaosDSN)
	}
	return svcCtx
}
