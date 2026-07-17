package logic

import (
	"context"
	"database/sql"
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
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "参数错误")
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
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询执行项失败")
	}
	// 查询计划批次
	var planBatch gormmodel.PlanBatch
	if err := db.Where("id = ?", execItem.BatchPk).First(&planBatch).Error; err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询计划批次失败")
	}
	// 查询计划
	var plan gormmodel.Plan
	if err := db.Where("plan_id = ?", execItem.PlanId).First(&plan).Error; err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询计划失败")
	}
	if plan.Status == int64(model.PlanStatusTerminated) || plan.FinishedTime.Valid {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划状态已结束,无需暂停")
	}

	if planBatch.Status == int64(model.PlanStatusTerminated) || planBatch.FinishedTime.Valid {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划批次状态已结束,无需暂停")
	}

	if execItem.Status == int64(model.StatusCompleted) || execItem.Status == int64(model.StatusTerminated) || execItem.Status == int64(model.StatusPaused) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "执行项状态已结束,无需暂停")
	}

	if execItem.Status == int64(model.StatusRunning) {
		return &trigger.PausePlanExecItemRes{}, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "执行项正在运行中，请稍后再试")
	}

	// 执行事务
	err = db.Transaction(func(tx *gorm.DB) error {
		// 更新执行项状态为暂停
		execItem.Status = int64(model.StatusPaused)
		execItem.PausedTime = sql.NullTime{Time: time.Now(), Valid: true}
		execItem.PausedReason = sql.NullString{String: in.Reason, Valid: in.Reason != ""}
		execItem.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(l.ctx, nil), Valid: tool.GetCurrentUserId(l.ctx, nil) != ""}

		// 更新执行项
		return tx.Save(&execItem).Error
	})

	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "暂停执行项事务失败")
	}

	planscope.ExecScope(&execItem).Logger(l.ctx).Info("RPC 暂停执行项：执行项状态已更新，事务已提交")
	return &trigger.PausePlanExecItemRes{}, nil
}
