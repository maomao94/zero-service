package svc

import (
	"github.com/go-playground/validator/v10"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/core/threading"

	"zero-service/app/file/internal/config"
	"zero-service/model"
)

type ServiceContext struct {
	Config          config.Config
	Validate        *validator.Validate
	OssModel        model.OssModel
	ThumbTaskRunner *threading.TaskRunner
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:          c,
		Validate:        validator.New(),
		OssModel:        model.NewOssModel(sqlx.NewMysql(c.DB.DataSource)),
		ThumbTaskRunner: threading.NewTaskRunner(c.ThumbTaskConcurrency),
	}
}
