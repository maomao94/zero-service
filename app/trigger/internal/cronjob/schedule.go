package cronjob

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"zero-service/app/trigger/trigger"
	"zero-service/common/crontask"

	"github.com/dromara/carbon/v2"
	"github.com/teambition/rrule-go"
	"google.golang.org/protobuf/encoding/protojson"
)

const dateTimeLayout = "2006-01-02 15:04:05"
const maxPlanScheduleYears = 3
const maxCronJobScheduleYears = 100

// Schedule 是 Trigger 业务规则编译后的调度配置。
type Schedule struct {
	// RRuleStr 是包含 DTSTART、RRULE、RDATE 和 EXDATE 的 RFC 5545 规则集。
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
func CompileSchedule(rule *trigger.PlanRulePb, startText, endText string, excludeDates, specifiedTimes, excludedTimes []string, skipTimeFilter bool, now time.Time) (*Schedule, error) {
	return compileSchedule(rule, startText, endText, excludeDates, specifiedTimes, excludedTimes, skipTimeFilter, now, maxPlanScheduleYears)
}

// CompileCronJobSchedule 编译不预展开执行数据的 CronJob 规则，允许最长 100 年有效期。
func CompileCronJobSchedule(rule *trigger.PlanRulePb, startText, endText string, excludeDates, specifiedTimes, excludedTimes []string, skipTimeFilter bool, now time.Time) (*Schedule, error) {
	return compileSchedule(rule, startText, endText, excludeDates, specifiedTimes, excludedTimes, skipTimeFilter, now, maxCronJobScheduleYears)
}

func compileSchedule(rule *trigger.PlanRulePb, startText, endText string, excludeDates, specifiedTimes, excludedTimes []string, skipTimeFilter bool, now time.Time, maxYears int) (*Schedule, error) {
	if rule == nil {
		return nil, errors.New("计划规则不能为空")
	}
	if err := rule.Validate(); err != nil {
		return nil, fmt.Errorf("计划规则无效: %w", err)
	}
	startTime, endTime, err := normalizeRange(startText, endText, now, maxYears)
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
	parsedSpecifiedTimes, err := parseExactTimes(specifiedTimes, startTime, endTime, "指定执行时间")
	if err != nil {
		return nil, err
	}
	for _, specifiedTime := range parsedSpecifiedTimes {
		set.RDate(specifiedTime)
	}
	parsedExcludedTimes, err := parseExactTimes(excludedTimes, startTime, endTime, "精确排除时间")
	if err != nil {
		return nil, err
	}
	for _, excludedTime := range parsedExcludedTimes {
		set.ExDate(excludedTime)
	}
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
		for _, specifiedTime := range parsedSpecifiedTimes {
			if specifiedTime.Format("2006-01-02") == value {
				set.ExDate(specifiedTime)
			}
		}
	}

	current := carbon.CreateFromStdTime(now, carbon.Shanghai).StartOfSecond()
	querySet := set
	if shifted := crontask.ShiftSetForQuery(set, current.StdTime()); shifted != nil {
		querySet = shifted
	}
	nextRun := querySet.After(current.StdTime(), true)
	if skipTimeFilter {
		if previous := set.Before(current.StdTime(), true); !previous.IsZero() {
			nextRun = previous
		}
	}
	ruleJSON, err := protojson.Marshal(rule)
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

func parseExactTimes(values []string, startTime, endTime time.Time, fieldName string) ([]time.Time, error) {
	if len(values) == 0 {
		return nil, nil
	}
	location, err := time.LoadLocation(carbon.Shanghai)
	if err != nil {
		return nil, fmt.Errorf("加载时区失败: %w", err)
	}
	result := make([]time.Time, 0, len(values))
	for _, value := range values {
		if len(value) != len(dateTimeLayout) {
			return nil, fmt.Errorf("%s格式错误 %q: 必须为 yyyy-MM-dd HH:mm:ss", fieldName, value)
		}
		exactTime, err := time.ParseInLocation(dateTimeLayout, value, location)
		if err != nil {
			return nil, fmt.Errorf("%s格式错误 %q: %w", fieldName, value, err)
		}
		if exactTime.Format(dateTimeLayout) != value {
			return nil, fmt.Errorf("%s格式错误 %q: 必须为 yyyy-MM-dd HH:mm:ss", fieldName, value)
		}
		exactTime = exactTime.Truncate(time.Second)
		if exactTime.Before(startTime) || exactTime.After(endTime) {
			return nil, fmt.Errorf("%s超出计划时间范围 %q", fieldName, value)
		}
		result = append(result, exactTime)
	}
	return result, nil
}

// ConvertToRRuleOption 将 PlanRulePb 映射为 rrule.ROption。
func ConvertToRRuleOption(planRule *trigger.PlanRulePb, startTime, endTime time.Time) (rrule.ROption, error) {
	opts := rrule.ROption{
		Freq:     rrule.Frequency(planRule.Freq),
		Dtstart:  startTime,
		Until:    endTime,
		Bysecond: []int{0},
	}
	// INTERVAL 仅在大于 1 时显式设置：缺省 0 由 rrule-go 归一化为 1，
	// 且序列化时不会输出 INTERVAL 段，保证存量规则的 rrule_str 不变。
	if planRule.Interval > 1 {
		opts.Interval = int(planRule.Interval)
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

func normalizeRange(startText, endText string, now time.Time, maxYears int) (time.Time, time.Time, error) {
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
	if end.Gt(start.AddYears(maxYears)) {
		return time.Time{}, time.Time{}, fmt.Errorf("计划时间跨度不能超过 %d 年", maxYears)
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
