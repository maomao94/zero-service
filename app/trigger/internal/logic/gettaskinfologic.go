package logic

import (
	"context"
	"github.com/jinzhu/copier"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/copierx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTaskInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetTaskInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTaskInfoLogic {
	return &GetTaskInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取任务
func (l *GetTaskInfoLogic) GetTaskInfo(in *trigger.GetTaskInfoReq) (*trigger.GetTaskInfoRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	taskInfo, err := l.svcCtx.AsynqInspector.GetTaskInfo(in.Queue, in.Id)
	if err != nil {
		return nil, err
	}
	pbTaskInfo := trigger.PbTaskInfo{}
	copier.CopyWithOption(&pbTaskInfo, taskInfo, copierx.Option)
	pbTaskInfo.State = int32(taskInfo.State)
	return &trigger.GetTaskInfoRes{
		TaskInfo: &pbTaskInfo,
	}, nil
}
