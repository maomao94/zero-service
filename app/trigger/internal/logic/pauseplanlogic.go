package logic

import (
	"context"
	"database/sql"
	"errors"
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
	"gorm.io/gorm"
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
func (l *PausePlanLogic) PausePlan(in *trigger.PausePlanReq) (*trigger.PausePlanRes, error) {
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
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划状态已结束,无需暂停")
	}
	if plan.Status != model.PlanStatusEnabled {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划状态非启用状态,无需暂停")
	}

	// 执行事务
	err = db.Transaction(func(tx *gorm.DB) error {
		// 条件更新计划：期间若已被终止或已结束，本次暂停不生效，避免把终态整行覆盖回暂停。
		result := tx.Model(&gormmodel.Plan{}).
			Where("id = ?", plan.Id).
			Where("status = ?", model.PlanStatusEnabled).
			Where("finished_time IS NULL").
			Updates(map[string]any{
				"status":        model.PlanStatusPaused,
				"paused_time":   sql.NullTime{Time: time.Now(), Valid: true},
				"paused_reason": sql.NullString{String: in.Reason, Valid: in.Reason != ""},
				"update_user":   sql.NullString{String: authctx.GetUserId(l.ctx), Valid: authctx.GetUserId(l.ctx) != ""},
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return model.ErrNoRowsUpdate
		}

		// 更新批次
		return tx.Model(&gormmodel.PlanBatch{}).
			Where("plan_id = ?", plan.PlanId).
			Where("status = ?", model.PlanStatusEnabled).
			Where("finished_time IS NULL").
			Updates(map[string]any{
				"status":        model.PlanStatusPaused,
				"paused_time":   sql.NullTime{Time: time.Now(), Valid: true},
				"paused_reason": sql.NullString{String: in.Reason, Valid: in.Reason != ""},
				"update_user":   sql.NullString{String: authctx.GetUserId(l.ctx), Valid: authctx.GetUserId(l.ctx) != ""},
			}).Error
	})

	if err != nil {
		if errors.Is(err, model.ErrNoRowsUpdate) {
			return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划状态已变化，请刷新后重试")
		}
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "暂停计划事务失败")
	}
	plan.Status = model.PlanStatusPaused

	planscope.PlanScope(&plan).Logger(l.ctx).Info("RPC 暂停计划：计划状态已更新，事务已提交")
	return &trigger.PausePlanRes{}, nil
}
