package logic

import (
	"context"
	"database/sql"
	"errors"
	"time"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/common/tool"
	"zero-service/model"
	"zero-service/third_party/extproto"

	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type ResumePlanLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResumePlanLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResumePlanLogic {
	return &ResumePlanLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 恢复计划
func (l *ResumePlanLogic) ResumePlan(in *trigger.ResumePlanReq) (*trigger.ResumePlanRes, error) {
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &trigger.ResumePlanRes{}, nil
		}
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_DB, "查询计划失败")
	}

	// 只有暂停状态的计划才能恢复，已终止的计划无法恢复
	if plan.Status != model.PlanStatusPaused {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划非暂停,不可恢复")
	}

	// 执行事务
	err = db.Transaction(func(tx *gorm.DB) error {
		// 更新计划状态为启用
		plan.Status = model.PlanStatusEnabled
		plan.PausedTime = sql.NullTime{}
		plan.PausedReason = sql.NullString{}
		plan.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(l.ctx, nil), Valid: tool.GetCurrentUserId(l.ctx, nil) != ""}
		plan.UpdateTime = time.Now()

		// 更新计划
		transErr := tx.Save(&plan).Error
		if transErr != nil {
			return transErr
		}

		// 更新批次
		transErr = tx.Model(&gormmodel.PlanBatch{}).
			Where("plan_id = ?", plan.PlanId).
			Where("status = ?", model.PlanStatusPaused).
			Updates(map[string]any{
				"status":        model.PlanStatusEnabled,
				"paused_time":   sql.NullTime{},
				"paused_reason": sql.NullString{},
				"update_user":   sql.NullString{String: tool.GetCurrentUserId(l.ctx, nil), Valid: tool.GetCurrentUserId(l.ctx, nil) != ""},
			}).Error
		return transErr
	})

	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "恢复计划事务失败")
	}

	planscope.PlanScope(&plan).Logger(l.ctx).Info("RPC 恢复计划：计划状态已更新，事务已提交")
	return &trigger.ResumePlanRes{}, nil
}
