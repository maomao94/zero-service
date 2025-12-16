package svc

import (
	"database/sql"
	"zero-service/facade/streamevent/internal/config"

	_ "github.com/taosdata/driver-go/v3/taosWS"
)

type ServiceContext struct {
	Config config.Config
	TaoDB  *sql.DB
}

func NewServiceContext(c config.Config) *ServiceContext {
	svcCtx := &ServiceContext{
		Config: c,
	}
	if len(c.TaosDSN) > 0 {
		taos, err := sql.Open("taosWS", c.TaosDSN)
		if err != nil {
			panic(err)
		}
		svcCtx.TaoDB = taos
	}
	return svcCtx
}
