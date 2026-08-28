package asynqx

import (
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
		logx.Errorw("[asynq] scheduler run error", logx.Field("err", err))
		panic(err)
	}
}

func (q *SchedulerServer) Stop() {
	q.Scheduler.Shutdown()
}

func NewScheduler(addr, pass string, db int) *asynq.Scheduler {
	location, _ := time.LoadLocation("Asia/Shanghai")
	logx.Infow("[asynq] scheduler creating",
		logx.Field("addr", addr),
		logx.Field("db", db),
		logx.Field("location", location.String()),
	)
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
					logx.Errorw("[asynq] scheduler enqueue failed",
						logx.Field("type", info.Type),
						logx.Field("err", err),
					)
				} else {
					logx.Infow("[asynq] scheduler enqueue success",
						logx.Field("type", info.Type),
						logx.Field("taskId", info.ID),
						logx.Field("queue", info.Queue),
					)
				}
			},
			Logger: &BaseLogger{},
		})
}

func (q *SchedulerServer) RegisterTest() {
	task := asynq.NewTask(SchedulerDeferTask, []byte("test"), asynq.Retention(7*24*time.Hour))
	entryID, err := q.Scheduler.Register("*/1 * * * *", task)
	if err != nil {
		logx.Errorw("[asynq] scheduleDelayTask register failed",
			logx.Field("type", SchedulerDeferTask),
			logx.Field("err", err),
		)
		return
	}
	logx.Infow("[asynq] scheduleDelayTask registered",
		logx.Field("type", SchedulerDeferTask),
		logx.Field("entryID", entryID),
		logx.Field("cron", "*/1 * * * *"),
	)
}
