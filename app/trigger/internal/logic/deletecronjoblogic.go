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

type DeleteCronJobLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteCronJobLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteCronJobLogic {
	return &DeleteCronJobLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 软删除 Cron Job，重复删除按幂等成功处理
func (l *DeleteCronJobLogic) DeleteCronJob(in *trigger.DeleteCronJobReq) (*trigger.DeleteCronJobRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if err := l.svcCtx.CronJobStore.Delete(l.ctx, in.JobId); err != nil && !errors.Is(err, crontask.ErrNotFound) {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "删除 Cron Job 失败")
	}
	return &trigger.DeleteCronJobRes{}, nil
}
