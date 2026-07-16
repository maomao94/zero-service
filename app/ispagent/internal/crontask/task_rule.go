package crontask

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"zero-service/common/crontask"

	"github.com/dromara/carbon/v2"
	"github.com/teambition/rrule-go"
)

// 巡视类型常量（与 ISP 协议 type 字段对齐）。
type PatrolType string

const (
	PatrolRoutine  PatrolType = "1" // 例行巡视
	PatrolSpecial  PatrolType = "2" // 特殊巡视
	PatrolSpecific PatrolType = "3" // 专项巡视
	PatrolCustom   PatrolType = "4" // 自定义巡视
)

// 间隔类型常量（与 ISP 协议 interval_type 字段对齐）。
type IntervalType string

const (
	IntervalHour IntervalType = "1" // 小时
	IntervalDay  IntervalType = "2" // 天
)

// ISP 星期数字 → rrule 星期常量 (1=MO, 2=TU, ..., 7=SU)。
var ispWeekdayToRRule = []rrule.Weekday{
	rrule.MO, rrule.TU, rrule.WE, rrule.TH, rrule.FR, rrule.SA, rrule.SU,
}

// IspTaskFields ISP 任务扩展字段，存储在 TaskConfig.Extra 中。
type IspTaskFields struct {
	SubstationCode      string `json:"substation_code"`
	PatrolType          string `json:"patrol_type"`
	TaskCode            string `json:"task_code"`
	TaskName            string `json:"task_name"`
	Priority            string `json:"priority"`
	DeviceLevel         int    `json:"device_level"`
	DeviceList          string `json:"device_list"`
	FixedStartTime      string `json:"fixed_start_time"`
	CycleMonth          string `json:"cycle_month"`
	CycleWeek           string `json:"cycle_week"`
	CycleExecuteTime    string `json:"cycle_execute_time"`
	CycleStartTime      string `json:"cycle_start_time"`
	CycleEndTime        string `json:"cycle_end_time"`
	IntervalNumber      string `json:"interval_number"`
	IntervalType        string `json:"interval_type"`
	IntervalExecuteTime string `json:"interval_execute_time"`
	IntervalStartTime   string `json:"interval_start_time"`
	IntervalEndTime     string `json:"interval_end_time"`
	InvalidStartTime    string `json:"invalid_start_time"`
	InvalidEndTime      string `json:"invalid_end_time"`
	IsEnable            string `json:"isenable"`
	Creator             string `json:"creator"`
	CreateTime          string `json:"create_time"`
}

// TaskType 根据 ISP 协议指令 a/b/c/d 优先级判断任务类型。
func (f *IspTaskFields) TaskType() string {
	if f.FixedStartTime != "" {
		return "fixed"
	}
	if f.CycleMonth != "" && f.CycleWeek != "" && f.CycleExecuteTime != "" {
		return "cycle"
	}
	if f.IntervalNumber != "" && f.IntervalType != "" && f.IntervalExecuteTime != "" {
		return "interval"
	}
	return ""
}

// ToPriority 将 ISP 优先级字符串转为 int（1-4，默认 1）。
func (f *IspTaskFields) ToPriority() int {
	p, err := strconv.Atoi(f.Priority)
	if err != nil || p < 1 || p > 4 {
		return 1
	}
	return p
}

// ToStatus 将 ISP isenable 转为 crontask.TaskStatus。
func (f *IspTaskFields) ToStatus() crontask.TaskStatus {
	switch f.IsEnable {
	case "0":
		return crontask.StatusEnabled
	default:
		return crontask.StatusDisabled
	}
}

// ToRRuleStr 根据 ISP 任务类型生成对应的 rrule 字符串（用于持久化）。
func (f *IspTaskFields) ToRRuleStr() string {
	switch f.TaskType() {
	case "fixed":
		return buildFixedRRule(f)
	case "cycle":
		return buildCycleRRule(f)
	case "interval":
		return buildIntervalRRule(f)
	default:
		return ""
	}
}

