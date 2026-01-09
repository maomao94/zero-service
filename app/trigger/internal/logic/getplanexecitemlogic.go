package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type GetPlanExecItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetPlanExecItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPlanExecItemLogic {
	return &GetPlanExecItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取执行项详情
func (l *GetPlanExecItemLogic) GetPlanExecItem(in *trigger.GetPlanExecItemReq) (*trigger.GetPlanExecItemRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 查询执行项
	execItem, err := l.svcCtx.PlanExecItemModel.FindOneByPlanIdItemId(l.ctx, in.PlanId, in.ItemId)
	if err != nil {
		if err == sqlx.ErrNotFound {
			return nil, err
		}
		return nil, err
	}

	// 构建响应
	pbExecItem := &trigger.PbPlanExecItem{
		PlanId:          execItem.PlanId,
		ItemId:          execItem.ItemId,
		ItemName:        execItem.ItemName,
		ServiceAddr:     execItem.ServiceAddr,
		Payload:         execItem.Payload,
		RequestTimeout:  execItem.RequestTimeout,
		PlanTriggerTime: carbon.CreateFromStdTime(execItem.PlanTriggerTime).ToDateTimeString(),
		Status:          int32(execItem.Status),
		LastResult:      execItem.LastResult,
		LastError:       execItem.LastError,
		IsTerminated:    execItem.IsTerminated == 1,
		IsPaused:        execItem.IsPaused == 1,
		TriggerCount:    int32(execItem.TriggerCount),
	}

	// 设置下次触发时间
	if !execItem.NextTriggerTime.IsZero() {
		pbExecItem.NextTriggerTime = carbon.CreateFromStdTime(execItem.NextTriggerTime).ToDateTimeString()
	}

	// 设置上次触发时间
	if execItem.LastTriggerTime.Valid {
		pbExecItem.LastTriggerTime = carbon.CreateFromStdTime(execItem.LastTriggerTime.Time).ToDateTimeString()
	}

	// 设置终止时间和原因
	if execItem.IsTerminated == 1 && execItem.TerminatedTime.Valid {
		pbExecItem.TerminatedTime = carbon.CreateFromStdTime(execItem.TerminatedTime.Time).ToDateTimeString()
		pbExecItem.TerminatedReason = execItem.TerminatedReason
	}

	// 设置暂停时间和原因
	if execItem.IsPaused == 1 && execItem.PausedTime.Valid {
		pbExecItem.PausedTime = carbon.CreateFromStdTime(execItem.PausedTime.Time).ToDateTimeString()
		pbExecItem.PausedReason = execItem.PausedReason
	}

	return &trigger.GetPlanExecItemRes{
		PlanExecItem: pbExecItem,
	}, nil
}
