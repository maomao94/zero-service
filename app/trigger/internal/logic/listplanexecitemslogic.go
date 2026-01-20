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
	if in.ExecId != "" {
		builder = builder.Where("exec_id = ?", in.ExecId)
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
			CreateTime:       carbon.CreateFromStdTime(execItem.CreateTime).ToDateTimeString(),
			UpdateTime:       carbon.CreateFromStdTime(execItem.UpdateTime).ToDateTimeString(),
			CreateUser:       execItem.CreateUser.String,
			UpdateUser:       execItem.UpdateUser.String,
			DeptCode:         execItem.DeptCode.String,
			Id:               execItem.Id,
			PlanPk:           execItem.PlanPk,
			PlanId:           execItem.PlanId,
			BatchPk:          execItem.BatchPk,
			BatchId:          execItem.BatchId,
			ExecId:           execItem.ExecId,
			ItemId:           execItem.ItemId,
			ItemType:         execItem.ItemType.String,
			ItemName:         execItem.ItemName.String,
			PointId:          execItem.PointId.String,
			ServiceAddr:      execItem.ServiceAddr,
			Payload:          execItem.Payload,
			RequestTimeout:   execItem.RequestTimeout,
			PlanTriggerTime:  carbon.CreateFromStdTime(execItem.PlanTriggerTime).ToDateTimeString(),
			NextTriggerTime:  carbon.CreateFromStdTime(execItem.NextTriggerTime).ToDateTimeString(),
			TriggerCount:     int32(execItem.TriggerCount),
			Status:           int32(execItem.Status),
			LastResult:       execItem.LastResult.String,
			LastMessage:      execItem.LastMessage.String,
			LastReason:       execItem.LastReason.String,
			TerminatedReason: execItem.TerminatedReason.String,
			PausedReason:     execItem.PausedReason.String,
			Ext1:             execItem.Ext1.String,
			Ext2:             execItem.Ext2.String,
			Ext3:             execItem.Ext3.String,
			Ext4:             execItem.Ext4.String,
			Ext5:             execItem.Ext5.String,
		}
		// 设置上次触发时间
		if execItem.LastTriggerTime.Valid {
			pbExecItem.LastTriggerTime = carbon.CreateFromStdTime(execItem.LastTriggerTime.Time).ToDateTimeString()
		}

		// 设置暂停时间和原因
		if execItem.PausedTime.Valid {
			pbExecItem.PausedTime = carbon.CreateFromStdTime(execItem.PausedTime.Time).ToDateTimeString()
		}

		resp.PlanExecItems = append(resp.PlanExecItems, pbExecItem)
	}

	return resp, nil
}
