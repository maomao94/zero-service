package logic

import (
	"context"
	"database/sql"
	"errors"
	"time"
	"zero-service/common/errx"

	"zero-service/app/trigger/internal/confirm"
	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/authctx"
	"zero-service/model"
	"zero-service/third_party/extproto"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
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

	// 检查参数
	if strutil.IsBlank(in.Id) && strutil.IsBlank(in.PlanId) {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_PARAM, "参数错误")
	}

	// 查询计划
	db := l.svcCtx.DB.WithContext(l.ctx).DB
	var plan gormmodel.Plan
	if !strutil.IsBlank(in.Id) {
		err = db.Where("id = ?", in.Id).First(&plan).Error
	} else {
		err = db.Where("plan_id = ?", in.PlanId).First(&plan).Error
	}
	if err != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询计划失败")
	}

	if plan.Status == model.PlanStatusTerminated || plan.FinishedTime.Valid {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划状态已结束,无需终止")
	}

	closeReason := in.Reason
	if closeReason == "" {
		closeReason = "计划被终止"
	}
	now := time.Now()
	var terminatedItems int64

	// 执行事务
	err = db.Transaction(func(tx *gorm.DB) error {
		// 不拦截执行中的执行项：计划终止时把未结束执行项统一收敛为终止态，
		// 与批次终止保持一致。在途下游任务由业务侧按通知自行闭环。
		var transErr error
		terminatedItems, transErr = gormmodel.TerminateUnfinishedExecItemsByPlan(
			l.ctx, tx, plan.Id, closeReason, authctx.GetUserId(l.ctx), now,
		)
		if transErr != nil {
			return transErr
		}

		// 只更新父级状态；cron claim 后会重新加载父级并在非 enabled/finished 时停止下发。
		updated, transErr := gormmodel.UpdatePlanTerminated(
			l.ctx, tx, plan.Id, closeReason, authctx.GetUserId(l.ctx), now,
		)
		if transErr != nil {
			return transErr
		}
		if updated == 0 {
			return model.ErrNoRowsUpdate
		}
		plan.Status = model.PlanStatusTerminated
		plan.FinishedTime.Time = now
		plan.FinishedTime.Valid = true

		// 更新批次
		transErr = tx.Model(&gormmodel.PlanBatch{}).
			Where("plan_id = ?", plan.PlanId).
			Where("status != ?", model.PlanStatusTerminated).
			Where("finished_time IS NULL").
			Updates(map[string]any{
				"status":            model.PlanStatusTerminated,
				"terminated_reason": sql.NullString{String: closeReason, Valid: true},
				"finished_time":     now,
				"update_user":       sql.NullString{String: authctx.GetUserId(l.ctx), Valid: authctx.GetUserId(l.ctx) != ""},
			}).Error
		if transErr != nil {
			return transErr
		}
		if _, transErr := gormmodel.ClosePlanConfirmsByPlan(l.ctx, tx, plan.Id, authctx.GetUserId(l.ctx), closeReason, now); transErr != nil {
			return transErr
		}
		return nil
	})

	if err != nil {
		if errors.Is(err, model.ErrNoRowsUpdate) {
			return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划状态已变化，无法终止")
		}
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "终止计划事务失败")
	}
	confirm.NotifyPlanFinished(l.ctx, l.svcCtx.StreamEventCli, &plan, closeReason)

	planLog := planscope.PlanScope(&plan).Logger(l.ctx)
	planLog.WithFields(logx.Field("terminated_items", terminatedItems)).Info("RPC 终止计划：计划与未结束执行项已更新，事务已提交")
	return &trigger.TerminatePlanRes{}, nil
}
