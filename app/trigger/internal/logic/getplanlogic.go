package logic

import (
	"context"
	"encoding/json"
	"zero-service/model"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/dromara/carbon/v2"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/songzhibin97/gkit/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetPlanLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetPlanLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPlanLogic {
	return &GetPlanLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取计划详情
func (l *GetPlanLogic) GetPlan(in *trigger.GetPlanReq) (*trigger.GetPlanRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	if in.Id <= 0 && strutil.IsBlank(in.PlanId) {
		return nil, errors.BadRequest("", "参数错误")
	}
	var plan *model.Plan
	if in.Id > 0 {
		plan, err = l.svcCtx.PlanModel.FindOne(l.ctx, in.Id)
	} else {
		plan, err = l.svcCtx.PlanModel.FindOneByPlanId(l.ctx, in.PlanId)
	}
	if err != nil {
		return nil, err
	}
	// 解析规则
	var pbRule trigger.PbPlanRule
	err = json.Unmarshal([]byte(plan.RecurrenceRule), &pbRule)
	if err != nil {
		return nil, err
	}

	// 构建响应
	pbPlan := &trigger.PbPlan{
		CreateTime:       carbon.CreateFromStdTime(plan.CreateTime).ToDateTimeString(),
		UpdateTime:       carbon.CreateFromStdTime(plan.UpdateTime).ToDateTimeString(),
		CreateUser:       plan.CreateUser.String,
		UpdateUser:       plan.UpdateUser.String,
		DeptCode:         plan.DeptCode.String,
		Id:               plan.Id,
		PlanId:           plan.PlanId,
		PlanName:         plan.PlanName.String,
		Type:             plan.Type.String,
		GroupId:          plan.GroupId.String,
		Description:      plan.Description.String,
		StartTime:        carbon.CreateFromStdTime(plan.StartTime).ToDateTimeString(),
		EndTime:          carbon.CreateFromStdTime(plan.EndTime).ToDateTimeString(),
		Rule:             &pbRule,
		Status:           int32(plan.Status),
		TerminatedReason: plan.TerminatedReason.String,
		PausedReason:     plan.PausedReason.String,
		Ext1:             plan.Ext1.String,
		Ext2:             plan.Ext2.String,
		Ext3:             plan.Ext3.String,
		Ext4:             plan.Ext4.String,
		Ext5:             plan.Ext5.String,
	}

	// 设置终止时间和原因
	if plan.TerminatedTime.Valid {
		pbPlan.TerminatedTime = carbon.CreateFromStdTime(plan.TerminatedTime.Time).ToDateTimeString()
	}

	// 设置暂停时间和原因
	if plan.PausedTime.Valid {
		pbPlan.PausedTime = carbon.CreateFromStdTime(plan.PausedTime.Time).ToDateTimeString()
	}
	progress, err := l.svcCtx.PlanBatchModel.CalculatePlanProgress(l.ctx, plan.Id)
	if err != nil {
		return nil, err
	}
	pbPlan.Progress = progress
	return &trigger.GetPlanRes{
		Plan: pbPlan,
	}, nil
}