// toROption 返回 ISP 任务对应的 ROption。CalcInitNextRun 使用此方法避免字符串往返丢失时区。
func (f *IspTaskFields) toROption() *rrule.ROption {
	switch f.TaskType() {
	case "fixed":
		return buildFixedROption(f)
	case "cycle":
		return buildCycleROption(f)
	case "interval":
		return buildIntervalROption(f)
	}
	return nil
}

// CalcInitNextRun 根据 ISP 协议规则计算首次调度时间。
// 使用 ROption 直传避免 ROption.String() → StrToRRule 往返丢失 Dtstart 时区。
func (f *IspTaskFields) CalcInitNextRun() (time.Time, error) {
	opt := f.toROption()
	if opt == nil {
		return time.Time{}, fmt.Errorf("unknown task type")
	}
	rule, err := rrule.NewRRule(*opt)
	if err != nil {
		return time.Time{}, err
	}
	next := rule.After(carbon.Now().StdTime(), false)
	return f.skipInvalidTime(rule, next), nil
}

// buildFixedRRule 为定期任务生成单次执行的 rrule。
// 使用 FREQ=DAILY;COUNT=1 保证触发一次后自然终止。
func buildFixedRRule(f *IspTaskFields) string {
	opt := buildFixedROption(f)
	if opt == nil {
		return ""
	}
	return opt.String()
}

func buildFixedROption(f *IspTaskFields) *rrule.ROption {
	t := parseTime(f.FixedStartTime)
	opt := &rrule.ROption{
		Freq:  rrule.DAILY,
		Count: 1,
	}
	if !t.IsZero() {
		opt.Dtstart = t
	}
	return opt
}

// buildCycleRRule 为周期任务生成 rrule 字符串。
func buildCycleRRule(f *IspTaskFields) string {
	opt := buildCycleROption(f)
	if opt == nil {
		return ""
	}
	return opt.String()
}

// buildIntervalRRule 为间隔任务生成 rrule。
func buildIntervalRRule(f *IspTaskFields) string {
	opt := buildIntervalROption(f)
	if opt == nil {
		return ""
	}
	return opt.String()
}

// skipInvalidTime 跳过不可用时间范围内的触发点。
// 循环调用 rule.After() 直到找到第一个不在不可用区间内的时间。
func (f *IspTaskFields) skipInvalidTime(rule *rrule.RRule, next time.Time) time.Time {
	is := parseTime(f.InvalidStartTime)
	ie := parseTime(f.InvalidEndTime)
	if is.IsZero() || ie.IsZero() {
		return next
	}
	for !next.IsZero() && !next.Before(is) && !next.After(ie) {
		next = rule.After(next, false)
	}
	return next
}

// buildCycleROption 构建周期任务的 ROption。
func buildCycleROption(f *IspTaskFields) *rrule.ROption {
	var byweekday []rrule.Weekday
	for _, w := range splitCSV(f.CycleWeek) {
		if idx, err := strconv.Atoi(w); err == nil && idx >= 1 && idx <= 7 {
			byweekday = append(byweekday, ispWeekdayToRRule[idx-1])
		}
	}
	if len(byweekday) == 0 {
		return nil
	}

	opt := &rrule.ROption{
		Freq:      rrule.WEEKLY,
		Byweekday: byweekday,
	}
	for _, m := range splitCSV(f.CycleMonth) {
		if n, err := strconv.Atoi(m); err == nil && n >= 1 && n <= 12 {
			opt.Bymonth = append(opt.Bymonth, n)
		}
	}
	if t := f.CycleExecuteTime; len(t) >= 5 && t[2] == ':' {
		h, _ := strconv.Atoi(t[:2])
		m, _ := strconv.Atoi(t[3:5])
		opt.Byhour = []int{h}
		opt.Byminute = []int{m}
	}
	if st := parseTime(f.CycleStartTime); !st.IsZero() {
		opt.Dtstart = st
	}
	if et := parseTime(f.CycleEndTime); !et.IsZero() {
		opt.Until = et
	}
	return opt
}

