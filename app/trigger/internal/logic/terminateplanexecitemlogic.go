package logic

import (
	"context"
	"database/sql"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"
	"zero-service/model"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/songzhibin97/gkit/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
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

	// 查询计划批次
	planBatch, err := l.svcCtx.PlanBatchModel.FindOne(l.ctx, execItem.BatchPk)
	if err != nil {
		return nil, err
	}
	// 查询计划
	plan, err := l.svcCtx.PlanModel.FindOneByPlanId(l.ctx, execItem.PlanId)
	if err != nil {
		return nil, err
	}
	if plan.Status == int64(model.PlanStatusTerminated) || plan.FinishedTime.Valid {
		return nil, errors.BadRequest("", "计划状态已结束,无需终止")
	}

	if planBatch.Status == int64(model.PlanStatusTerminated) || planBatch.FinishedTime.Valid {
		return nil, errors.BadRequest("", "计划批次状态已结束,无需终止")
	}

	if execItem.Status == int64(model.StatusCompleted) || execItem.Status == int64(model.StatusTerminated) {
		return nil, errors.BadRequest("", "执行项状态已结束,无需终止")
	}

	// 执行事务
	err = l.svcCtx.PlanModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		// 更新执行项状态为已终止
		execItem.Status = int64(model.StatusTerminated)
		execItem.TerminatedReason = sql.NullString{String: in.Reason, Valid: in.Reason != ""}
		execItem.PausedTime = sql.NullTime{}
		execItem.PausedReason = sql.NullString{}
		execItem.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(in.CurrentUser), Valid: tool.GetCurrentUserId(in.CurrentUser) != ""}

		// 更新执行项
		transErr := l.svcCtx.PlanExecItemModel.UpdateWithVersion(ctx, tx, execItem)
		if transErr != nil {
			return transErr
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	batchCount, err := l.svcCtx.PlanBatchModel.UpdateBatchFinishedTime(l.ctx, execItem.BatchPk)
	if err != nil {
		l.Errorf("Error updating batch %s completed time: %v", execItem.BatchId, err)
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
		l.Errorf("Error updating plan %s completed time: %v", execItem.PlanId, err)
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

	return &trigger.TerminatePlanExecItemRes{}, nil
}
