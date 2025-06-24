package task

import (
	"fmt"
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
	logx.Infow(fmt.Sprint("asynq cronJob-task registered"), logx.Field("type", tasktype.DeferTriggerTask))

	mux.Handle(tasktype.DeferTriggerProtoTask, NewDeferTriggerProtoTask(l.svcCtx))
	logx.Infow(fmt.Sprint("asynq cronJob-task registered"), logx.Field("type", tasktype.DeferTriggerProtoTask))

	//scheduler job
	mux.Handle(tasktype.SchedulerDeferTask, NewDeferTriggerTask(l.svcCtx))
	logx.Infow(fmt.Sprint("asynq cronJob-scheduler registered"), logx.Field("type", tasktype.SchedulerDeferTask))
	return mux
}
