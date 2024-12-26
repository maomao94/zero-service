package asynqx

import (
	"context"
	"fmt"
	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
	"time"
	"zero-service/zerorpc/tasktype"
)

type SchedulerServer struct {
	*asynq.Scheduler
}

func NewSchedulerServer(server *asynq.Scheduler) *SchedulerServer {
	return &SchedulerServer{
		Scheduler: server,
	}
}

func (q *SchedulerServer) Start() {
	if err := q.Scheduler.Run(); err != nil {
		logx.Errorf("asynq cronServer run err:%+v", err)
		panic(err)
	}
}

func (q *SchedulerServer) Stop() {
	q.Scheduler.Shutdown()
}

func NewScheduler(addr, pass string) *asynq.Scheduler {
	location, _ := time.LoadLocation("Asia/Shanghai")
	return asynq.NewScheduler(
		asynq.RedisClientOpt{
			Addr:     addr,
			Password: pass,
		}, &asynq.SchedulerOpts{
			Location: location,
			PostEnqueueFunc: func(info *asynq.TaskInfo, err error) {
				ctx := context.Background()
				if info != nil {
					ctx = logx.ContextWithFields(ctx, logx.Field("taskId", info.ID), logx.Field("type", info.Type))
				}
				if err != nil {
					logx.WithContext(ctx).Errorf("asynq scheduler err:%+v", err)
				} else {
					logx.WithContext(ctx).Info("asynq scheduler success")
				}
			},
		})
}

func (q *SchedulerServer) RegisterTest() {
	task := asynq.NewTask(tasktype.SchedulerDeferTask, []byte("test"), asynq.Retention(7*24*time.Hour))
	entryID, err := q.Scheduler.Register("*/1 * * * *", task)
	if err != nil {
		logx.Errorf("asynq scheduleDelayTask err:%+v,task:%+v", err, task)
	}
	logx.Infow(fmt.Sprintf("asynq scheduleDelayTask registered %s", entryID), logx.Field("type", tasktype.SchedulerDeferTask))
}
