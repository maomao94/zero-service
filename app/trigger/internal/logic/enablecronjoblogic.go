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

type EnableCronJobLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewEnableCronJobLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnableCronJobLogic {
	return &EnableCronJobLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 启用 Cron Job，下次执行时间由调度器统一计算
func (l *EnableCronJobLogic) EnableCronJob(in *trigger.EnableCronJobReq) (*trigger.EnableCronJobRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if l.svcCtx == nil || l.svcCtx.CronJobScheduler == nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, errors.New("cron job scheduler is not initialized"), "启用 Cron Job 失败")
	}
	if err := l.svcCtx.CronJobScheduler.Enable(l.ctx, in.JobId); err != nil {
		if errors.Is(err, crontask.ErrNotFound) {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)
		}
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "启用 Cron Job 失败")
	}
	return &trigger.EnableCronJobRes{}, nil
}
