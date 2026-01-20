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

		// 设置暂停时间和原因
		if plan.PausedTime.Valid {
			pbPlan.PausedTime = carbon.CreateFromStdTime(plan.PausedTime.Time).ToDateTimeString()
		}

		progress, err := l.svcCtx.PlanBatchModel.CalculatePlanProgress(l.ctx, plan.Id)
		if err != nil {
			return nil, err
		}
		pbPlan.Progress = progress

		resp.Plans = append(resp.Plans, pbPlan)
	}

	return resp, nil
}
