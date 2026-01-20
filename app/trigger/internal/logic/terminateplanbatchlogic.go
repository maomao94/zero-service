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
	l.svcCtx.PlanModel.UpdateBatchFinishedTime(l.ctx, planBatch.PlanPk)
	return &trigger.TerminatePlanBatchRes{}, nil
}
