package logic

import (
	"context"
	"database/sql"
	"time"

	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/tool"
	"zero-service/model"
	"zero-service/third_party/extproto"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
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

	// 检查参数
	if strutil.IsBlank(in.Id) && strutil.IsBlank(in.BatchId) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "参数错误")
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
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询计划批次失败")
	}
	// 查询计划
	var plan gormmodel.Plan
	if err := db.Where("plan_id = ?", planBatch.PlanId).First(&plan).Error; err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询计划失败")
	}

	if plan.Status == int64(model.PlanStatusTerminated) || plan.FinishedTime.Valid {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划状态已结束,无需暂停")
	}
	if planBatch.Status == int64(model.PlanStatusTerminated) || planBatch.FinishedTime.Valid {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划批次状态已结束,无需暂停")
	}
	if planBatch.Status != int64(model.PlanStatusEnabled) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划批次状态非启用状态,不可暂停")
	}
	if plan.Status != int64(model.PlanStatusEnabled) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划状态非启用状态,无需暂停")
	}

	// 执行事务
	err = db.Transaction(func(tx *gorm.DB) error {
		// 更新计划批次状态为暂停
		planBatch.Status = int64(model.PlanStatusPaused) // 暂停
		planBatch.PausedTime = sql.NullTime{Time: time.Now(), Valid: true}
		planBatch.PausedReason = sql.NullString{String: in.Reason, Valid: in.Reason != ""}
		planBatch.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(l.ctx, nil), Valid: tool.GetCurrentUserId(l.ctx, nil) != ""}

		// 更新计划批次
		return tx.Save(&planBatch).Error
	})

	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "暂停批次事务失败")
	}

	planscope.BatchScope(&plan, &planBatch).Logger(l.ctx).Info("RPC 暂停批次：批次状态已更新，事务已提交")
	return &trigger.PausePlanBatchRes{}, nil
}
