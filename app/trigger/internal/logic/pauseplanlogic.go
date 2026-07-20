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
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "参数错误")
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
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询计划失败")
	}
	if plan.Status == model.PlanStatusTerminated || plan.FinishedTime.Valid {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划状态已结束,无需暂停")
	}
	if plan.Status != model.PlanStatusEnabled {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划状态非启用状态,无需暂停")
	}

	// 执行事务
	err = db.Transaction(func(tx *gorm.DB) error {
		// 更新计划状态为暂停
		plan.Status = model.PlanStatusPaused // 暂停
		plan.PausedTime = sql.NullTime{Time: time.Now(), Valid: true}
		plan.PausedReason = sql.NullString{String: in.Reason, Valid: in.Reason != ""}
		plan.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(l.ctx, nil), Valid: tool.GetCurrentUserId(l.ctx, nil) != ""}

		// 更新计划
		transErr := tx.Save(&plan).Error
		if transErr != nil {
			return transErr
		}

		// 更新批次
		transErr = tx.Model(&gormmodel.PlanBatch{}).
			Where("plan_id = ?", plan.PlanId).
			Where("status = ?", model.PlanStatusEnabled).
			Where("finished_time IS NULL").
			Updates(map[string]any{
				"status":        model.PlanStatusPaused,
				"paused_time":   sql.NullTime{Time: time.Now(), Valid: true},
				"paused_reason": sql.NullString{String: in.Reason, Valid: in.Reason != ""},
				"update_user":   sql.NullString{String: tool.GetCurrentUserId(l.ctx, nil), Valid: tool.GetCurrentUserId(l.ctx, nil) != ""},
			}).Error
		return transErr
	})

	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "暂停计划事务失败")
	}

	planscope.PlanScope(&plan).Logger(l.ctx).Info("RPC 暂停计划：计划状态已更新，事务已提交")
	return &trigger.PausePlanRes{}, nil
}
