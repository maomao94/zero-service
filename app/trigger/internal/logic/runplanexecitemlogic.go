package logic

import (
	"context"
	"fmt"
	"time"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/model"

	"github.com/zeromicro/go-zero/core/logx"
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
	execItem, err := l.svcCtx.PlanExecItemModel.FindOne(l.ctx, in.Id)
	if err != nil {
		return nil, err
	}
	if execItem.Status != int64(model.StatusWaiting) && execItem.Status != int64(model.StatusDelayed) {
		return nil, fmt.Errorf("执行项当前状态为%d，无法立即执行，仅支持等待调度(0)或延期等待(10)状态", execItem.Status)
	}

	// 更新下次触发时间为当前时间，使其立即执行
	now := time.Now()
	execItem.NextTriggerTime = now
	err = l.svcCtx.PlanExecItemModel.UpdateWithVersion(l.ctx, nil, execItem)
	if err != nil {
		return nil, err
	}

	l.Logger.Infof("Plan exec item %d will be executed immediately, nextTriggerTime updated to %v, current status: %d", execItem.Id, now, execItem.Status)

	return &trigger.RunPlanExecItemRes{}, nil
}
