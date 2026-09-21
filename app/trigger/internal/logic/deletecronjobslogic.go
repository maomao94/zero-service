package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteCronJobsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteCronJobsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteCronJobsLogic {
	return &DeleteCronJobsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 按 JobId 批量软删除 Cron Job，重复删除按幂等成功处理
func (l *DeleteCronJobsLogic) DeleteCronJobs(in *trigger.DeleteCronJobsReq) (*trigger.DeleteCronJobsRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	deleted, err := l.svcCtx.CronJobStore.DeleteMany(l.ctx, in.JobIds)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "批量删除 Cron Job 失败")
	}
	return &trigger.DeleteCronJobsRes{Deleted: deleted}, nil
}
