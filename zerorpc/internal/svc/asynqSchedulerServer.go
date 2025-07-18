package svc

import (
	"context"
	"fmt"
	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
	"time"
	"zero-service/common/asynqx"
	"zero-service/zerorpc/internal/config"
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

func newScheduler(c config.Config) *asynq.Scheduler {
	location, _ := time.LoadLocation("Asia/Shanghai")
	return asynq.NewScheduler(
		asynq.RedisClientOpt{
			Addr:     c.Redis.Host,
			Password: c.Redis.Pass,
		}, &asynq.SchedulerOpts{
			Location: location,
			PostEnqueueFunc: func(info *asynq.TaskInfo, err error) {
				ctx := context.Background()
				ctx = logx.ContextWithFields(ctx, logx.Field("taskId", info.ID), logx.Field("type", info.Type))
				if err != nil {
					logx.WithContext(ctx).Errorf("asynq scheduler err:%+v", err)
				} else {
					logx.WithContext(ctx).Info("asynq scheduler success")
				}
			},
			Logger: &asynqx.BaseLogger{},
		})
}

func (q *SchedulerServer) Register() {
	task := asynq.NewTask(tasktype.SchedulerDeferTask, nil, asynq.Retention(24*time.Hour))
	entryID, err := q.Scheduler.Register("*/1 * * * *", task)
	if err != nil {
		logx.Errorf("asynq scheduleDelayTask err:%+v,task:%+v", err, task)
	}
	logx.Infow(fmt.Sprintf("asynq scheduleDelayTask registered %s", entryID), logx.Field("type", tasktype.SchedulerDeferTask))
}
