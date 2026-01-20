package logic

import (
	"context"
	"database/sql"
	"time"
	"zero-service/common/tool"
	"zero-service/model"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/songzhibin97/gkit/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ResumePlanLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResumePlanLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResumePlanLogic {
	return &ResumePlanLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 恢复计划
func (l *ResumePlanLogic) ResumePlan(in *trigger.ResumePlanReq) (*trigger.ResumePlanRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 查询计划
	plan, err := l.svcCtx.PlanModel.FindOneByPlanId(l.ctx, in.PlanId)
	if err != nil {
		if err == sqlx.ErrNotFound {
			return &trigger.ResumePlanRes{}, nil
		}
		return nil, err
	}

	// 只有暂停状态的计划才能恢复，已终止的计划无法恢复
	if plan.Status != int64(model.PlanStatusPaused) {
		return nil, errors.BadRequest("", "计划非暂停,不可恢复")
	}

	// 执行事务
	err = l.svcCtx.PlanModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		// 更新计划状态为启用
		plan.Status = int64(model.PlanStatusEnabled)
		plan.PausedTime = sql.NullTime{}
		plan.PausedReason = sql.NullString{}
		plan.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(in.CurrentUser), Valid: tool.GetCurrentUserId(in.CurrentUser) != ""}
		plan.UpdateTime = time.Now()

		// 更新计划
		transErr := l.svcCtx.PlanModel.UpdateWithVersion(ctx, tx, plan)
		if transErr != nil {
			return transErr
		}

		builder := l.svcCtx.PlanBatchModel.UpdateBuilder().
			Set("status", int64(model.PlanStatusEnabled)).
			Set("paused_time", sql.NullTime{}).
			Set("paused_reason", sql.NullString{}).
			Set("update_user", sql.NullString{String: tool.GetCurrentUserId(in.CurrentUser), Valid: tool.GetCurrentUserId(in.CurrentUser) != ""}).
			Where("plan_id = ?", in.PlanId).
			Where("status = ?", int64(model.PlanStatusPaused))
		_, transErr = l.svcCtx.PlanBatchModel.UpdateWithBuilder(ctx, tx, builder)
		if transErr != nil {
			return transErr
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &trigger.ResumePlanRes{}, nil
}
