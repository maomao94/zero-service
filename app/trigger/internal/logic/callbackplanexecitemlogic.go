package logic

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"zero-service/app/trigger/internal/execdelay"
	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/facade/streamevent/streamevent"
	"zero-service/model"

	"github.com/dromara/carbon/v2"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/songzhibin97/gkit/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/core/trace"
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

// 回调计划执行项 ongoing 回执
func (l *CallbackPlanExecItemLogic) CallbackPlanExecItem(in *trigger.CallbackPlanExecItemReq) (*trigger.CallbackPlanExecItemRes, error) {
	traceID := trace.TraceIDFromContext(l.ctx)
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
		if err == sqlx.ErrNotFound {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)
		}
		return nil, err
	}
	lockKey := fmt.Sprintf("trigger:lock:plan:exec:%s", execItem.ExecId)
	lock := redis.NewRedisLock(l.svcCtx.Redis, lockKey)
	b, lockErr := lock.AcquireCtx(l.ctx)
	if lockErr != nil {
		logx.WithContext(l.ctx).Errorf("%s CallbackPlanExecItem Redis 锁 Acquire 错误: %v", planscope.ExecCallback(execItem), lockErr)
		return nil, lockErr
	}
	if !b {
		lockScope := planscope.ExecCallback(execItem)
		logx.WithContext(l.ctx).Errorf("%s CallbackPlanExecItem 未抢到 Redis 锁", lockScope)
		err = fmt.Errorf("%s CallbackPlanExecItem 未抢到 Redis 锁", lockScope)
		return nil, err
	}
	defer lock.Release()
	// 补 查询一次
	execItem, err = l.svcCtx.PlanExecItemModel.FindOne(l.ctx, execItem.Id)
	if err != nil {
		return nil, err
	}
	// 查询计划
	plan, err := l.svcCtx.PlanModel.FindOne(l.ctx, execItem.PlanPk)
	if err != nil {
		return nil, err
	}
	// 查询计划批次
	batch, err := l.svcCtx.PlanBatchModel.FindOne(l.ctx, execItem.BatchPk)
	if err != nil {
		return nil, err
	}
	scope := planscope.CallbackScope(execItem, plan, batch)
	logx.WithContext(l.ctx).Infof("%s 收到执行回调 execResult=%s item_status=%d message=%s",
		scope, in.GetExecResult(), execItem.Status, in.Message)

	// 检查执行项状态是否为终态
	if execItem.Status == int64(model.StatusCompleted) || execItem.Status == int64(model.StatusTerminated) {
		return nil, errors.BadRequest("", "执行项已结束")
	}
	if execItem.Status != int64(model.StatusRunning) {
		return nil, errors.BadRequest("", "执行项状态错误")
	}
	// 执行事务
	err = l.svcCtx.PlanModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		var transErr error
		var reason = in.Reason
		switch in.GetExecResult() {
		case model.ResultCompleted:
			// 更新执行项状态为成功
			transErr = l.svcCtx.PlanExecItemModel.UpdateStatusToCompleted(ctx, execItem.Id, in.Message, in.Message,
				[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
			)
		case model.ResultFailed:
			// 更新执行项状态为失败
			transErr = l.svcCtx.PlanExecItemModel.UpdateStatusToFail(ctx, execItem.Id, model.ResultFailed, in.Message, "",
				[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
			)
		case model.ResultDelayed:
			currentTime := carbon.Now()
			dr := execdelay.Resolve(in.DelayConfig, in.Message, in.Reason, currentTime, execdelay.ModeDelayed)
			execdelay.LogWarnings(ctx, scope, dr)
			delayTriggerTime := dr.NextTrigger
			delayReason := execdelay.FinalReason(dr.ReasonStem, delayTriggerTime)
			reason = delayReason
			transErr = l.svcCtx.PlanExecItemModel.UpdateStatusToDelayed(ctx, execItem.Id, in.ExecResult, in.Message, delayReason, delayTriggerTime,
				[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
			)
		case model.ResultOngoing:
			currentTime := carbon.Now()
			or := execdelay.Resolve(in.DelayConfig, in.Message, in.Reason, currentTime, execdelay.ModeOngoing)
			execdelay.LogWarnings(ctx, scope, or)
			delayTriggerTime := or.NextTrigger
			delayReason := execdelay.FinalReason(or.ReasonStem, delayTriggerTime)
			reason = delayReason
			transErr = l.svcCtx.PlanExecItemModel.UpdateStatusToOngoing(ctx, execItem.Id, in.Message, delayReason, false, delayTriggerTime,
				[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
			)
		case model.ResultTerminated:
			transErr = l.svcCtx.PlanExecItemModel.UpdateStatusToTerminated(ctx, execItem.Id, in.Message, in.Reason,
				[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
			)
		default:
			return fmt.Errorf("invalid execResult: %s", in.GetExecResult())
		}
		if transErr != nil {
			return transErr
		}

		// 记录执行日志
		logEntry := &model.PlanExecLog{
			CreateUser:  sql.NullString{},
			UpdateUser:  sql.NullString{},
			DeptCode:    execItem.DeptCode,
			PlanPk:      execItem.PlanPk,
			PlanId:      execItem.PlanId,
			PlanName:    plan.PlanName,
			BatchPk:     execItem.BatchPk,
			BatchId:     execItem.BatchId,
			ItemPk:      execItem.Id,
			ExecId:      execItem.ExecId,
			ItemId:      execItem.ItemId,
			ItemType:    execItem.ItemType,
			ItemName:    execItem.ItemName,
			PointId:     execItem.PointId,
			TriggerTime: time.Now(),
			TraceId:     sql.NullString{String: traceID, Valid: traceID != ""},
			ExecResult:  sql.NullString{String: in.ExecResult, Valid: in.ExecResult != ""},
			Message:     sql.NullString{String: in.Message, Valid: in.Message != ""},
			Reason:      sql.NullString{String: reason, Valid: reason != ""},
		}
		// 插入执行日志
		if _, err := l.svcCtx.PlanExecLogModel.Insert(ctx, nil, logEntry); err != nil {
			logx.WithContext(ctx).Errorf("%s 插入 plan_exec_log 失败: %v", scope, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	batchCount, err := l.svcCtx.PlanBatchModel.UpdateBatchFinishedTime(l.ctx, execItem.BatchPk)
	if err != nil {
		logx.WithContext(l.ctx).Errorf("%s 更新 plan_batch.finished_time 失败: %v", scope, err)
	}
	if batchCount > 0 {
		batchNotifyReq := streamevent.NotifyPlanEventReq{
			EventType:  streamevent.PlanEventType_BATCH_FINISHED,
			PlanId:     execItem.PlanId,
			PlanType:   plan.Type.String,
			BatchId:    execItem.BatchId,
			Attributes: map[string]string{},
		}
		l.svcCtx.StreamEventCli.NotifyPlanEvent(l.ctx, &batchNotifyReq)
	}

	planCount, err := l.svcCtx.PlanModel.UpdateBatchFinishedTime(l.ctx, execItem.PlanPk)
	if err != nil {
		logx.WithContext(l.ctx).Errorf("%s 更新 plan.finished_time 失败: %v", scope, err)
	}
	if planCount > 0 {
		planPlanReq := streamevent.NotifyPlanEventReq{
			EventType: streamevent.PlanEventType_PLAN_FINISHED,
			PlanId:    execItem.PlanId,
			PlanType:  plan.Type.String,
			//BatchId:    execItem.BatchId,
			Attributes: map[string]string{},
		}
		l.svcCtx.StreamEventCli.NotifyPlanEvent(l.ctx, &planPlanReq)
	}

	return &trigger.CallbackPlanExecItemRes{}, nil
}
