package task

import (
	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/app/trigger/internal/svc"
	"zero-service/common/asynqx"
	"zero-service/zerorpc/tasktype"
)

type CronJob struct {
	svcCtx *svc.ServiceContext
}

func NewCronJob(svcCtx *svc.ServiceContext) *CronJob {
	return &CronJob{
		svcCtx: svcCtx,
	}
}

func (l *CronJob) Register() *asynq.ServeMux {
	mux := asynq.NewServeMux()
	mux.Use(asynqx.LoggingMiddleware)
	mux.Handle(tasktype.DeferTriggerTask, NewDeferTriggerTask(l.svcCtx))
	logx.Infof("asynq cronJob-task registered: %s", tasktype.DeferTriggerTask)

	mux.Handle(tasktype.DeferTriggerProtoTask, NewDeferTriggerProtoTask(l.svcCtx))
	logx.Infof("asynq cronJob-task registered: %s", tasktype.DeferTriggerProtoTask)

	//scheduler job
	mux.Handle(tasktype.SchedulerDeferTask, NewDeferTriggerTask(l.svcCtx))
	logx.Infof("asynq cronJob-scheduler registered: %s", tasktype.SchedulerDeferTask)
	return mux
}
