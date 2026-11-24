package logic

import (
	"context"
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

	// 检查参数
	if strutil.IsBlank(in.Id) && strutil.IsBlank(in.BatchId) {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_PARAM, "参数错误")
	}

	// 查询计划批次
	db := l.svcCtx.DB.WithContext(l.ctx).DB
	var planBatch gormmodel.PlanBatch
	if !strutil.IsBlank(in.Id) {
		err = db.Where("id = ?", in.Id).First(&planBatch).Error
	} else {
		err = db.Where("batch_id = ?", in.BatchId).First(&planBatch).Error
	}
	if err != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询计划批次失败")
	}

	// 查询计划
	var plan gormmodel.Plan
	if err := db.Where("plan_id = ?", planBatch.PlanId).First(&plan).Error; err != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询计划失败")
	}

	// 检查当前状态是否允许终止操作
	if plan.Status == model.PlanStatusTerminated || plan.FinishedTime.Valid {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划状态已结束,无需终止")
	}

	if planBatch.Status == model.PlanStatusTerminated || planBatch.FinishedTime.Valid {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划批次状态已结束,无需终止")
	}

	closeReason := in.Reason
	if closeReason == "" {
		closeReason = "计划批次被终止"
	}
	now := time.Now()
	var terminatedItems int64

	// 执行事务
	err = db.Transaction(func(tx *gorm.DB) error {
		// 不拦截执行中的执行项：批次终止时把未结束执行项统一收敛为终止态，
		// 避免父级结束后遗留等待中/执行中的执行项。在途下游任务由业务侧按通知自行闭环。
		var transErr error
		terminatedItems, transErr = gormmodel.TerminateUnfinishedExecItemsByBatch(
			l.ctx, tx, planBatch.Id, closeReason, authctx.GetUserId(l.ctx), now,
		)
		if transErr != nil {
			return transErr
		}

		// 更新父级状态；cron claim 后会重新加载父级并在非 enabled/finished 时停止下发。
		updated, transErr := gormmodel.UpdatePlanBatchTerminated(
			l.ctx, tx, planBatch.Id, closeReason, authctx.GetUserId(l.ctx), now,
		)
		if transErr != nil {
			return transErr
		}
		if updated == 0 {
			return model.ErrNoRowsUpdate
		}
		if _, transErr := gormmodel.ClosePlanConfirmsByBatch(l.ctx, tx, planBatch.Id, authctx.GetUserId(l.ctx), closeReason, now); transErr != nil {
			return transErr
		}
		planBatch.Status = model.PlanStatusTerminated
		planBatch.FinishedTime.Time = now
		planBatch.FinishedTime.Valid = true
		return nil
	})

	if err != nil {
		if errors.Is(err, model.ErrNoRowsUpdate) {
			return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划批次状态已变化，无法终止")
		}
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "终止批次事务失败")
	}
	bScope := planscope.BatchScope(&plan, &planBatch)
	bLog := bScope.Logger(l.ctx)
	if _, notifyErr := confirm.NotifyBatchFinished(l.ctx, db, l.svcCtx.StreamEventCli, &plan, &planBatch, closeReason); notifyErr != nil {
		bLog.Errorf("更新计划 finished_time 失败: %v", notifyErr)
	}
	bLog.WithFields(logx.Field("terminated_items", terminatedItems)).Info("RPC 终止批次：批次与未结束执行项已更新，事务已提交")
	return &trigger.TerminatePlanBatchRes{}, nil
}
