package logic

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/model"

	"github.com/dromara/carbon/v2"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/songzhibin97/gkit/errors"
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
	if in.Id <= 0 && strutil.IsBlank(in.ExecId) {
		return nil, errors.BadRequest("", "参数错误")
	}
	// 查询执行项
	var execItem *model.PlanExecItem
	if in.Id > 0 {
		execItem, err = l.svcCtx.PlanExecItemModel.FindOne(l.ctx, in.Id)
	} else {
		execItem, err = l.svcCtx.PlanExecItemModel.FindOneByExecId(l.ctx, in.ExecId)
	}
	if err != nil {
		return nil, err
	}
	// 查询计划项
	plan, err := l.svcCtx.PlanModel.FindOne(l.ctx, execItem.PlanPk)
	if err != nil {
		return nil, err
	}

	// 检查执行项状态是否为终态
	if execItem.Status == int64(model.StatusCompleted) || execItem.Status == int64(model.StatusTerminated) {
		return &trigger.CallbackPlanExecItemRes{}, nil
	}
	// 执行事务
	err = l.svcCtx.PlanModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		var transErr error
		var reason = in.Reason
		switch in.GetExecResult() {
		case model.ResultCompleted:
			// 更新执行项状态为成功
			transErr = l.svcCtx.PlanExecItemModel.UpdateStatusToCompleted(ctx, execItem.Id, in.Message, in.Message)
		case model.ResultFailed:
			// 更新执行项状态为失败
			transErr = l.svcCtx.PlanExecItemModel.UpdateStatusToFail(ctx, execItem.Id, model.ResultFailed, in.Message, "")
		case model.ResultDelayed:
			currentTime := carbon.Now()
			delayTriggerTime := currentTime.AddMinutes(5).ToDateTimeString()
			delayReason := ""
			if len(in.Message) == 0 {
				delayReason = in.GetExecResult()
			} else {
				delayReason = in.Message
			}
			if in.DelayConfig == nil {
				l.Errorf("No delay config provided for exec item %d", execItem.Id)
			} else {
				if len(in.DelayConfig.DelayReason) != 0 {
					delayReason = fmt.Sprintf("reason: %s, message: %s", in.DelayConfig.DelayReason, in.Message)
				}
				delayTime := carbon.ParseByLayout(in.DelayConfig.NextTriggerTime, carbon.DateTimeLayout)
				isTrue := true
				if delayTime.Error != nil || delayTime.IsInvalid() {
					l.Errorf("Invalid delay time format for exec item %d: %s", execItem.Id, in.DelayConfig.NextTriggerTime)
					isTrue = false
				} else {
					if delayTime.Lt(currentTime) {
						l.Errorf("Delay time for exec item %d is in the past: %v, current time: %v", execItem.Id, delayTime.ToDateTimeString(), currentTime.ToDateTimeString())
						isTrue = false
					}
				}
				if isTrue {
					delayTriggerTime = delayTime.ToDateTimeString()
				}
			}
			delayReason = fmt.Sprintf("%s, delay time: %s", delayReason, delayTriggerTime)
			reason = delayReason
			transErr = l.svcCtx.PlanExecItemModel.UpdateStatusToDelayed(ctx, execItem.Id, in.ExecResult, in.Message, delayReason, delayTriggerTime)
		case model.ResultOngoing:
			l.Infof("Ongoing exec item %d", execItem.Id)
		default:
			return fmt.Errorf("invalid execResult: %s", in.GetExecResult())
		}
		if transErr != nil {
			return transErr
		}
		transErr = l.svcCtx.PlanBatchModel.UpdateBatchCompletedTime(ctx, execItem.BatchPk)
		if transErr != nil {
			return transErr
		}
		transErr = l.svcCtx.PlanModel.UpdatePlanCompletedTime(ctx, execItem.PlanPk)
		if transErr != nil {
			return transErr
		}

		// 记录执行日志
		logEntry := &model.PlanExecLog{
			PlanPk:      execItem.PlanPk,
			PlanId:      execItem.PlanId,
			PlanName:    plan.PlanName,
			BatchPk:     execItem.BatchPk,
			BatchId:     execItem.BatchId,
			ItemPk:      execItem.Id,
			ExecId:      execItem.ExecId,
			ItemId:      execItem.ItemId,
			ItemName:    execItem.ItemName,
			PointId:     execItem.PointId,
			TriggerTime: time.Now(),
			TraceId:     sql.NullString{String: "", Valid: false},
			ExecResult:  sql.NullString{String: in.ExecResult, Valid: in.ExecResult != ""},
			Message:     sql.NullString{String: in.Message, Valid: in.Message != ""},
			Reason:      sql.NullString{String: reason, Valid: reason != ""},
		}
		// 插入执行日志
		if _, err := l.svcCtx.PlanExecLogModel.Insert(ctx, nil, logEntry); err != nil {
			logx.Errorf("Callback Error inserting plan exec log for item %d: %v", execItem.Id, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &trigger.CallbackPlanExecItemRes{}, nil
}
