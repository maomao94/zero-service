package logic

import (
	"context"

	"zero-service/facade/streamevent/internal/svc"
	"zero-service/facade/streamevent/streamevent"

	"github.com/zeromicro/go-zero/core/logx"
)

type HandleCronJobEventLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewHandleCronJobEventLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HandleCronJobEventLogic {
	return &HandleCronJobEventLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 处理 Trigger RRULE Cron Job 到点事件，并返回明确业务回执
func (l *HandleCronJobEventLogic) HandleCronJobEvent(in *streamevent.HandleCronJobEventReq) (*streamevent.HandleCronJobEventRes, error) {
	return &streamevent.HandleCronJobEventRes{
		Receipt: streamevent.CronJobReceiptPb_CRON_JOB_RECEIPT_SUCCESS,
	}, nil
}
