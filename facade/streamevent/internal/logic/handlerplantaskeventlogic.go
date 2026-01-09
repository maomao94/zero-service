package logic

import (
	"context"

	"zero-service/facade/streamevent/internal/svc"
	"zero-service/facade/streamevent/streamevent"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

type HandlerPlanTaskEventLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewHandlerPlanTaskEventLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HandlerPlanTaskEventLogic {
	return &HandlerPlanTaskEventLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 计划任务事件处理
func (l *HandlerPlanTaskEventLogic) HandlerPlanTaskEvent(in *streamevent.HandlerPlanTaskEventReq) (*streamevent.HandlerPlanTaskEventRes, error) {
	return &streamevent.HandlerPlanTaskEventRes{
		ExecResult: 2,
		Message:    "延期",
		DelayConfig: &streamevent.PbDelayConfig{
			NextTriggerTime: carbon.Now().AddSeconds(500).ToDateTimeString(),
			DelayReason:     "延时 500秒",
		},
	}, nil
}
