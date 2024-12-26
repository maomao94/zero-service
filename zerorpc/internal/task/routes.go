package task

import (
	"fmt"
	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/zerorpc/internal/svc"
	"zero-service/zerorpc/internal/task/scheduler"
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
	mux.Use(svc.LoggingMiddleware)
	//defer task
	mux.Handle(tasktype.DeferDelayTask, NewDeferDelayTask(l.svcCtx))
	logx.Infow(fmt.Sprint("asynq cronJob-task registered"), logx.Field("type", tasktype.DeferDelayTask))

	mux.Handle(tasktype.DeferTriggerTask, NewDeferForwardTask(l.svcCtx))
	logx.Infow(fmt.Sprint("asynq cronJob-task registered"), logx.Field("type", tasktype.DeferTriggerTask))

	//scheduler job
	mux.Handle(tasktype.SchedulerDeferTask, scheduler.NewSchedulerDeferTask(l.svcCtx))
	logx.Infow(fmt.Sprint("asynq cronJob-scheduler registered"), logx.Field("type", tasktype.SchedulerDeferTask))
	return mux
}
