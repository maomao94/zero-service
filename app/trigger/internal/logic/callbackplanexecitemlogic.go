package logic

import (
	"context"
	"fmt"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/model"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type CallbackPlanExecItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCallbackPlanExecItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CallbackPlanExecItemLogic {
	return &CallbackPlanExecItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 回调计划项
func (l *CallbackPlanExecItemLogic) CallbackPlanExecItem(in *trigger.CallbackPlanExecItemReq) (*trigger.CallbackPlanExecItemRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 查询执行项
	execItem, err := l.svcCtx.PlanExecItemModel.FindOne(l.ctx, in.Id)
	if err != nil {
		if err == sqlx.ErrNotFound {
			return &trigger.CallbackPlanExecItemRes{}, nil
		}
		return nil, err
	}

	// 检查执行项状态是否为终态
	if execItem.Status == int64(model.StatusCompleted) || execItem.Status == int64(model.StatusTerminated) {
		return &trigger.CallbackPlanExecItemRes{}, nil
	}
	// 执行事务
	err = l.svcCtx.PlanModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		var transErr error
		switch in.GetExecResult() {
		case model.ResultCompleted:
			// 更新执行项状态为成功
			transErr = l.svcCtx.PlanExecItemModel.UpdateStatusToCompleted(ctx, execItem.Id, in.Message)
		case model.ResultFailed:
			// 更新执行项状态为失败
			transErr = l.svcCtx.PlanExecItemModel.UpdateStatusToCallback(ctx, execItem.Id, model.ResultFailed, in.Message)
		case model.ResultDelayed, model.ResultRunning:
			currentTime := carbon.Now()
			delayTriggerTime := currentTime.AddMinutes(5).ToDateTimeString()
			delayReason := ""
			if len(in.Message) == 0 {
				delayReason = in.GetExecResult()
			} else {
				delayReason = in.Message
			}
			if in.DelayConfig == nil {
				logx.Errorf("No delay config provided for exec item %d", execItem.Id)
			} else {
				if len(in.DelayConfig.DelayReason) != 0 {
					delayReason = fmt.Sprintf("reason: %s, message: %s", in.DelayConfig.DelayReason, in.Message)
				}
				delayTime := carbon.ParseByLayout(in.DelayConfig.NextTriggerTime, carbon.DateTimeLayout)
				isTrue := true
				if delayTime.Error != nil || delayTime.IsInvalid() {
					logx.Errorf("Invalid delay time format for exec item %d: %s", execItem.Id, in.DelayConfig.NextTriggerTime)
					isTrue = false
				} else {
					if delayTime.Lt(currentTime) {
						logx.Errorf("Delay time for exec item %d is in the past: %v, current time: %v", execItem.Id, delayTime.ToDateTimeString(), currentTime.ToDateTimeString())
						isTrue = false
					}
				}
				if isTrue {
					delayTriggerTime = delayTime.ToDateTimeString()
				}
			}
			delayReason = fmt.Sprintf("%s, delay time: %s", delayReason, delayTriggerTime)
			transErr = l.svcCtx.PlanExecItemModel.UpdateStatusToDelayed(ctx, execItem.Id, in.ExecResult, delayReason, delayTriggerTime)
		default:
			return fmt.Errorf("invalid execResult: %s", in.GetExecResult())
		}
		if transErr != nil {
			return transErr
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &trigger.CallbackPlanExecItemRes{}, nil
}
