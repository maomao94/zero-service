package logic

import (
	"context"
	"time"

	"zero-service/app/trigger/internal/cronjob"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/crontask"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/dromara/carbon/v2"
	"github.com/teambition/rrule-go"
	"github.com/zeromicro/go-zero/core/logx"
)

type CalcPlanTaskDateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCalcPlanTaskDateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CalcPlanTaskDateLogic {
	return &CalcPlanTaskDateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 计算计划任务日期
func (l *CalcPlanTaskDateLogic) CalcPlanTaskDate(in *trigger.CalcPlanTaskDateReq) (*trigger.CalcPlanTaskDateRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	schedule, err := cronjob.CompileSchedule(in.Rule, in.StartTime, in.EndTime, in.ExcludeDates, in.SpecifiedTimes, in.ExcludedTimes, false, time.Now())
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, err, "生成计划规则失败")
	}
	set, err := rrule.StrToRRuleSet(schedule.RRuleStr)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, err, "解析计划规则失败")
	}
	// 获取所有触发时间
	dates := set.All()
	var planDates []string
	for _, date := range dates {
		planDates = append(planDates, carbon.NewCarbon(date).ToDateTimeString())
	}
	scheduleDescription, err := crontask.DescribeRRule(schedule.RRuleStr)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, err, "生成计划规则描述失败")
	}
	return &trigger.CalcPlanTaskDateRes{
		PlanDates:           planDates,
		ScheduleDescription: scheduleDescription,
		RruleStr:            schedule.RRuleStr,
	}, nil
}
