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

type PausePlanBatchLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPausePlanBatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PausePlanBatchLogic {
	return &PausePlanBatchLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 暂停计划批次
func (l *PausePlanBatchLogic) PausePlanBatch(in *trigger.PausePlanBatchReq) (*trigger.PausePlanBatchRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 查询计划批次
	planBatch, err := l.svcCtx.PlanBatchModel.FindOne(l.ctx, in.Id)
	if err != nil {
		return nil, err
	}
	// 查询计划
	plan, err := l.svcCtx.PlanModel.FindOneByPlanId(l.ctx, planBatch.PlanId)
	if err != nil {
		return nil, err
	}

	if plan.Status == int64(model.PlanStatusTerminated) || plan.FinishedTime.Valid {
		return nil, errors.BadRequest("", "计划状态已结束,无需暂停")
	}
	if planBatch.Status == int64(model.PlanStatusTerminated) || plan.FinishedTime.Valid {
		return nil, errors.BadRequest("", "计划批次状态已结束,无需暂停")
	}
	if planBatch.Status != int64(model.PlanStatusEnabled) {
		return nil, errors.BadRequest("", "计划批次状态非启用状态,不可暂停")
	}
	if plan.Status != int64(model.PlanStatusEnabled) {
		return nil, errors.BadRequest("", "计划状态非启用状态,无需暂停")
	}

	// 执行事务
	err = l.svcCtx.PlanBatchModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		// 更新计划批次状态为暂停
		planBatch.Status = int64(model.PlanStatusPaused) // 暂停
		planBatch.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(in.CurrentUser), Valid: tool.GetCurrentUserId(in.CurrentUser) != ""}

		// 更新计划批次
		transErr := l.svcCtx.PlanBatchModel.UpdateWithVersion(ctx, tx, planBatch)
		if transErr != nil {
			return transErr
		}
		return nil
	})

	if err != nil {
		return &trigger.PausePlanBatchRes{}, nil
	}

	return &trigger.PausePlanBatchRes{}, nil
}