// buildIntervalROption 构建间隔任务的 ROption。
// 按 ISP 协议规则计算首次执行时间 T0 作为 Dtstart：
//
//	a) T0 = interval_start_time 日期 + interval_execute_time HHmmss
//	b) 若 T0 < interval_start_time，则 T0 + 1 天
//
// 后续执行时间由 FREQ + INTERVAL 自然递增，Until 约束结束时间。
func buildIntervalROption(f *IspTaskFields) *rrule.ROption {
	opt := &rrule.ROption{}
	switch f.IntervalType {
	case string(IntervalHour):
		opt.Freq = rrule.HOURLY
	case string(IntervalDay):
		opt.Freq = rrule.DAILY
	default:
		return nil
	}
	if n, _ := strconv.Atoi(f.IntervalNumber); n > 0 {
		opt.Interval = n
	}

	// T0 = interval_start_time 日期 + interval_execute_time 时间
	startT := parseTime(f.IntervalStartTime)
	if !startT.IsZero() && f.IntervalExecuteTime != "" {
		dateStr := carbon.CreateFromStdTime(startT).ToDateString()          // "2026-07-09"
		t0 := carbon.Parse(dateStr + " " + f.IntervalExecuteTime).StdTime() // "2026-07-09 08:59:00"
		if t0.Before(startT) {
			t0 = t0.Add(24 * time.Hour)
		}
		opt.Dtstart = t0
	}
	if et := parseTime(f.IntervalEndTime); !et.IsZero() {
		opt.Until = et
	}
	return opt
}

// parseTime 使用 carbon 解析时间字符串，支持 "yyyy-MM-dd HH:mm:ss" 格式。
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if c := carbon.Parse(s); c.Error == nil {
		return c.StdTime()
	}
	return time.Time{}
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// SerializeExtra 将 IspTaskFields 序列化为 JSON 字符串。
func SerializeExtra(fields *IspTaskFields) string {
	if fields == nil {
		return ""
	}
	data, _ := json.Marshal(fields)
	return string(data)
}

// DeserializeExtra 从 JSON 字符串反序列化 IspTaskFields，失败时返回 nil。
func DeserializeExtra(extra string) *IspTaskFields {
	if extra == "" {
		return nil
	}
	var f IspTaskFields
	if err := json.Unmarshal([]byte(extra), &f); err != nil {
		return nil
	}
	return &f
}

// NewTaskConfig 从 ISP 字段创建 crontask.TaskConfig。
// RRuleStr 和 NextRun 每次从 ISP 字段重新计算，保证配置更新后使用最新值。
func NewTaskConfig(existingID string, fields *IspTaskFields) *crontask.TaskConfig {
	nextRun, err := fields.CalcInitNextRun()
	status := fields.ToStatus()
	if err != nil || nextRun.IsZero() {
		nextRun = carbon.Now().AddYears(100).StdTime()
	}

	return &crontask.TaskConfig{
		ID:       existingID,
		TaskCode: fields.TaskCode,
		TaskName: fields.TaskName,
		RRuleStr: fields.ToRRuleStr(),
		Priority: fields.ToPriority(),
		Status:   status,
		NextRun:  nextRun,
		Extra:    json.RawMessage(SerializeExtra(fields)),
	}
}

// NewInvalidTimeFilter 创建 crontask 的 InvalidTimeFilter，复用 CalcInitNextRun 的 skipInvalidTime 逻辑。
func NewInvalidTimeFilter() crontask.InvalidTimeFilter {
	return func(task *crontask.TaskConfig, next time.Time) time.Time {
		fields := DeserializeExtra(string(task.Extra))
		if fields == nil {
			return next
		}
		is := parseTime(fields.InvalidStartTime)
		ie := parseTime(fields.InvalidEndTime)
		if is.IsZero() || ie.IsZero() {
			return next
		}
		rule, err := rrule.StrToRRule(task.RRuleStr)
		if err != nil {
			return next
		}
		next = fields.skipInvalidTime(rule, next)
		if next.IsZero() {
			return carbon.Now().AddYears(100).StdTime()
		}
		return next
	}
}
