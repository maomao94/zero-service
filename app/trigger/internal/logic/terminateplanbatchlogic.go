package logic

import (
	"context"
	"database/sql"
	"time"
	"zero-service/facade/streamevent/streamevent"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/tool"
	"zero-service/model"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/songzhibin97/gkit/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
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
	if in.Id <= 0 && strutil.IsBlank(in.BatchId) {
		return nil, errors.BadRequest("", "参数错误")
	}

	// 查询计划批次
	var planBatch *model.PlanBatch
	if in.Id > 0 {
		planBatch, err = l.svcCtx.PlanBatchModel.FindOne(l.ctx, in.Id)
	} else {
		planBatch, err = l.svcCtx.PlanBatchModel.FindOneByBatchId(l.ctx, in.BatchId)
	}
	if err != nil {
		return nil, err
	}

	// 查询计划
	plan, err := l.svcCtx.PlanModel.FindOneByPlanId(l.ctx, planBatch.PlanId)
	if err != nil {
		return nil, err
	}

	// 检查当前状态是否允许终止操作
	if plan.Status == int64(model.PlanStatusTerminated) || plan.FinishedTime.Valid {
		return nil, errors.BadRequest("", "计划状态已结束,无需终止")
	}

	if planBatch.Status == int64(model.PlanStatusTerminated) || planBatch.FinishedTime.Valid {
		return nil, errors.BadRequest("", "计划批次状态已结束,无需终止")
	}

	// 执行事务
	err = l.svcCtx.PlanBatchModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		now := time.Now()
		// 更新计划批次状态为终止
		planBatch.Status = int64(model.PlanStatusTerminated) // 终止
		planBatch.PausedTime = sql.NullTime{Time: time.Now(), Valid: true}
		planBatch.PausedReason = sql.NullString{String: in.Reason, Valid: in.Reason != ""}
		planBatch.FinishedTime = sql.NullTime{Time: now, Valid: true}
		planBatch.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(in.CurrentUser), Valid: tool.GetCurrentUserId(in.CurrentUser) != ""}

		// 更新计划批次
		transErr := l.svcCtx.PlanBatchModel.UpdateWithVersion(ctx, tx, planBatch)
		if transErr != nil {
			return transErr
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	planCount, err := l.svcCtx.PlanModel.UpdateBatchFinishedTime(l.ctx, planBatch.PlanPk)
	if err != nil {
		l.Errorf("Error updating plan %s completed time: %v", planBatch.PlanId, err)
	}
	if planCount > 0 {
		planPlanReq := streamevent.NotifyPlanEventReq{
			EventType: streamevent.PlanEventType_PLAN_FINISHED,
			PlanId:    planBatch.PlanId,
			PlanType:  plan.Type.String,
			//BatchId:    planBatch.BatchId,
			Attributes: map[string]string{},
		}
		l.svcCtx.StreamEventCli.NotifyPlanEvent(l.ctx, &planPlanReq)
	}
	return &trigger.TerminatePlanBatchRes{}, nil
}
