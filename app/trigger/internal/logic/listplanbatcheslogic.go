package logic

import (
	"context"
	"fmt"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListPlanBatchesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListPlanBatchesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPlanBatchesLogic {
	return &ListPlanBatchesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 分页获取计划批次列表
func (l *ListPlanBatchesLogic) ListPlanBatches(in *trigger.ListPlanBatchesReq) (*trigger.ListPlanBatchesRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 构建查询条件
	builder := l.svcCtx.PlanBatchModel.SelectBuilder()
	if in.PlanId != "" {
		builder = builder.Where("plan_id = ?", in.PlanId)
	}
	if in.BatchId != "" {
		builder = builder.Where("batch_id = ?", in.BatchId)
	}
	if len(in.Status) > 0 {
		statusInts := make([]int64, len(in.Status))
		for i, status := range in.Status {
			statusInts[i] = int64(status)
		}
		builder = builder.Where("status IN (?)", statusInts)
	}

	// 查询计划批次列表
	planBatches, total, err := l.svcCtx.PlanBatchModel.FindPageListByPageWithTotal(l.ctx, builder, in.PageNum, in.PageSize, "plan_trigger_time DESC", "id DESC")
	if err != nil {
		return nil, err
	}

	// 构建响应
	resp := &trigger.ListPlanBatchesRes{
		PlanBatches: make([]*trigger.PbPlanBatch, 0, len(planBatches)),
		Total:       total,
	}

	// 转换计划批次列表
	for _, planBatch := range planBatches {
		// 获取批次执行项状态统计
		statusCounts, err := l.svcCtx.PlanExecItemModel.GetBatchStatusCounts(l.ctx, planBatch.Id)
		if err != nil {
			return nil, err
		}

		// 获取批次总执行项数
		totalExecItems, err := l.svcCtx.PlanExecItemModel.GetBatchTotalExecItems(l.ctx, planBatch.Id)
		if err != nil {
			return nil, err
		}
		statusCountMap := make(map[string]int64)

		// 统计各状态数量
		for _, sc := range statusCounts {
			statusStr := fmt.Sprintf("%d", sc.Status)
			statusCountMap[statusStr] = sc.Count
		}

		pbPlanBatch := &trigger.PbPlanBatch{
			CreateTime:       carbon.CreateFromStdTime(planBatch.CreateTime).ToDateTimeString(),
			UpdateTime:       carbon.CreateFromStdTime(planBatch.UpdateTime).ToDateTimeString(),
			CreateUser:       planBatch.CreateUser.String,
			UpdateUser:       planBatch.UpdateUser.String,
			DeptCode:         planBatch.DeptCode.String,
			Id:               planBatch.Id,
			PlanPk:           planBatch.PlanPk,
			PlanId:           planBatch.PlanId,
			BatchId:          planBatch.BatchId,
			BatchName:        planBatch.BatchName.String,
			Status:           int32(planBatch.Status),
			ExecCnt:          totalExecItems,
			PlanTriggerTime:  carbon.CreateFromStdTime(planBatch.PlanTriggerTime.Time).ToDateTimeString(),
			TerminatedReason: planBatch.TerminatedReason.String,
			PausedReason:     planBatch.PausedReason.String,
			StatusCountMap:   statusCountMap,
			Ext1:             planBatch.Ext1.String,
			Ext2:             planBatch.Ext2.String,
			Ext3:             planBatch.Ext3.String,
			Ext4:             planBatch.Ext4.String,
			Ext5:             planBatch.Ext5.String,
		}
		if planBatch.PausedTime.Valid {
			pbPlanBatch.PausedTime = carbon.CreateFromStdTime(planBatch.PausedTime.Time).ToDateTimeString()
		}
		if planBatch.FinishedTime.Valid {
			pbPlanBatch.FinishedTime = carbon.CreateFromStdTime(planBatch.FinishedTime.Time).ToDateTimeString()
		}

		resp.PlanBatches = append(resp.PlanBatches, pbPlanBatch)
	}
	return resp, nil
}
