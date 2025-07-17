package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/zeromicro/go-zero/core/logx"
)

type RunTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRunTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RunTaskLogic {
	return &RunTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 运行任务
func (l *RunTaskLogic) RunTask(in *trigger.RunTaskReq) (*trigger.RunTaskRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.AsynqInspector.RunTask(in.Queue, in.Id); err != nil {
		return nil, err
	}
	return &trigger.RunTaskRes{}, nil
}
