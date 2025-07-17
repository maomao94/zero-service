package logic

import (
	"context"
	"github.com/hibiken/asynq"
	"github.com/jinzhu/copier"
	"zero-service/common/copierx"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListArchivedTasksLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListArchivedTasksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListArchivedTasksLogic {
	return &ListArchivedTasksLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取已归档任务列表
func (l *ListArchivedTasksLogic) ListArchivedTasks(in *trigger.ListArchivedTasksReq) (*trigger.ListArchivedTasksRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	tasks, err := l.svcCtx.AsynqInspector.ListArchivedTasks(
		in.Queue, asynq.PageSize(int(in.PageSize)), asynq.Page(int(in.PageNum)))
	if err != nil {
		return nil, err
	}
	taskInfo := []*trigger.PbTaskInfo{}
	copier.CopyWithOption(&taskInfo, tasks, copierx.Option)
	qinfo, err := l.svcCtx.AsynqInspector.GetQueueInfo(in.Queue)
	if err != nil {
		return nil, err
	}
	queueInfo := trigger.PbQueueInfo{}
	copier.CopyWithOption(&queueInfo, qinfo, copierx.Option)
	return &trigger.ListArchivedTasksRes{
		TasksInfo: taskInfo,
		QueueInfo: &queueInfo,
	}, nil
}
