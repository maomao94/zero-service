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

	// 执行事务
	err = l.svcCtx.PlanModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		// 更新计划状态为暂停
		plan.Status = 3 // 3-暂停
		plan.IsPaused = 1
		plan.PausedTime = sql.NullTime{Time: time.Now(), Valid: true}
		plan.PausedReason = in.Reason

		// 更新计划
		_, transErr := l.svcCtx.PlanModel.Update(ctx, tx, plan)
		if transErr != nil {
			return transErr
		}

		// 更新计划下所有执行项状态为暂停
		builder := l.svcCtx.PlanExecItemModel.UpdateBuilder().
			Set("status", 6).
			Set("is_paused", 1).
			Set("paused_time", time.Now()).
			Set("paused_reason", in.Reason).
			Where("plan_id = ?", in.PlanId)

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
