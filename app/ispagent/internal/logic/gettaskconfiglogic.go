package logic

import (
	"context"
	"errors"
	"strings"

	"zero-service/app/ispagent/internal/svc"
	"zero-service/app/ispagent/ispagent"
	"zero-service/app/ispagent/model/gormmodel"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type GetTaskConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetTaskConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTaskConfigLogic {
	return &GetTaskConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetTaskConfig 按任务编码查询配置详情
func (l *GetTaskConfigLogic) GetTaskConfig(in *ispagent.GetTaskConfigReq) (*ispagent.GetTaskConfigRes, error) {
	taskCode := strings.TrimSpace(in.GetTaskCode())
	if taskCode == "" {
		return nil, errors.New("task_code 不能为空")
	}
	var record gormmodel.GormTaskConfig
	if err := l.svcCtx.DB.WithContext(l.ctx).Where("task_code = ?", taskCode).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("任务配置不存在")
		}
		return nil, err
	}
	item, err := toTaskConfigItemPb(&record)
	if err != nil {
		return nil, err
	}
	return &ispagent.GetTaskConfigRes{Item: item}, nil
}
