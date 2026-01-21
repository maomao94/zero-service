package logic

import (
	"context"

	"zero-service/facade/streamevent/internal/svc"
	"zero-service/facade/streamevent/streamevent"

	"github.com/zeromicro/go-zero/core/logx"
)

type NotifyPlanEventLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewNotifyPlanEventLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyPlanEventLogic {
	return &NotifyPlanEventLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 通知计划任务事件
func (l *NotifyPlanEventLogic) NotifyPlanEvent(in *streamevent.NotifyPlanEventReq) (*streamevent.NotifyPlanEventRes, error) {
	// todo: add your logic here and delete this line

	return &streamevent.NotifyPlanEventRes{}, nil
}
