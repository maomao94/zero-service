package cronjob

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"zero-service/app/trigger/trigger"

	"github.com/dromara/carbon/v2"
	"github.com/teambition/rrule-go"
)

const dateTimeLayout = "2006-01-02 15:04:05"

// Schedule 是 Trigger 业务规则编译后的调度配置。
type Schedule struct {
	// RRuleStr 是包含 DTSTART、RRULE 和 EXDATE 的 RFC 5545 规则集。
	RRuleStr string
	// StartTime 是补齐默认值后参与 RRULE 编译的生效开始时间。
	StartTime time.Time
	// EndTime 是补齐默认值后参与 RRULE 编译的生效结束时间。
	EndTime time.Time
	// NextRun 是首次执行时间，零值表示规则已经耗尽。
	NextRun time.Time
	// RuleJSON 是创建请求中的 PlanRulePb JSON。
	RuleJSON json.RawMessage
}

// CompileSchedule 将 Trigger 业务规则编译为 crontask 可直接消费的 RRULE set。
// skipTimeFilter 仅影响首次 NextRun：允许时最多选择一个已发生计划用于立即补触发。
func CompileSchedule(rule *trigger.PlanRulePb, startText, endText string, excludeDates []string, skipTimeFilter bool, now time.Time) (*Schedule, error) {
	if rule == nil {
		return nil, errors.New("计划规则不能为空")
	}
	if err := rule.Validate(); err != nil {
		return nil, fmt.Errorf("计划规则无效: %w", err)
	}
	startTime, endTime, err := normalizeRange(startText, endText, now)
	if err != nil {
		return nil, err
	}
	opts, err := ConvertToRRuleOption(rule, startTime, endTime)
	if err != nil {
		return nil, err
	}
	r, err := rrule.NewRRule(opts)
	if err != nil {
		return nil, fmt.Errorf("生成 RRULE 失败: %w", err)
	}
	set := &rrule.Set{}
	set.RRule(r)
	for _, value := range excludeDates {
		exclude := carbon.ParseByFormat(value, carbon.DateFormat, carbon.Shanghai)
		if exclude.Error != nil || exclude.IsInvalid() {
			return nil, fmt.Errorf("排除日期格式错误 %q: %w", value, exclude.Error)
		}
		for _, hour := range rule.Hours {
			for _, minute := range rule.Minutes {
				excludeTime := exclude.Copy().SetHour(int(hour)).SetMinute(int(minute)).SetSecond(0).StartOfSecond()
				set.ExDate(excludeTime.StdTime())
			}
		}
	}

	current := carbon.CreateFromStdTime(now, carbon.Shanghai).StartOfSecond()
	nextRun := set.After(current.StdTime(), true)
	if skipTimeFilter {
		if previous := set.Before(current.StdTime(), true); !previous.IsZero() {
			nextRun = previous
		}
	}
	ruleJSON, err := json.Marshal(rule)
	if err != nil {
		return nil, fmt.Errorf("序列化计划规则失败: %w", err)
	}
	return &Schedule{
		RRuleStr:  set.String(),
		StartTime: startTime,
		EndTime:   endTime,
		NextRun:   nextRun,
		RuleJSON:  ruleJSON,
	}, nil
}

// ConvertToRRuleOption 将 PlanRulePb 映射为 rrule.ROption。
func ConvertToRRuleOption(planRule *trigger.PlanRulePb, startTime, endTime time.Time) (rrule.ROption, error) {
	opts := rrule.ROption{
		Freq:     rrule.Frequency(planRule.Freq),
		Dtstart:  startTime,
		Until:    endTime,
		Bysecond: []int{0},
	}
	opts.Byhour = int32sToInts(planRule.Hours)
	opts.Byminute = int32sToInts(planRule.Minutes)
	opts.Bymonth = int32sToInts(planRule.Month)
	opts.Bymonthday = int32sToInts(planRule.Day)
	if len(planRule.Week) > 0 {
		opts.Byweekday = make([]rrule.Weekday, len(planRule.Week))
		for i, week := range planRule.Week {
			switch week {
			case 1:
				opts.Byweekday[i] = rrule.MO
			case 2:
				opts.Byweekday[i] = rrule.TU
			case 3:
				opts.Byweekday[i] = rrule.WE
			case 4:
				opts.Byweekday[i] = rrule.TH
			case 5:
				opts.Byweekday[i] = rrule.FR
			case 6:
				opts.Byweekday[i] = rrule.SA
			case 7:
				opts.Byweekday[i] = rrule.SU
			default:
				return rrule.ROption{}, fmt.Errorf("星期参数不合法: %d", week)
			}
		}
	}
	return opts, nil
}

func normalizeRange(startText, endText string, now time.Time) (time.Time, time.Time, error) {
	current := carbon.CreateFromStdTime(now, carbon.Shanghai).StartOfSecond()
	var start *carbon.Carbon
	if startText == "" {
		start = current.StartOfYear()
	} else {
		start = carbon.ParseByLayout(startText, carbon.DateTimeLayout, carbon.Shanghai)
		if start.Error != nil || start.IsInvalid() {
			return time.Time{}, time.Time{}, fmt.Errorf("开始时间格式错误: %w", start.Error)
		}
	}
	var end *carbon.Carbon
	if endText == "" {
		end = start.EndOfYear().StartOfSecond()
	} else {
		end = carbon.ParseByLayout(endText, carbon.DateTimeLayout, carbon.Shanghai)
		if end.Error != nil || end.IsInvalid() {
			return time.Time{}, time.Time{}, fmt.Errorf("结束时间格式错误: %w", end.Error)
		}
	}
	if end.Lt(start) {
		return time.Time{}, time.Time{}, errors.New("结束时间必须晚于开始时间")
	}
	if end.Gt(start.AddYears(3)) {
		return time.Time{}, time.Time{}, errors.New("计划时间跨度不能超过 3 年")
	}
	return start.StdTime(), end.StdTime(), nil
}

func int32sToInts(values []int32) []int {
	if len(values) == 0 {
		return nil
	}
	result := make([]int, len(values))
	for i, value := range values {
		result[i] = int(value)
	}
	return result
}
