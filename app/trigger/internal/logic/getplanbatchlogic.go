package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetPlanBatchLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetPlanBatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPlanBatchLogic {
	return &GetPlanBatchLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取计划批次详情
func (l *GetPlanBatchLogic) GetPlanBatch(in *trigger.GetPlanBatchReq) (*trigger.GetPlanBatchRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 查询计划批次
	planBatch, err := l.svcCtx.PlanBatchModel.FindOne(l.ctx, in.Id)
	if err != nil {
		return nil, err
	}

	// 构建响应
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

	return &trigger.GetPlanBatchRes{
		PlanBatch: pbPlanBatch,
	}, nil
}
