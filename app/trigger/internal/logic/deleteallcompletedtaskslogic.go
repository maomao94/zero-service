package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteAllCompletedTasksLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteAllCompletedTasksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteAllCompletedTasksLogic {
	return &DeleteAllCompletedTasksLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 删除所有已完成任务
func (l *DeleteAllCompletedTasksLogic) DeleteAllCompletedTasks(in *trigger.DeleteAllCompletedTasksReq) (*trigger.DeleteAllCompletedTasksRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	n, err := l.svcCtx.AsynqInspector.DeleteAllCompletedTasks(in.Queue)
	if err != nil {
		return nil, err
	}
	return &trigger.DeleteAllCompletedTasksRes{
		Count: int64(n),
	}, nil
}
