package task

import (
	"fmt"
	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/common/asynqx"
	"zero-service/zerorpc/internal/svc"
	"zero-service/zerorpc/internal/task/scheduler"
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
	mux.Use(svc.LoggingMiddleware)
	//defer task
	mux.Handle(asynqx.DeferDelayTask, NewDeferDelayTask(l.svcCtx))
	logx.Infow(fmt.Sprint("asynq cronJob-task registered"), logx.Field("type", asynqx.DeferDelayTask))

	mux.Handle(asynqx.DeferTriggerTask, NewDeferForwardTask(l.svcCtx))
	logx.Infow(fmt.Sprint("asynq cronJob-task registered"), logx.Field("type", asynqx.DeferTriggerTask))

	//scheduler job
	mux.Handle(asynqx.SchedulerDeferTask, scheduler.NewSchedulerDeferTask(l.svcCtx))
	logx.Infow(fmt.Sprint("asynq cronJob-scheduler registered"), logx.Field("type", asynqx.SchedulerDeferTask))
	return mux
}
