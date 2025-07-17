package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteAllArchivedTasksLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteAllArchivedTasksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteAllArchivedTasksLogic {
	return &DeleteAllArchivedTasksLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 删除所有已归档任务
func (l *DeleteAllArchivedTasksLogic) DeleteAllArchivedTasks(in *trigger.DeleteAllArchivedTasksReq) (*trigger.DeleteAllArchivedTasksRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	n, err := l.svcCtx.AsynqInspector.DeleteAllArchivedTasks(in.Queue)
	if err != nil {
		return nil, err
	}
	return &trigger.DeleteAllArchivedTasksRes{
		Count: int64(n),
	}, nil
}
