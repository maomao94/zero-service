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

	// 执行事务
	err = db.Transaction(func(tx *gorm.DB) error {
		var runningCount int64
		runningCount, transErr := gormmodel.CountRunningExecItemsByBatch(l.ctx, tx, planBatch.Id)
		if transErr != nil {
			return transErr
		}
		if runningCount > 0 {
			return errPlanExecItemRunning
		}

		now := time.Now()
		// 只更新父级状态；cron claim 后会重新加载父级并在非 enabled/finished 时停止下发。
		updated, transErr := gormmodel.UpdatePlanBatchTerminated(
			l.ctx, tx, planBatch.Id, in.Reason, authctx.GetUserId(l.ctx), now,
		)
		if transErr != nil {
			return transErr
		}
		if updated == 0 {
			return model.ErrNoRowsUpdate
		}
		closeReason := in.Reason
		if closeReason == "" {
			closeReason = "计划批次被终止"
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
		if errors.Is(err, errPlanExecItemRunning) {
			return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划批次存在正在执行的执行项，请执行项结束后再终止")
		}
		if errors.Is(err, model.ErrNoRowsUpdate) {
			return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划批次状态已变化，无法终止")
		}
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "终止批次事务失败")
	}
	bScope := planscope.BatchScope(&plan, &planBatch)
	bLog := bScope.Logger(l.ctx)
	if _, notifyErr := confirm.NotifyBatchFinished(l.ctx, db, l.svcCtx.StreamEventCli, &plan, &planBatch, in.Reason); notifyErr != nil {
		bLog.Errorf("更新计划 finished_time 失败: %v", notifyErr)
	}
	bLog.Info("RPC 终止批次：批次状态已更新，事务已提交")
	return &trigger.TerminatePlanBatchRes{}, nil
}
