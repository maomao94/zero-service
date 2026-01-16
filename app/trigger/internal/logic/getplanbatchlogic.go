package logic

import (
	"context"
	"zero-service/model"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/dromara/carbon/v2"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/songzhibin97/gkit/errors"
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

	if in.Id <= 0 && strutil.IsBlank(in.BatchId) {
		return nil, errors.BadRequest("", "参数错误")
	}
	var planBatch *model.PlanBatch
	if in.Id > 0 {
		planBatch, err = l.svcCtx.PlanBatchModel.FindOne(l.ctx, in.Id)
	} else {
		planBatch, err = l.svcCtx.PlanBatchModel.FindOneByBatchId(l.ctx, in.BatchId)
	}
	if err != nil {
		return nil, err
	}
	// 构建响应
	pbPlanBatch := &trigger.PbPlanBatch{
		CreateTime:      carbon.CreateFromStdTime(planBatch.CreateTime).ToDateTimeString(),
		UpdateTime:      carbon.CreateFromStdTime(planBatch.UpdateTime).ToDateTimeString(),
		CreateUser:      planBatch.CreateUser.String,
		UpdateUser:      planBatch.UpdateUser.String,
		DeptCode:        planBatch.DeptCode.String,
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
