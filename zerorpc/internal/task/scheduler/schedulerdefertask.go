package scheduler

import (
	"context"
	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/zerorpc/internal/svc"
)

type SchedulerDeferTaskHandler struct {
	svcCtx *svc.ServiceContext
}

func NewSchedulerDeferTask(svcCtx *svc.ServiceContext) *SchedulerDeferTaskHandler {
	return &SchedulerDeferTaskHandler{
		svcCtx: svcCtx,
	}
}

func (l *SchedulerDeferTaskHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	logx.WithContext(ctx).Infof("do scheduler something")
	t.ResultWriter().Write([]byte("scheduler something"))
	return nil
}
