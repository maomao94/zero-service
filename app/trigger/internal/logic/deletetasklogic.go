package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteTaskLogic {
	return &DeleteTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 删除任务
func (l *DeleteTaskLogic) DeleteTask(in *trigger.DeleteTaskReq) (*trigger.DeleteTaskRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.AsynqInspector.DeleteTask(in.Queue, in.Id); err != nil {
		return nil, err
	}
	return &trigger.DeleteTaskRes{}, nil
}
