package svc

import (
	"zero-service/app/lalhook/internal/config"
	"zero-service/model"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config          config.Config
	HlsTsFilesModel model.HlsTsFilesModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:          c,
		HlsTsFilesModel: model.NewHlsTsFilesModel(sqlx.NewMysql(c.DB.DataSource)),
	}
}
