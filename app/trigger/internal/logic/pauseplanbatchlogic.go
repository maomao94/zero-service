package logic

import (
	"context"
	"database/sql"
	"time"

	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/authctx"
	"zero-service/common/errx"
	"zero-service/model"
	"zero-service/third_party/extproto"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
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

	if plan.Status == model.PlanStatusTerminated || plan.FinishedTime.Valid {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划状态已结束,无需暂停")
	}
	if planBatch.Status == model.PlanStatusTerminated || planBatch.FinishedTime.Valid {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划批次状态已结束,无需暂停")
	}
	if planBatch.Status != model.PlanStatusEnabled {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划批次状态非启用状态,不可暂停")
	}
	if plan.Status != model.PlanStatusEnabled {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划状态非启用状态,无需暂停")
	}

	// 条件更新批次：期间若已被终止或已结束，本次暂停不生效，避免把终态整行覆盖回暂停。
	result := db.Model(&gormmodel.PlanBatch{}).
		Where("id = ?", planBatch.Id).
		Where("status = ?", model.PlanStatusEnabled).
		Where("finished_time IS NULL").
		Updates(map[string]any{
			"status":        model.PlanStatusPaused,
			"paused_time":   sql.NullTime{Time: time.Now(), Valid: true},
			"paused_reason": sql.NullString{String: in.Reason, Valid: in.Reason != ""},
			"update_user":   sql.NullString{String: authctx.GetUserId(l.ctx), Valid: authctx.GetUserId(l.ctx) != ""},
		})
	if result.Error != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, result.Error, "暂停批次失败")
	}
	if result.RowsAffected == 0 {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划批次状态已变化，请刷新后重试")
	}
	planBatch.Status = model.PlanStatusPaused

	planscope.BatchScope(&plan, &planBatch).Logger(l.ctx).Info("RPC 暂停批次：批次状态已更新")
	return &trigger.PausePlanBatchRes{}, nil
}
