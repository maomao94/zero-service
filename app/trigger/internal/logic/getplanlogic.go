package logic

import (
	"context"
	"errors"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/common/carbonx"
	"zero-service/common/rrulex"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"
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
	if strutil.IsBlank(in.Id) && strutil.IsBlank(in.PlanId) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "参数错误")
	}
	var plan gormmodel.Plan
	if !strutil.IsBlank(in.Id) {
		err = l.svcCtx.DB.WithContext(l.ctx).Where("id = ?", in.Id).First(&plan).Error
	} else {
		err = l.svcCtx.DB.WithContext(l.ctx).Where("plan_id = ?", in.PlanId).First(&plan).Error
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)
		}
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询计划失败")
	}
	// 解析规则
	var pbRule trigger.PlanRulePb
	err = protojson.Unmarshal([]byte(plan.RecurrenceRule), &pbRule)
	if err != nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "计划规则格式错误")
	}
	scheduleDescription, err := rrulex.Describe(plan.RRuleStr)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, err, "生成计划规则描述失败")
	}

	// 构建响应
	pbPlan := &trigger.PlanPb{
		CreateTime:          carbonx.FormatDateTime(plan.CreateTime),
		UpdateTime:          carbonx.FormatDateTime(plan.UpdateTime),
		CreateUser:          plan.CreateUser.String,
		UpdateUser:          plan.UpdateUser.String,
		DeptCode:            plan.DeptCode.String,
		Id:                  plan.Id,
		PlanId:              plan.PlanId,
		PlanName:            plan.PlanName.String,
		Type:                plan.Type.String,
		GroupId:             plan.GroupId.String,
		Description:         plan.Description.String,
		StartTime:           carbonx.FormatDateTime(plan.StartTime),
		EndTime:             carbonx.FormatDateTime(plan.EndTime),
		Rule:                &pbRule,
		Status:              int32(plan.Status),
		ScanFlg:             int32(plan.ScanFlg),
		TerminatedReason:    plan.TerminatedReason.String,
		PausedReason:        plan.PausedReason.String,
		Ext1:                plan.Ext1.String,
		Ext2:                plan.Ext2.String,
		Ext3:                plan.Ext3.String,
		Ext4:                plan.Ext4.String,
		Ext5:                plan.Ext5.String,
		RruleStr:            plan.RRuleStr,
		ScheduleDescription: scheduleDescription,
	}
	// 设置暂停时间和原因
	if plan.PausedTime.Valid {
		pbPlan.PausedTime = carbonx.FormatDateTime(plan.PausedTime.Time)
	}
	progress, err := gormmodel.CalculatePlanProgress(l.ctx, l.svcCtx.DB.WithContext(l.ctx).DB, plan.Id)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "计算计划进度失败")
	}
	pbPlan.Progress = progress
	return &trigger.GetPlanRes{
		Plan: pbPlan,
	}, nil
}
