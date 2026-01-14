package logic

import (
	"context"
	"database/sql"
	"time"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/tool"
	"zero-service/model"

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
func (l *PausePlanLogic) PausePlan(in *trigger.PausePlanReq) (*trigger.PlanOperateRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 查询计划
	plan, err := l.svcCtx.PlanModel.FindOneByPlanId(l.ctx, in.PlanId)
	if err != nil {
		if err == sqlx.ErrNotFound {
			return &trigger.PlanOperateRes{}, nil
		}
		return nil, err
	}

	if plan.Status == int64(model.PlanStatusDisabled) || plan.Status == int64(model.PlanStatusTerminated) || plan.Status == int64(model.PlanStatusPaused) {
		return &trigger.PlanOperateRes{}, nil
	}

	// 执行事务
	err = l.svcCtx.PlanModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		// 更新计划状态为暂停
		plan.Status = int64(model.PlanStatusPaused) // 暂停
		plan.PausedTime = sql.NullTime{Time: time.Now(), Valid: true}
		plan.PausedReason = sql.NullString{String: in.Reason, Valid: in.Reason != ""}
		plan.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(in.CurrentUser), Valid: tool.GetCurrentUserId(in.CurrentUser) != ""}

		// 更新计划
		_, transErr := l.svcCtx.PlanModel.Update(ctx, tx, plan)
		if transErr != nil {
			return transErr
		}

		builder := l.svcCtx.PlanExecItemModel.UpdateBuilder().
			Set("status", int64(model.StatusPaused)).
			Set("paused_time", time.Now()).
			Set("paused_reason", sql.NullString{String: in.Reason, Valid: in.Reason != ""}).
			Set("update_user", sql.NullString{String: tool.GetCurrentUserId(in.CurrentUser), Valid: tool.GetCurrentUserId(in.CurrentUser) != ""}).
			Where("plan_id = ?", in.PlanId).
			Where("status != ?", int64(model.StatusCompleted)).
			Where("status != ?", int64(model.StatusTerminated))
		_, transErr = l.svcCtx.PlanExecItemModel.UpdateWithBuilder(ctx, tx, builder)
		if transErr != nil {
			return transErr
		}
		return nil
	})

	if err != nil {
		return &trigger.PlanOperateRes{}, nil
	}

	return &trigger.PlanOperateRes{}, nil
}
