package svc

import (
	"zero-service/app/gis/internal/config"
	"zero-service/app/gis/model"
	"zero-service/app/gis/model/gormmodel"
	"zero-service/common/gisx"
	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
)

type ServiceContext struct {
	Config     config.Config
	FenceStore gisx.FenceStore
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))
	var fenceStore gisx.FenceStore = &gisx.NoopFenceStore{}

	if c.DB.DataSource != "" {
		db := gormx.MustOpenWithConf(c.DB)
		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			db.MustAutoMigrate(
				&gormmodel.GisFence{},
				&gormmodel.GisFenceCell{},
			)
		}
		fenceStore = model.NewGormFenceStore(db)
		logx.Info("[gis] FenceStore: GORM")
	} else {
		logx.Info("[gis] FenceStore: Noop（未配置DB）")
	}

	return &ServiceContext{
		Config:     c,
		FenceStore: fenceStore,
	}
}
