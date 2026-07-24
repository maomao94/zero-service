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

type DisableCronJobLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDisableCronJobLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableCronJobLogic {
	return &DisableCronJobLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 禁用 Cron Job，禁用后调度器不再扫描该任务
func (l *DisableCronJobLogic) DisableCronJob(in *trigger.DisableCronJobReq) (*trigger.DisableCronJobRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if err := l.svcCtx.CronJobStore.Disable(l.ctx, in.JobId); err != nil {
		if errors.Is(err, crontask.ErrNotFound) {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)
		}
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "禁用 Cron Job 失败")
	}
	return &trigger.DisableCronJobRes{}, nil
}
