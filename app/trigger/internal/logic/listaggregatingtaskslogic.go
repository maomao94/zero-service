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

type ListAggregatingTasksLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAggregatingTasksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAggregatingTasksLogic {
	return &ListAggregatingTasksLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取聚合任务列表
func (l *ListAggregatingTasksLogic) ListAggregatingTasks(in *trigger.ListAggregatingTasksReq) (*trigger.ListAggregatingTasksRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	tasks, err := l.svcCtx.AsynqInspector.ListAggregatingTasks(
		in.Queue, in.Group, asynq.PageSize(int(in.PageSize)), asynq.Page(int(in.PageNum)))
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
	return &trigger.ListAggregatingTasksRes{
		TasksInfo: taskInfo,
		QueueInfo: &queueInfo,
	}, nil
}
