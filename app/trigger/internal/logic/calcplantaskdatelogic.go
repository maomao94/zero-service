package logic

import (
	"context"
	"errors"
	"fmt"
	"time"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

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
		return nil, startTime.Error
	}
	if len(in.EndTime) == 0 {
		in.EndTime = fmt.Sprintf("%d-12-31 23:59:59", startTime.Year())
	}
	endTime := carbon.Parse(in.EndTime)
	if endTime.Error != nil {
		return nil, endTime.Error
	}
	if endTime.Lt(startTime) {
		return nil, errors.New("结束时间必须晚于开始时间")
	}
	if endTime.Gt(startTime.AddYears(3)) {
		return nil, errors.New("计划时间跨度不能超过3年")
	}
	rruleOption, err := l.ConvertToRRuleOption(in.Rule, startTime, endTime)
	if err != nil {
		return nil, err
	}
	set := rrule.Set{}
	r, err := rrule.NewRRule(rruleOption)
	if err != nil {
		return nil, err
	}
	set.RRule(r)
	// 添加排除日期
	for _, excludeDate := range in.ExcludeDates {
		excludeTime := carbon.ParseByFormat(excludeDate, carbon.DateFormat)
		if excludeTime.Error != nil || excludeTime.IsInvalid() {
			return nil, fmt.Errorf("排除日期格式错误: %s", excludeDate)
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

func (l *CalcPlanTaskDateLogic) ConvertToRRuleOption(planRule *trigger.PbPlanRule, startTime, endTime *carbon.Carbon) (rrule.ROption, error) {
	// 设置默认的rrule选项
	opts := rrule.ROption{
		Freq:     rrule.Frequency(planRule.Freq),
		Dtstart:  startTime.StdTime(),
		Until:    endTime.StdTime(),
		Bysecond: []int{0}, // 默认秒为0
	}

	// 设置小时
	if len(planRule.Hours) > 0 {
		byhour := make([]int, len(planRule.Hours))
		for i, h := range planRule.Hours {
			byhour[i] = int(h)
		}
		opts.Byhour = byhour
	}

	// 设置分钟
	if len(planRule.Minutes) > 0 {
		byminute := make([]int, len(planRule.Minutes))
		for i, m := range planRule.Minutes {
			byminute[i] = int(m)
		}
		opts.Byminute = byminute
	}

	// 设置月份
	if len(planRule.Month) > 0 {
		bymonth := make([]int, len(planRule.Month))
		for i, m := range planRule.Month {
			bymonth[i] = int(m)
		}
		opts.Bymonth = bymonth
	}

	// 设置月中的天数
	if len(planRule.Day) > 0 {
		bymonthday := make([]int, len(planRule.Day))
		for i, d := range planRule.Day {
			bymonthday[i] = int(d)
		}
		opts.Bymonthday = bymonthday
	}

	// 设置星期几
	if len(planRule.Week) > 0 {
		byweekday := make([]rrule.Weekday, len(planRule.Week))
		for i, w := range planRule.Week {
			switch w {
			case 1:
				byweekday[i] = rrule.MO
			case 2:
				byweekday[i] = rrule.TU
			case 3:
				byweekday[i] = rrule.WE
			case 4:
				byweekday[i] = rrule.TH
			case 5:
				byweekday[i] = rrule.FR
			case 6:
				byweekday[i] = rrule.SA
			case 7:
				byweekday[i] = rrule.SU
			default:
				return rrule.ROption{}, fmt.Errorf("invalid week day: %d", w)
			}
		}
		opts.Byweekday = byweekday
	}

	return opts, nil
}
