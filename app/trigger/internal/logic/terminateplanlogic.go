package logic

import (
	"context"
	"database/sql"
	"time"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/tool"
	"zero-service/model"

	"github.com/songzhibin97/gkit/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type TerminatePlanLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTerminatePlanLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TerminatePlanLogic {
	return &TerminatePlanLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 终止计划
func (l *TerminatePlanLogic) TerminatePlan(in *trigger.TerminatePlanReq) (*trigger.TerminatePlanRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 查询计划
	plan, err := l.svcCtx.PlanModel.FindOneByPlanId(l.ctx, in.PlanId)
	if err != nil {
		return nil, err
	}

	if plan.Status == int64(model.PlanStatusTerminated) || plan.FinishedTime.Valid {
		return nil, errors.BadRequest("", "计划状态已结束,无需终止")
	}

	// 执行事务
	err = l.svcCtx.PlanModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		now := time.Now()
		// 更新计划状态为已终止
		plan.Status = int64(model.PlanStatusTerminated) // 终止
		plan.FinishedTime = sql.NullTime{Time: now, Valid: true}
		plan.TerminatedReason = sql.NullString{String: in.Reason, Valid: in.Reason != ""}
		plan.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(in.CurrentUser), Valid: tool.GetCurrentUserId(in.CurrentUser) != ""}

		// 更新计划
		transErr := l.svcCtx.PlanModel.UpdateWithVersion(ctx, tx, plan)
		if transErr != nil {
			return transErr
		}

		builder := l.svcCtx.PlanBatchModel.UpdateBuilder().
			Set("status", int64(model.PlanStatusTerminated)).
			Set("terminated_reason", sql.NullString{String: in.Reason, Valid: in.Reason != ""}).
			Set("finished_time", now).
			Set("update_user", sql.NullString{String: tool.GetCurrentUserId(in.CurrentUser), Valid: tool.GetCurrentUserId(in.CurrentUser) != ""}).
			Where("plan_id = ?", in.PlanId).
			Where("status != ?", int64(model.PlanStatusTerminated)).
			Where("finished_time IS NULL")
		_, transErr = l.svcCtx.PlanBatchModel.UpdateWithBuilder(ctx, tx, builder)
		if transErr != nil {
			return transErr
		}
		return nil
	})

	if err != nil {
		return &trigger.TerminatePlanRes{}, nil
	}

	return &trigger.TerminatePlanRes{}, nil
}
