package logic

import (
	"context"
	"database/sql"
	"time"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/common/authctx"
	"zero-service/common/errx"
	"zero-service/model"
	"zero-service/third_party/extproto"

	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
)

type PausePlanExecItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPausePlanExecItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PausePlanExecItemLogic {
	return &PausePlanExecItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 暂停执行项
func (l *PausePlanExecItemLogic) PausePlanExecItem(in *trigger.PausePlanExecItemReq) (*trigger.PausePlanExecItemRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 检查参数
	if strutil.IsBlank(in.Id) && strutil.IsBlank(in.ExecId) {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_PARAM, "参数错误")
	}

	// 查询执行项
	db := l.svcCtx.DB.WithContext(l.ctx).DB
	var execItem gormmodel.PlanExecItem
	if !strutil.IsBlank(in.Id) {
		err = db.Where("id = ?", in.Id).First(&execItem).Error
	} else {
		err = db.Where("exec_id = ?", in.ExecId).First(&execItem).Error
	}
	if err != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询执行项失败")
	}
	// 查询计划批次
	var planBatch gormmodel.PlanBatch
	if err := db.Where("id = ?", execItem.BatchPk).First(&planBatch).Error; err != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询计划批次失败")
	}
	// 查询计划
	var plan gormmodel.Plan
	if err := db.Where("plan_id = ?", execItem.PlanId).First(&plan).Error; err != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询计划失败")
	}
	if plan.Status == model.PlanStatusTerminated || plan.FinishedTime.Valid {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划状态已结束,无需暂停")
	}

	if planBatch.Status == model.PlanStatusTerminated || planBatch.FinishedTime.Valid {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划批次状态已结束,无需暂停")
	}

	if execItem.Status == model.StatusCompleted || execItem.Status == model.StatusTerminated || execItem.Status == model.StatusPaused {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "执行项状态已结束,无需暂停")
	}

	if execItem.Status == model.StatusRunning {
		return &trigger.PausePlanExecItemRes{}, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "执行项正在运行中，请稍后再试")
	}

	// 更新执行项状态为暂停；更新带上读取时的状态条件，期间被终止或抢占时不覆盖已变更的状态。
	now := time.Now()
	result := db.Model(&gormmodel.PlanExecItem{}).
		Where("id = ?", execItem.Id).
		Where("status = ?", execItem.Status).
		Updates(map[string]any{
			"status":        model.StatusPaused,
			"paused_time":   sql.NullTime{Time: now, Valid: true},
			"paused_reason": sql.NullString{String: in.Reason, Valid: in.Reason != ""},
			"update_user":   sql.NullString{String: authctx.GetUserId(l.ctx), Valid: authctx.GetUserId(l.ctx) != ""},
			"update_time":   now,
		})
	if result.Error != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, result.Error, "暂停执行项失败")
	}
	if result.RowsAffected == 0 {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "执行项状态已变化，请刷新后重试")
	}
	execItem.Status = model.StatusPaused

	planscope.ExecScope(&execItem).Logger(l.ctx).Info("RPC 暂停执行项：执行项状态已更新")
	return &trigger.PausePlanExecItemRes{}, nil
}
