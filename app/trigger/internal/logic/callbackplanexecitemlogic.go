package logic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"zero-service/app/trigger/internal/execdelay"
	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/internal/triggerutil"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"
	"zero-service/model"
	"zero-service/third_party/extproto"

	"github.com/dromara/carbon/v2"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/trace"
	"gorm.io/gorm"
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
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	if strutil.IsBlank(in.Id) && strutil.IsBlank(in.ExecId) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "参数错误")
	}

	db := l.svcCtx.DB.WithContext(l.ctx).DB
	var execItem gormmodel.PlanExecItem
	if !strutil.IsBlank(in.Id) {
		err = db.Where("id = ?", in.Id).First(&execItem).Error
	} else {
		err = db.Where("exec_id = ?", in.ExecId).First(&execItem).Error
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)
		}
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_DB, "查询执行项失败")
	}

	lockKey := fmt.Sprintf("trigger:lock:plan:exec:%s", execItem.ExecId)
	lock := redis.NewRedisLock(l.svcCtx.Redis, lockKey)
	timeoutMs := execItem.RequestTimeout
	if timeoutMs == 0 {
		timeoutMs = l.svcCtx.Config.StreamEventConf.Timeout
	}
	lock.SetExpire(triggerutil.RedisLockExpireSeconds(timeoutMs))
	b, lockErr := lock.AcquireCtx(l.ctx)
	if lockErr != nil {
		planscope.ExecCallback(&execItem).Logger(l.ctx).Errorf("RPC 执行回调：获取 Redis 分布式锁失败: %v", lockErr)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_03_CACHE, lockErr, "获取 Redis 分布式锁失败")
	}
	if !b {
		lockScope := planscope.ExecCallback(&execItem)
		lockScope.Logger(l.ctx).Error("RPC 执行回调：未获取到 Redis 锁（可能并发回调同一执行单）")
		return nil, tool.NewErrorByPbCode(extproto.Code__1_03_CACHE, "执行回调未获取到 Redis 锁")
	}
	defer lock.Release()

	// 加锁后补查询一次，获取最新状态
	if err := db.Where("id = ?", execItem.Id).First(&execItem).Error; err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询执行项失败")
	}

	var plan gormmodel.Plan
	if err := db.Where("id = ?", execItem.PlanPk).First(&plan).Error; err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询计划失败")
	}
	var planBatch gormmodel.PlanBatch
	if err := db.Where("id = ?", execItem.BatchPk).First(&planBatch).Error; err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询批次失败")
	}

	scope := planscope.CallbackScope(&execItem, &plan, &planBatch)
	log := scope.Logger(l.ctx)
	log.WithFields(
		logx.Field("exec_result", in.GetExecResult()),
		logx.Field("item_status", execItem.Status),
		logx.Field("message", in.Message),
	).Info("RPC 执行回调：收到下游回执，将按 exec_result 回写执行项与流水")

	if execItem.Status == int64(model.StatusCompleted) || execItem.Status == int64(model.StatusTerminated) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "执行项已结束")
	}
	if execItem.Status != int64(model.StatusRunning) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "执行项状态错误")
	}

	statusIn := []int{model.StatusRunning}
	statusOut := []int{model.StatusCompleted, model.StatusTerminated}

	err = db.Transaction(func(tx *gorm.DB) error {
		var transErr error
		var reason = in.Reason
		txCtx := tx
		switch in.GetExecResult() {
		case model.ResultCompleted:
			transErr = gormmodel.UpdateExecItemStatusToCompleted(l.ctx, txCtx, execItem.Id, in.Message, in.Reason, statusIn, statusOut)
		case model.ResultFailed:
			transErr = gormmodel.UpdateExecItemStatusToFail(l.ctx, txCtx, execItem.Id, model.ResultFailed, in.Message, in.Reason, statusIn, statusOut)
		case model.ResultDelayed:
			currentTime := carbon.Now()
			dr := execdelay.Resolve(in.DelayConfig, in.Message, in.Reason, currentTime, execdelay.ModeDelayed)
			execdelay.LogWarnings(l.ctx, scope, dr)
			delayReason := execdelay.FinalReason(dr.ReasonStem, dr.NextTrigger)
			reason = delayReason
			parsedTime := carbon.Parse(dr.NextTrigger)
			if parsedTime.Error != nil {
				return parsedTime.Error
			}
			transErr = gormmodel.UpdateExecItemStatusToDelayed(l.ctx, txCtx, execItem.Id, in.ExecResult, in.Message, delayReason, parsedTime.StdTime(), statusIn, statusOut)
		case model.ResultOngoing:
			currentTime := carbon.Now()
			or := execdelay.Resolve(in.DelayConfig, in.Message, in.Reason, currentTime, execdelay.ModeOngoing)
			execdelay.LogWarnings(l.ctx, scope, or)
			delayReason := execdelay.FinalReason(or.ReasonStem, or.NextTrigger)
			reason = delayReason
			var nextTime *time.Time
			if or.NextTrigger != "" {
				parsedTime := carbon.Parse(or.NextTrigger)
				if parsedTime.Error != nil {
					return parsedTime.Error
				}
				t := parsedTime.StdTime()
				nextTime = &t
			}
			transErr = gormmodel.UpdateExecItemStatusToOngoing(l.ctx, txCtx, execItem.Id, in.Message, delayReason, statusIn, statusOut, nextTime, false)
		case model.ResultTerminated:
			transErr = gormmodel.UpdateExecItemStatusToTerminated(l.ctx, txCtx, execItem.Id, in.Message, in.Reason, statusIn, statusOut)
		default:
			return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "无效的回执执行结果: "+in.GetExecResult())
		}
		if transErr != nil {
			return transErr
		}

		execLog := &gormmodel.PlanExecLog{
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
		if err := txCtx.Create(execLog).Error; err != nil {
			scope.Logger(l.ctx).Errorf("写入执行流水 plan_exec_log 失败: %v", err)
			return fmt.Errorf("写入执行流水 plan_exec_log 失败: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "回调事务失败")
	}

	batchCount, err := gormmodel.UpdatePlanBatchFinishedTime(l.ctx, db, execItem.BatchPk)
	if err != nil {
		scope.Logger(l.ctx).Errorf("更新批次 finished_time（用于收尾判断）失败: %v", err)
	}
	if batchCount > 0 {
		batchNotifyReq := streamevent.NotifyPlanEventReq{
			EventType:  streamevent.PlanEventType_BATCH_FINISHED,
			PlanId:     execItem.PlanId,
			PlanType:   plan.Type.String,
			BatchId:    execItem.BatchId,
			Attributes: map[string]string{},
		}
		scope.WithFields(logx.Field("notify_event", planscope.NotifyEventBatchFinished)).Logger(l.ctx).Info("下游通知：调用 NotifyPlanEvent（批次收尾）")
		l.svcCtx.StreamEventCli.NotifyPlanEvent(l.ctx, &batchNotifyReq)
	}

	planCount, err := gormmodel.UpdatePlanFinishedTime(l.ctx, db, execItem.PlanPk)
	if err != nil {
		scope.Logger(l.ctx).Errorf("更新计划 finished_time（用于收尾判断）失败: %v", err)
	}
	if planCount > 0 {
		planPlanReq := streamevent.NotifyPlanEventReq{
			EventType:  streamevent.PlanEventType_PLAN_FINISHED,
			PlanId:     execItem.PlanId,
			PlanType:   plan.Type.String,
			Attributes: map[string]string{},
		}
		scope.WithFields(logx.Field("notify_event", planscope.NotifyEventPlanFinished)).Logger(l.ctx).Info("下游通知：调用 NotifyPlanEvent（计划收尾）")
		l.svcCtx.StreamEventCli.NotifyPlanEvent(l.ctx, &planPlanReq)
	}

	return &trigger.CallbackPlanExecItemRes{}, nil
}
