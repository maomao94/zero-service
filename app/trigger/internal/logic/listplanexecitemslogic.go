package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/gormx"

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
	db := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.PlanExecItem{})

	// 处理计划主键id
	if in.Id != "" {
		// 根据计划主键id查询plan_id
		var plan gormmodel.Plan
		if err := l.svcCtx.DB.WithContext(l.ctx).Where("id = ?", in.Id).First(&plan).Error; err != nil {
			return nil, err
		}
		in.PlanId = plan.PlanId
	}

	if in.PlanId != "" {
		db = db.Where("plan_id = ?", in.PlanId)
	}
	if in.BatchId != "" {
		db = db.Where("batch_id = ?", in.BatchId)
	}
	if in.ExecId != "" {
		db = db.Where("exec_id = ?", in.ExecId)
	}
	if in.ItemId != "" {
		db = db.Where("item_id LIKE ?", "%"+in.ItemId+"%")
	}
	if in.ItemName != "" {
		db = db.Where("item_name LIKE ?", "%"+in.ItemName+"%")
	}
	if len(in.Status) > 0 {
		statusInts := make([]int64, len(in.Status))
		for i, status := range in.Status {
			statusInts[i] = int64(status)
		}
		db = db.Where("status IN ?", statusInts)
	}

	var items []gormmodel.PlanExecItem
	page, err := gormx.QueryPage(db.Order("next_trigger_time ASC, status ASC, id DESC"), int(in.PageNum), int(in.PageSize), &items)
	if err != nil {
		return nil, err
	}

	// 构建响应
	resp := &trigger.ListPlanExecItemsRes{
		PlanExecItems: make([]*trigger.PlanExecItemPb, 0, len(items)),
		Total:         page.Total,
	}

	// 转换执行项列表
	for i := range items {
		pbExecItem := &trigger.PlanExecItemPb{
			CreateTime:       carbon.CreateFromStdTime(items[i].CreateTime).ToDateTimeString(),
			UpdateTime:       carbon.CreateFromStdTime(items[i].UpdateTime).ToDateTimeString(),
			CreateUser:       items[i].CreateUser.String,
			UpdateUser:       items[i].UpdateUser.String,
			DeptCode:         items[i].DeptCode.String,
			Id:               items[i].Id,
			PlanPk:           items[i].PlanPk,
			PlanId:           items[i].PlanId,
			BatchPk:          items[i].BatchPk,
			BatchId:          items[i].BatchId,
			ExecId:           items[i].ExecId,
			ItemId:           items[i].ItemId,
			ItemType:         items[i].ItemType.String,
			ItemName:         items[i].ItemName.String,
			ItemRowId:        items[i].ItemRowId,
			PointId:          items[i].PointId.String,
			Payload:          items[i].Payload,
			RequestTimeout:   items[i].RequestTimeout,
			PlanTriggerTime:  carbon.CreateFromStdTime(items[i].PlanTriggerTime).ToDateTimeString(),
			NextTriggerTime:  carbon.CreateFromStdTime(items[i].NextTriggerTime).ToDateTimeString(),
			TriggerCount:     int32(items[i].TriggerCount),
			Status:           trigger.ExecItemStatusPb(items[i].Status),
			LastResult:       items[i].LastResult.String,
			LastMessage:      items[i].LastMessage.String,
			LastReason:       items[i].LastReason.String,
			TerminatedReason: items[i].TerminatedReason.String,
			PausedReason:     items[i].PausedReason.String,
			Ext1:             items[i].Ext1.String,
			Ext2:             items[i].Ext2.String,
			Ext3:             items[i].Ext3.String,
			Ext4:             items[i].Ext4.String,
			Ext5:             items[i].Ext5.String,
		}
		// 设置上次触发时间
		if items[i].LastTriggerTime.Valid {
			pbExecItem.LastTriggerTime = carbon.CreateFromStdTime(items[i].LastTriggerTime.Time).ToDateTimeString()
		}

		// 设置暂停时间和原因
		if items[i].PausedTime.Valid {
			pbExecItem.PausedTime = carbon.CreateFromStdTime(items[i].PausedTime.Time).ToDateTimeString()
		}

		resp.PlanExecItems = append(resp.PlanExecItems, pbExecItem)
	}

	return resp, nil
}
