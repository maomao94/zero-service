package logic

import (
	"context"
	"database/sql"
	"time"
	"zero-service/facade/streamevent/streamevent"

	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/tool"
	"zero-service/model"
	"zero-service/third_party/extproto"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type TerminatePlanBatchLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTerminatePlanBatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TerminatePlanBatchLogic {
	return &TerminatePlanBatchLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 终止计划批次
func (l *TerminatePlanBatchLogic) TerminatePlanBatch(in *trigger.TerminatePlanBatchReq) (*trigger.TerminatePlanBatchRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 检查参数
	if strutil.IsBlank(in.Id) && strutil.IsBlank(in.BatchId) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "参数错误")
	}

	// 查询计划批次
	db := l.svcCtx.DB.WithContext(l.ctx).DB
	var planBatch gormmodel.PlanBatch
	if !strutil.IsBlank(in.Id) {
		err = db.Where("id = ?", in.Id).First(&planBatch).Error
	} else {
		err = db.Where("batch_id = ?", in.BatchId).First(&planBatch).Error
	}
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询计划批次失败")
	}

	// 查询计划
	var plan gormmodel.Plan
	if err := db.Where("plan_id = ?", planBatch.PlanId).First(&plan).Error; err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询计划失败")
	}

	// 检查当前状态是否允许终止操作
	if plan.Status == int64(model.PlanStatusTerminated) || plan.FinishedTime.Valid {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划状态已结束,无需终止")
	}

	if planBatch.Status == int64(model.PlanStatusTerminated) || planBatch.FinishedTime.Valid {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划批次状态已结束,无需终止")
	}

	// 执行事务
	err = db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		// 更新计划批次状态为终止
		planBatch.Status = int64(model.PlanStatusTerminated) // 终止
		planBatch.TerminatedReason = sql.NullString{String: in.Reason, Valid: in.Reason != ""}
		planBatch.PausedTime = sql.NullTime{}
		planBatch.PausedReason = sql.NullString{}
		planBatch.FinishedTime = sql.NullTime{Time: now, Valid: true}
		planBatch.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(l.ctx, in.CurrentUser), Valid: tool.GetCurrentUserId(l.ctx, in.CurrentUser) != ""}

		// 更新计划批次
		return tx.Save(&planBatch).Error
	})

	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "终止批次事务失败")
	}
	bScope := planscope.BatchScope(&plan, &planBatch)
	bLog := bScope.Logger(l.ctx)
	batchNotifyReq := streamevent.NotifyPlanEventReq{
		EventType:  streamevent.PlanEventType_BATCH_FINISHED,
		PlanId:     planBatch.PlanId,
		PlanType:   plan.Type.String,
		BatchId:    planBatch.BatchId,
		Attributes: map[string]string{},
	}
	bLog.WithFields(logx.Field("notify_event", planscope.NotifyEventBatchFinished)).Info("下游通知：调用 NotifyPlanEvent（批次收尾）")
	l.svcCtx.StreamEventCli.NotifyPlanEvent(l.ctx, &batchNotifyReq)
	planCount, err := gormmodel.UpdatePlanFinishedTime(l.ctx, db, planBatch.PlanPk)
	if err != nil {
		bLog.Errorf("更新计划 finished_time（用于收尾判断）失败: %v", err)
	}
	if planCount > 0 {
		planPlanReq := streamevent.NotifyPlanEventReq{
			EventType:  streamevent.PlanEventType_PLAN_FINISHED,
			PlanId:     planBatch.PlanId,
			PlanType:   plan.Type.String,
			Attributes: map[string]string{},
		}
		bLog.WithFields(logx.Field("notify_event", planscope.NotifyEventPlanFinished)).Info("下游通知：调用 NotifyPlanEvent（计划收尾）")
		l.svcCtx.StreamEventCli.NotifyPlanEvent(l.ctx, &planPlanReq)
	}

	bLog.Info("RPC 终止批次：批次状态已更新，事务已提交")
	return &trigger.TerminatePlanBatchRes{}, nil
}
