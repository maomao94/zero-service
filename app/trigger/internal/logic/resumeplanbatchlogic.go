package logic

import (
	"context"
	"database/sql"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/tool"
	"zero-service/model"

	"github.com/songzhibin97/gkit/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ResumePlanBatchLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResumePlanBatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResumePlanBatchLogic {
	return &ResumePlanBatchLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 恢复计划批次
func (l *ResumePlanBatchLogic) ResumePlanBatch(in *trigger.ResumePlanBatchReq) (*trigger.ResumePlanBatchRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 查询计划批次
	planBatch, err := l.svcCtx.PlanBatchModel.FindOne(l.ctx, in.Id)
	if err != nil {
		if err == sqlx.ErrNotFound {
			return &trigger.ResumePlanBatchRes{}, nil
		}
		return nil, err
	}

	// 查询计划
	plan, err := l.svcCtx.PlanModel.FindOneByPlanId(l.ctx, planBatch.PlanId)
	if err != nil {
		return nil, err
	}

	if plan.Status == int64(model.PlanStatusTerminated) || plan.FinishedTime.Valid {
		return nil, errors.BadRequest("", "计划状态已结束,无需恢复")
	}
	if planBatch.Status == int64(model.PlanStatusTerminated) || planBatch.FinishedTime.Valid {
		return nil, errors.BadRequest("", "计划批次状态已结束,无需恢复")
	}
	if planBatch.Status != int64(model.PlanStatusPaused) {
		return nil, errors.BadRequest("", "计划批次非暂停,不可恢复")
	}

	// 执行事务
	err = l.svcCtx.PlanBatchModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		// 更新计划批次状态为启用
		planBatch.Status = int64(model.PlanStatusEnabled) // 启用
		planBatch.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(in.CurrentUser), Valid: tool.GetCurrentUserId(in.CurrentUser) != ""}

		// 更新计划批次
		transErr := l.svcCtx.PlanBatchModel.UpdateWithVersion(ctx, tx, planBatch)
		if transErr != nil {
			return transErr
		}

		// 更新计划启用
		builder := l.svcCtx.PlanModel.UpdateBuilder().
			Set("status", int64(model.PlanStatusEnabled)).
			Set("update_user", sql.NullString{String: tool.GetCurrentUserId(in.CurrentUser), Valid: tool.GetCurrentUserId(in.CurrentUser) != ""}).
			Where("id = ?", planBatch.PlanPk).
			Where("status = ?", int64(model.PlanStatusPaused))
		_, transErr = l.svcCtx.PlanModel.UpdateWithBuilder(ctx, tx, builder)
		if transErr != nil {
			return transErr
		}
		return nil
	})

	if err != nil {
		return &trigger.ResumePlanBatchRes{}, nil
	}

	return &trigger.ResumePlanBatchRes{}, nil
}
