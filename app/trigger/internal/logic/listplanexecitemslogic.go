package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListPlanExecItemsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListPlanExecItemsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPlanExecItemsLogic {
	return &ListPlanExecItemsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 分页获取执行项列表
func (l *ListPlanExecItemsLogic) ListPlanExecItems(in *trigger.ListPlanExecItemsReq) (*trigger.ListPlanExecItemsRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 构建查询条件
	builder := l.svcCtx.PlanExecItemModel.SelectBuilder()

	// 处理计划主键id
	if in.Id > 0 {
		// 根据计划主键id查询plan_id
		plan, err := l.svcCtx.PlanModel.FindOne(l.ctx, in.Id)
		if err != nil {
			return nil, err
		}
		in.PlanId = plan.PlanId
	}

	if in.PlanId != "" {
		builder = builder.Where("plan_id = ?", in.PlanId)
	}
	if in.ItemId != "" {
		builder = builder.Where("item_id LIKE ?", "%"+in.ItemId+"%")
	}
	if in.ItemName != "" {
		builder = builder.Where("item_name LIKE ?", "%"+in.ItemName+"%")
	}
	if len(in.Status) > 0 {
		statusInts := make([]int64, len(in.Status))
		for i, status := range in.Status {
			statusInts[i] = int64(status)
		}
		builder = builder.Where("status IN (?) ", statusInts)
	}

	// 查询执行项列表
	execItems, total, err := l.svcCtx.PlanExecItemModel.FindPageListByPageWithTotal(l.ctx, builder, in.PageNum, in.PageSize, "id DESC")
	if err != nil {
		return nil, err
	}

	// 构建响应
	resp := &trigger.ListPlanExecItemsRes{
		PlanExecItems: make([]*trigger.PbPlanExecItem, 0, len(execItems)),
		Total:         total,
	}

	// 转换执行项列表
	for _, execItem := range execItems {
		pbExecItem := &trigger.PbPlanExecItem{
			PlanId:          execItem.PlanId,
			ItemId:          execItem.ItemId,
			ItemName:        execItem.ItemName.String,
			ServiceAddr:     execItem.ServiceAddr,
			Payload:         execItem.Payload,
			RequestTimeout:  execItem.RequestTimeout,
			PlanTriggerTime: carbon.CreateFromStdTime(execItem.PlanTriggerTime).ToDateTimeString(),
			Status:          int32(execItem.Status),
			LastResult:      execItem.LastResult.String,
			LastMsg:         execItem.LastMsg.String,
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
			pbExecItem.TerminatedReason = execItem.TerminatedReason.String
		}

		// 设置暂停时间和原因
		if execItem.IsPaused == 1 && execItem.PausedTime.Valid {
			pbExecItem.PausedTime = carbon.CreateFromStdTime(execItem.PausedTime.Time).ToDateTimeString()
			pbExecItem.PausedReason = execItem.PausedReason.String
		}

		resp.PlanExecItems = append(resp.PlanExecItems, pbExecItem)
	}

	return resp, nil
}
