package svc

import (
	"zero-service/common/dbx"
	"zero-service/common/gormx"
	"zero-service/facade/streamevent/internal/config"
	"zero-service/model/gormmodel"

	_ "github.com/taosdata/driver-go/v3/taosWS"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var emptyDevicePointMapping = &gormmodel.GormDevicePointMapping{}

type ServiceContext struct {
	Config                  config.Config
	TaosConn                sqlx.SqlConn
	SqliteConn              sqlx.SqlConn
	DB                      *gormx.DB
	DevicePointMappingStore *gormmodel.DevicePointMappingStore
}

func NewServiceContext(c config.Config) *ServiceContext {
	if c.DisableStmtLog {
		sqlx.DisableStmtLog()
	}
	svcCtx := &ServiceContext{
		Config: c,
	}
	svcCtx.TaosConn = dbx.New(c.TaosDB.DataSource)
	svcCtx.DB = gormx.MustOpenWithConf(c.DB)
	svcCtx.DevicePointMappingStore = gormmodel.NewDevicePointMappingStore(svcCtx.DB)
	return svcCtx
}
