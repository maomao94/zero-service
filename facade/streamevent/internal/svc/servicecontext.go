package svc

import (
	"zero-service/common/dbx"
	"zero-service/facade/streamevent/internal/config"
	"zero-service/model"

	_ "github.com/taosdata/driver-go/v3/taosWS"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var emptyDevicePointMapping = &model.DevicePointMapping{}

type ServiceContext struct {
	Config                  config.Config
	TaosConn                sqlx.SqlConn
	SqliteConn              sqlx.SqlConn
	DevicePointMappingModel model.DevicePointMappingModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	if c.DisableStmtLog {
		sqlx.DisableStmtLog()
	}
	svcCtx := &ServiceContext{
		Config: c,
	}
	svcCtx.TaosConn = dbx.NewTaos(c.TaosDB.DataSource)
	svcCtx.SqliteConn = dbx.NewSqlite(c.SqliteDB.DataSource)
	svcCtx.DevicePointMappingModel = model.NewDevicePointMappingModel(svcCtx.SqliteConn)
	return svcCtx
}
