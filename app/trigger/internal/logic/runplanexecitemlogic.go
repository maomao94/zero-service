package logic

import (
	"context"
	"fmt"
	"time"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/model"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/songzhibin97/gkit/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
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
	if in.Id <= 0 && strutil.IsBlank(in.ExecId) {
		return nil, errors.BadRequest("", "参数错误")
	}
	// 查询执行项
	var execItem *model.PlanExecItem
	if in.Id > 0 {
		execItem, err = l.svcCtx.PlanExecItemModel.FindOne(l.ctx, in.Id)
	} else {
		execItem, err = l.svcCtx.PlanExecItemModel.FindOneByExecId(l.ctx, in.ExecId)
	}
	if err != nil {
		if err == sqlx.ErrNotFound {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)
		}
		return nil, err
	}
	if execItem.Status != int64(model.StatusWaiting) && execItem.Status != int64(model.StatusDelayed) {
		return nil, fmt.Errorf("执行项当前状态为%d，无法立即执行，仅支持等待调度(0)或延期等待(10)状态", execItem.Status)
	}
	// 查询计划批次
	planBatch, err := l.svcCtx.PlanBatchModel.FindOne(l.ctx, execItem.BatchPk)
	if err != nil {
		return nil, err
	}

	// 查询计划
	plan, err := l.svcCtx.PlanModel.FindOneByPlanId(l.ctx, execItem.PlanId)
	if err != nil {
		return nil, err
	}

	if plan.Status == int64(model.PlanStatusTerminated) || plan.FinishedTime.Valid {
		return nil, errors.BadRequest("", "计划状态已结束,不可立即执行")
	}

	if planBatch.Status == int64(model.PlanStatusTerminated) || planBatch.FinishedTime.Valid {
		return nil, errors.BadRequest("", "计划批次状态已结束,不可立即执行")
	}

	if plan.Status == int64(model.PlanStatusPaused) {
		return nil, errors.BadRequest("", "计划处于暂停状态,不可立即执行")
	}

	if planBatch.Status == int64(model.PlanStatusPaused) {
		return nil, errors.BadRequest("", "计划批次处于暂停状态,不可立即执行")
	}

	// 更新下次触发时间为当前时间，使其立即执行
	now := time.Now()
	execItem.NextTriggerTime = now
	err = l.svcCtx.PlanExecItemModel.UpdateWithVersion(l.ctx, nil, execItem)
	if err != nil {
		return nil, err
	}

	planscope.ExecScope(execItem).WithFields(
		logx.Field("plan_name", plan.PlanName.String),
		logx.Field("next_trigger", execItem.NextTriggerTime.Format(time.RFC3339Nano)),
		logx.Field("status", execItem.Status),
	).Logger(l.ctx).Info("RPC 立即执行：已将本执行项的下次调度时间改为当前时间，等待定时扫表触发下游")

	return &trigger.RunPlanExecItemRes{}, nil
}
