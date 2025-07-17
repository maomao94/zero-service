package logic

import (
	"context"
	"github.com/hibiken/asynq"
	"github.com/jinzhu/copier"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/copierx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListScheduledTasksLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListScheduledTasksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListScheduledTasksLogic {
	return &ListScheduledTasksLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取预定任务列表
func (l *ListScheduledTasksLogic) ListScheduledTasks(in *trigger.ListScheduledTasksReq) (*trigger.ListScheduledTasksRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	tasks, err := l.svcCtx.AsynqInspector.ListScheduledTasks(
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
	return &trigger.ListScheduledTasksRes{
		TasksInfo: taskInfo,
		QueueInfo: &queueInfo,
	}, nil
}
