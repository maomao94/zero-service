package logic

import (
	"context"
	"encoding/json"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListPlansLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListPlansLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPlansLogic {
	return &ListPlansLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 分页获取计划列表
func (l *ListPlansLogic) ListPlans(in *trigger.ListPlansReq) (*trigger.ListPlansRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 构建查询条件
	builder := l.svcCtx.PlanModel.SelectBuilder()
	if in.PlanId != "" {
		builder = builder.Where("plan_id = ?", in.PlanId)
	}
	if in.PlanName != "" {
		builder = builder.Where("plan_name LIKE ?", "%"+in.PlanName+"%")
	}
	if in.Type != "" {
		builder = builder.Where("type = ?", in.Type)
	}
	if len(in.Status) > 0 {
		statusInts := make([]int64, len(in.Status))
		for i, status := range in.Status {
			statusInts[i] = int64(status)
		}
		builder = builder.Where("status IN (?) ", statusInts)
	}

	// 查询计划列表
	plans, total, err := l.svcCtx.PlanModel.FindPageListByPageWithTotal(l.ctx, builder, in.PageNum, in.PageSize, "id DESC")
	if err != nil {
		return nil, err
	}

	// 构建响应
	resp := &trigger.ListPlansRes{
		Plans: make([]*trigger.PbPlan, 0, len(plans)),
		Total: total,
	}

	// 转换计划列表
	for _, plan := range plans {
		// 解析规则
		var pbRule trigger.PbPlanRule
		err = json.Unmarshal([]byte(plan.RecurrenceRule), &pbRule)
		if err != nil {
			continue
		}

		pbPlan := &trigger.PbPlan{
			PlanId:       plan.PlanId,
			PlanName:     plan.PlanName.String,
			Type:         plan.Type.String,
			Description:  plan.Description.String,
			StartTime:    carbon.CreateFromStdTime(plan.StartTime).ToDateTimeString(),
			EndTime:      carbon.CreateFromStdTime(plan.EndTime).ToDateTimeString(),
			Rule:         &pbRule,
			Status:       int32(plan.Status),
			IsTerminated: plan.IsTerminated == 1,
			IsPaused:     plan.IsPaused == 1,
		}

		// 设置终止时间和原因
		if plan.IsTerminated == 1 && plan.TerminatedTime.Valid {
			pbPlan.TerminatedTime = carbon.CreateFromStdTime(plan.TerminatedTime.Time).ToDateTimeString()
			pbPlan.TerminatedReason = plan.TerminatedReason.String
		}

		// 设置暂停时间和原因
		if plan.IsPaused == 1 && plan.PausedTime.Valid {
			pbPlan.PausedTime = carbon.CreateFromStdTime(plan.PausedTime.Time).ToDateTimeString()
			pbPlan.PausedReason = plan.PausedReason.String
		}

		resp.Plans = append(resp.Plans, pbPlan)
	}

	return resp, nil
}
