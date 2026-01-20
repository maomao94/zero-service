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

type PausePlanLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPausePlanLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PausePlanLogic {
	return &PausePlanLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 暂停计划
func (l *PausePlanLogic) PausePlan(in *trigger.PausePlanReq) (*trigger.PausePlanRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 查询计划
	plan, err := l.svcCtx.PlanModel.FindOneByPlanId(l.ctx, in.PlanId)
	if err != nil {
		if err == sqlx.ErrNotFound {
			return &trigger.PausePlanRes{}, nil
		}
		return nil, err
	}
	if plan.Status == int64(model.PlanStatusTerminated) || plan.FinishedTime.Valid {
		return nil, errors.BadRequest("", "计划状态已结束,无需暂停")
	}
	if plan.Status != int64(model.PlanStatusEnabled) {
		return nil, errors.BadRequest("", "计划状态非启用状态,无需暂停")
	}

	// 执行事务
	err = l.svcCtx.PlanModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		// 更新计划状态为暂停
		plan.Status = int64(model.PlanStatusPaused) // 暂停
		plan.PausedTime = sql.NullTime{Time: time.Now(), Valid: true}
		plan.PausedReason = sql.NullString{String: in.Reason, Valid: in.Reason != ""}
		plan.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(in.CurrentUser), Valid: tool.GetCurrentUserId(in.CurrentUser) != ""}

		// 更新计划
		transErr := l.svcCtx.PlanModel.UpdateWithVersion(ctx, tx, plan)
		if transErr != nil {
			return transErr
		}

		builder := l.svcCtx.PlanBatchModel.UpdateBuilder().
			Set("status", int64(model.PlanStatusPaused)).
			Set("paused_time", sql.NullTime{Time: time.Now(), Valid: true}).
			Set("paused_reason", sql.NullString{String: in.Reason, Valid: in.Reason != ""}).
			Set("update_user", sql.NullString{String: tool.GetCurrentUserId(in.CurrentUser), Valid: tool.GetCurrentUserId(in.CurrentUser) != ""}).
			Where("plan_id = ?", in.PlanId).
			Where("status = ?", int64(model.PlanStatusEnabled)).
			Where("finished_time IS NULL")
		_, transErr = l.svcCtx.PlanBatchModel.UpdateWithBuilder(ctx, tx, builder)
		if transErr != nil {
			return transErr
		}
		return nil
		return nil
	})

	if err != nil {
		return &trigger.PausePlanRes{}, nil
	}

	return &trigger.PausePlanRes{}, nil
}
