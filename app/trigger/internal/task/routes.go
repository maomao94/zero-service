package task

import (
	"zero-service/app/trigger/internal/svc"
	"zero-service/common/asynqx"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
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
	mux.Handle(asynqx.DeferTriggerTask, NewDeferTriggerTask(l.svcCtx))
	logx.Infof("asynq cronJob-task registered: %s", asynqx.DeferTriggerTask)

	mux.Handle(asynqx.DeferTriggerProtoTask, NewDeferTriggerProtoTask(l.svcCtx))
	logx.Infof("asynq cronJob-task registered: %s", asynqx.DeferTriggerProtoTask)

	//scheduler job
	mux.Handle(asynqx.SchedulerDeferTask, NewDeferTriggerTask(l.svcCtx))
	logx.Infof("asynq cronJob-scheduler registered: %s", asynqx.SchedulerDeferTask)
	return mux
}
