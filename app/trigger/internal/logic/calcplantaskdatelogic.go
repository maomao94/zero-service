package logic

import (
	"context"
	"fmt"
	"time"

	"zero-service/app/trigger/internal/cronjob"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
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
	if len(in.StartTime) == 0 {
		in.StartTime = fmt.Sprintf("%d-1-1 00:00:00", time.Now().Year())
	}
	startTime := carbon.Parse(in.StartTime)
	if startTime.Error != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, startTime.Error, "开始时间格式错误")
	}
	if len(in.EndTime) == 0 {
		in.EndTime = fmt.Sprintf("%d-12-31 23:59:59", startTime.Year())
	}
	endTime := carbon.Parse(in.EndTime)
	if endTime.Error != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, endTime.Error, "结束时间格式错误")
	}
	if endTime.Lt(startTime) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "结束时间必须晚于开始时间")
	}
	if endTime.Gt(startTime.AddYears(3)) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "计划时间跨度不能超过3年")
	}
	rruleOption, err := l.ConvertToRRuleOption(in.Rule, startTime, endTime)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, err, "生成计划规则失败")
	}
	set := rrule.Set{}
	r, err := rrule.NewRRule(rruleOption)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, err, "生成计划日期失败")
	}
	set.RRule(r)
	// 添加排除日期
	for _, excludeDate := range in.ExcludeDates {
		excludeTime := carbon.ParseByFormat(excludeDate, carbon.DateFormat)
		if excludeTime.Error != nil || excludeTime.IsInvalid() {
			if excludeTime.Error != nil {
				return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, excludeTime.Error, "排除日期格式错误: %s", excludeDate)
			}
			return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "排除日期格式错误: %s", excludeDate)
		}
		// 为每个排除日期添加一天中的所有小时分钟组合
		for _, hour := range in.Rule.Hours {
			for _, minute := range in.Rule.Minutes {
				excludeDateTime := excludeTime.SetHour(int(hour)).SetMinute(int(minute)).SetSecond(0)
				set.ExDate(excludeDateTime.StdTime())
			}
		}
	}
	// 获取所有触发时间
	dates := set.All()
	var planDates []string
	for _, date := range dates {
		planDates = append(planDates, carbon.NewCarbon(date).ToDateTimeString())
	}
	return &trigger.CalcPlanTaskDateRes{
		PlanDates: planDates,
	}, nil
}

func (l *CalcPlanTaskDateLogic) ConvertToRRuleOption(planRule *trigger.PlanRulePb, startTime, endTime *carbon.Carbon) (rrule.ROption, error) {
	return cronjob.ConvertToRRuleOption(planRule, startTime.StdTime(), endTime.StdTime())
}
