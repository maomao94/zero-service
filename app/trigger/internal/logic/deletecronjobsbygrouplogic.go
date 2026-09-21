package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteCronJobsByGroupLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteCronJobsByGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteCronJobsByGroupLogic {
	return &DeleteCronJobsByGroupLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 按任务分组批量软删除 Cron Job，重复删除按幂等成功处理
func (l *DeleteCronJobsByGroupLogic) DeleteCronJobsByGroup(in *trigger.DeleteCronJobsByGroupReq) (*trigger.DeleteCronJobsByGroupRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	deleted, err := l.svcCtx.CronJobStore.DeleteManyByGroup(l.ctx, in.GroupIds)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "按分组删除 Cron Job 失败")
	}
	return &trigger.DeleteCronJobsByGroupRes{Deleted: deleted}, nil
}
