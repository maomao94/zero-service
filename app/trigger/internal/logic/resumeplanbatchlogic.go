package logic

import (
	"context"
	"database/sql"
	"errors"
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

type ResumePlanBatchLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResumePlanBatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResumePlanBatchLogic {
	return &ResumePlanBatchLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 恢复计划批次
func (l *ResumePlanBatchLogic) ResumePlanBatch(in *trigger.ResumePlanBatchReq) (*trigger.ResumePlanBatchRes, error) {
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &trigger.ResumePlanBatchRes{}, nil
		}
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询计划批次失败")
	}

	// 查询计划
	var plan gormmodel.Plan
	if err := db.Where("plan_id = ?", planBatch.PlanId).First(&plan).Error; err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询计划失败")
	}

	if plan.Status == int64(model.PlanStatusTerminated) || plan.FinishedTime.Valid {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划状态已结束,无需恢复")
	}
	if planBatch.Status == int64(model.PlanStatusTerminated) || planBatch.FinishedTime.Valid {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划批次状态已结束,无需恢复")
	}
	if planBatch.Status != int64(model.PlanStatusPaused) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划批次非暂停,不可恢复")
	}

	// 执行事务
	err = db.Transaction(func(tx *gorm.DB) error {
		// 更新计划批次状态为启用
		planBatch.Status = int64(model.PlanStatusEnabled) // 启用
		planBatch.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(l.ctx, nil), Valid: tool.GetCurrentUserId(l.ctx, nil) != ""}

		// 更新计划批次
		transErr := tx.Save(&planBatch).Error
		if transErr != nil {
			return transErr
		}

		// 更新计划启用
		transErr = tx.Model(&gormmodel.Plan{}).
			Where("id = ?", planBatch.PlanPk).
			Where("status = ?", int64(model.PlanStatusPaused)).
			Updates(map[string]any{
				"status":        int64(model.PlanStatusEnabled),
				"paused_time":   sql.NullTime{},
				"paused_reason": sql.NullString{},
				"update_user":   sql.NullString{String: tool.GetCurrentUserId(l.ctx, nil), Valid: tool.GetCurrentUserId(l.ctx, nil) != ""},
			}).Error
		return transErr
	})

	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "恢复批次事务失败")
	}

	planscope.BatchScope(&plan, &planBatch).Logger(l.ctx).Info("RPC 恢复批次：批次状态已更新，事务已提交")
	return &trigger.ResumePlanBatchRes{}, nil
}
