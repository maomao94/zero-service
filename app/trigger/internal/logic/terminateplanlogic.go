package logic

import (
	"context"
	"database/sql"
	"time"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

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
		if err == sqlx.ErrNotFound {
			return &trigger.TerminatePlanRes{}, nil
		}
		return nil, err
	}

	// 执行事务
	err = l.svcCtx.PlanModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		// 更新计划状态为已终止
		plan.Status = 2 // 2-已终止
		plan.IsTerminated = 1
		plan.TerminatedTime = sql.NullTime{Time: time.Now(), Valid: true}
		plan.TerminatedReason = in.Reason

		// 更新计划
		_, transErr := l.svcCtx.PlanModel.Update(ctx, tx, plan)
		if transErr != nil {
			return transErr
		}

		// 更新计划下所有执行项状态为已终止
		builder := l.svcCtx.PlanExecItemModel.UpdateBuilder().
			Set("status", 5).
			Set("is_terminated", 1).
			Set("terminated_time", time.Now()).
			Set("terminated_reason", in.Reason).
			Where("plan_id = ?", in.PlanId)

		_, transErr = l.svcCtx.PlanExecItemModel.UpdateWithBuilder(ctx, tx, builder)
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
