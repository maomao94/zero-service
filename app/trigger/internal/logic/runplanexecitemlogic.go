package logic

import (
	"context"
	"errors"
	"fmt"
	"time"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/model"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type RunPlanExecItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRunPlanExecItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RunPlanExecItemLogic {
	return &RunPlanExecItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 立即执行计划项
func (l *RunPlanExecItemLogic) RunPlanExecItem(in *trigger.RunPlanExecItemReq) (*trigger.RunPlanExecItemRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	// 检查参数
	if strutil.IsBlank(in.Id) && strutil.IsBlank(in.ExecId) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "参数错误")
	}
	// 查询执行项
	var execItem gormmodel.PlanExecItem
	db := l.svcCtx.DB.WithContext(l.ctx).DB
	if !strutil.IsBlank(in.Id) {
		err = db.Where("id = ?", in.Id).First(&execItem).Error
	} else {
		err = db.Where("exec_id = ?", in.ExecId).First(&execItem).Error
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)
		}
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_DB, "查询执行项失败")
	}
	if execItem.Status != model.StatusWaiting && execItem.Status != model.StatusDelayed {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, fmt.Sprintf("执行项当前状态为%d，无法立即执行，仅支持等待调度(0)或延期等待(10)状态", execItem.Status))
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

	if plan.Status == model.PlanStatusTerminated || plan.FinishedTime.Valid {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划状态已结束,不可立即执行")
	}

	if planBatch.Status == model.PlanStatusTerminated || planBatch.FinishedTime.Valid {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划批次状态已结束,不可立即执行")
	}

	if plan.Status == model.PlanStatusPaused {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划处于暂停状态,不可立即执行")
	}

	if planBatch.Status == model.PlanStatusPaused {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划批次处于暂停状态,不可立即执行")
	}

	// 更新下次触发时间为当前时间，使其立即执行
	now := time.Now()
	execItem.NextTriggerTime = now
	if err := db.Save(&execItem).Error; err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "更新执行项失败")
	}

	planscope.ExecScope(&execItem).WithFields(
		logx.Field("plan_name", plan.PlanName.String),
		logx.Field("next_trigger", execItem.NextTriggerTime.Format(time.RFC3339Nano)),
		logx.Field("status", execItem.Status),
	).Logger(l.ctx).Info("RPC 立即执行：已将本执行项的下次调度时间改为当前时间，等待定时扫表触发下游")

	return &trigger.RunPlanExecItemRes{}, nil
}
