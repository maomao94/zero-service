package logic

import (
	"context"

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
		builder = builder.Where("batch_id LIKE ?", "%"+in.BatchId+"%")
	}
	if len(in.Status) > 0 {
		statusInts := make([]int64, len(in.Status))
		for i, status := range in.Status {
			statusInts[i] = int64(status)
		}
		builder = builder.Where("status IN (?)", statusInts)
	}

	// 查询计划批次列表
	planBatches, total, err := l.svcCtx.PlanBatchModel.FindPageListByPageWithTotal(l.ctx, builder, in.PageNum, in.PageSize, "id DESC")
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
		pbPlanBatch := &trigger.PbPlanBatch{
			CreateTime:      carbon.CreateFromStdTime(planBatch.CreateTime).ToDateTimeString(),
			UpdateTime:      carbon.CreateFromStdTime(planBatch.UpdateTime).ToDateTimeString(),
			CreateUser:      planBatch.CreateUser.String,
			UpdateUser:      planBatch.UpdateUser.String,
			Id:              planBatch.Id,
			PlanPk:          planBatch.PlanPk,
			PlanId:          planBatch.PlanId,
			BatchId:         planBatch.BatchId,
			BatchName:       planBatch.BatchName.String,
			Status:          int32(planBatch.Status),
			PlanTriggerTime: carbon.CreateFromStdTime(planBatch.PlanTriggerTime.Time).ToDateTimeString(),
			Ext1:            planBatch.Ext1.String,
			Ext2:            planBatch.Ext2.String,
			Ext3:            planBatch.Ext3.String,
			Ext4:            planBatch.Ext4.String,
			Ext5:            planBatch.Ext5.String,
		}

		// 设置完成时间
		if planBatch.CompletedTime.Valid {
			pbPlanBatch.CompletedTime = carbon.CreateFromStdTime(planBatch.CompletedTime.Time).ToDateTimeString()
		}

		resp.PlanBatches = append(resp.PlanBatches, pbPlanBatch)
	}

	return resp, nil
}
