package svc

import (
	"zero-service/app/ispserver/internal/config"
	isp "zero-service/app/ispserver/internal/isp"
)

type ServiceContext struct {
	Config    config.Config
	IspServer *isp.Server
}

func NewServiceContext(c config.Config) *ServiceContext {
	c.IspConf.ApplyDefaults()
	ispSrv, err := isp.NewServer(c.IspConf)
	if err != nil {
		panic(err)
	}
	return &ServiceContext{
		Config:    c,
		IspServer: ispSrv,
	}
}
