package asynqx

import (
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
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

func NewScheduler(addr, pass string, db int) *asynq.Scheduler {
	location, _ := time.LoadLocation("Asia/Shanghai")
	return asynq.NewScheduler(
		asynq.RedisClientOpt{
			Addr:         addr,
			Password:     pass,
			DB:           db,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			PoolSize:     50,
		}, &asynq.SchedulerOpts{
			Location: location,
			PostEnqueueFunc: func(info *asynq.TaskInfo, err error) {
				if err != nil {
					logx.Errorf("asynq scheduler err:%+v", err)
				}
			},
			Logger: &BaseLogger{},
		})
}

func (q *SchedulerServer) RegisterTest() {
	task := asynq.NewTask(SchedulerDeferTask, []byte("test"), asynq.Retention(7*24*time.Hour))
	entryID, err := q.Scheduler.Register("*/1 * * * *", task)
	if err != nil {
		logx.Errorf("asynq scheduleDelayTask err:%+v,task:%+v", err, task)
	}
	logx.Infow(fmt.Sprintf("asynq scheduleDelayTask registered %s", entryID), logx.Field("type", SchedulerDeferTask))
}
