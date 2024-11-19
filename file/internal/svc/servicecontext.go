package svc

import (
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"zero-service/file/internal/config"
	"zero-service/model"
)

type ServiceContext struct {
	Config   config.Config
	OssModel model.OssModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:   c,
		OssModel: model.NewOssModel(sqlx.NewMysql(c.DB.DataSource), c.Cache),
	}
}
