package logic

import (
	"context"
	"encoding/json"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
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

	// 查询计划
	plan, err := l.svcCtx.PlanModel.FindOne(l.ctx, in.Id)
	if err != nil {
		if err == sqlx.ErrNotFound {
			return nil, err
		}
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
		PlanId:      plan.PlanId,
		PlanName:    plan.PlanName.String,
		Type:        plan.Type.String,
		Description: plan.Description.String,
		StartTime:   carbon.CreateFromStdTime(plan.StartTime).ToDateTimeString(),
		EndTime:     carbon.CreateFromStdTime(plan.EndTime).ToDateTimeString(),
		Rule:        &pbRule,
		Status:      int32(plan.Status),
	}

	// 设置终止时间和原因
	if plan.TerminatedTime.Valid {
		pbPlan.TerminatedTime = carbon.CreateFromStdTime(plan.TerminatedTime.Time).ToDateTimeString()
		pbPlan.TerminatedReason = plan.TerminatedReason.String
	}

	// 设置暂停时间和原因
	if plan.PausedTime.Valid {
		pbPlan.PausedTime = carbon.CreateFromStdTime(plan.PausedTime.Time).ToDateTimeString()
		pbPlan.PausedReason = plan.PausedReason.String
	}

	return &trigger.GetPlanRes{
		Plan: pbPlan,
	}, nil
}
