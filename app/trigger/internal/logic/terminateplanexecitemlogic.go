package logic

import (
	"context"
	"database/sql"
	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"
	"zero-service/model"
	"zero-service/third_party/extproto"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type TerminatePlanExecItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTerminatePlanExecItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TerminatePlanExecItemLogic {
	return &TerminatePlanExecItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 终止执行项
func (l *TerminatePlanExecItemLogic) TerminatePlanExecItem(in *trigger.TerminatePlanExecItemReq) (*trigger.TerminatePlanExecItemRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 检查参数
	if strutil.IsBlank(in.Id) && strutil.IsBlank(in.ExecId) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "参数错误")
	}

	// 查询执行项
	db := l.svcCtx.DB.WithContext(l.ctx).DB
	var execItem gormmodel.PlanExecItem
	if !strutil.IsBlank(in.Id) {
		err = db.Where("id = ?", in.Id).First(&execItem).Error
	} else {
		err = db.Where("exec_id = ?", in.ExecId).First(&execItem).Error
	}
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询执行项失败")
	}

	// 查询计划批次
	var planBatch gormmodel.PlanBatch
	if err := db.Where("id = ?", execItem.BatchPk).First(&planBatch).Error; err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询计划批次失败")
	}
	// 查询计划
	var plan gormmodel.Plan
	if err := db.Where("plan_id = ?", execItem.PlanId).First(&plan).Error; err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询计划失败")
	}
	if plan.Status == int64(model.PlanStatusTerminated) || plan.FinishedTime.Valid {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划状态已结束,无需终止")
	}

	if planBatch.Status == int64(model.PlanStatusTerminated) || planBatch.FinishedTime.Valid {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划批次状态已结束,无需终止")
	}

	if execItem.Status == int64(model.StatusCompleted) || execItem.Status == int64(model.StatusTerminated) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "执行项状态已结束,无需终止")
	}

	scope := planscope.ExecScope(&execItem)
	log := scope.Logger(l.ctx)

	// 执行事务
	err = db.Transaction(func(tx *gorm.DB) error {
		// 更新执行项状态为已终止
		execItem.Status = int64(model.StatusTerminated)
		execItem.TerminatedReason = sql.NullString{String: in.Reason, Valid: in.Reason != ""}
		execItem.PausedTime = sql.NullTime{}
		execItem.PausedReason = sql.NullString{}
		execItem.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(l.ctx, nil), Valid: tool.GetCurrentUserId(l.ctx, nil) != ""}

		// 更新执行项
		return tx.Save(&execItem).Error
	})

	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "终止执行项事务失败")
	}
	batchCount, err := gormmodel.UpdatePlanBatchFinishedTime(l.ctx, db, execItem.BatchPk)
	if err != nil {
		log.Errorf("更新批次 finished_time（用于收尾判断）失败: %v", err)
	}
	if batchCount > 0 {
		batchNotifyReq := streamevent.NotifyPlanEventReq{
			EventType:  streamevent.PlanEventType_BATCH_FINISHED,
			PlanId:     execItem.PlanId,
			PlanType:   plan.Type.String,
			BatchId:    execItem.BatchId,
			Attributes: map[string]string{},
		}
		log.WithFields(logx.Field("notify_event", planscope.NotifyEventBatchFinished)).Info("下游通知：调用 NotifyPlanEvent（批次收尾）")
		l.svcCtx.StreamEventCli.NotifyPlanEvent(l.ctx, &batchNotifyReq)
	}

	planCount, err := gormmodel.UpdatePlanFinishedTime(l.ctx, db, execItem.PlanPk)
	if err != nil {
		log.Errorf("更新计划 finished_time（用于收尾判断）失败: %v", err)
	}
	if planCount > 0 {
		planPlanReq := streamevent.NotifyPlanEventReq{
			EventType:  streamevent.PlanEventType_PLAN_FINISHED,
			PlanId:     execItem.PlanId,
			PlanType:   plan.Type.String,
			Attributes: map[string]string{},
		}
		log.WithFields(logx.Field("notify_event", planscope.NotifyEventPlanFinished)).Info("下游通知：调用 NotifyPlanEvent（计划收尾）")
		l.svcCtx.StreamEventCli.NotifyPlanEvent(l.ctx, &planPlanReq)
	}

	log.Info("RPC 终止执行项：执行项状态已更新，事务已提交")
	return &trigger.TerminatePlanExecItemRes{}, nil
}
