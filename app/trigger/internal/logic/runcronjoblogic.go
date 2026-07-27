package logic

import (
	"context"
	"errors"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/crontask"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type RunCronJobLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRunCronJobLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RunCronJobLogic {
	return &RunCronJobLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 立即异步执行一次 Cron Job，不改变周期计划
func (l *RunCronJobLogic) RunCronJob(in *trigger.RunCronJobReq) (*trigger.RunCronJobRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	task, err := l.svcCtx.CronJobStore.GetByID(l.ctx, in.JobId)
	if err != nil {
		if errors.Is(err, crontask.ErrNotFound) {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)
		}
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询 Cron Job 失败")
	}
	if err := l.svcCtx.CronJobScheduler.RunNow(l.ctx, task.TaskCode); err != nil {
		if errors.Is(err, crontask.ErrNotFound) {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)
		}
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "立即执行 Cron Job 失败")
	}
	return &trigger.RunCronJobRes{}, nil
}
