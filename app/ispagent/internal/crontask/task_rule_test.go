package crontask

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"zero-service/common/crontask"
	"zero-service/common/rrulex"

	"github.com/dromara/carbon/v2"
	"github.com/teambition/rrule-go"
)

// mustNextAfter 用官方原生风格（ParseSet + set.After）查询单次 after 候选。
func mustNextAfter(t *testing.T, value string, after time.Time) time.Time {
	t.Helper()
	set, err := rrulex.ParseSet(value)
	if err != nil {
		t.Fatalf("parse RRULE Set: %v", err)
	}
	return set.After(after, false)
}

func mustParseSetRule(t *testing.T, value string) *rrule.RRule {
	t.Helper()
	set, err := rrule.StrToRRuleSet(value)
	if err != nil {
		t.Fatalf("parse RRULE Set failed: %v, value=%q", err, value)
	}
	if set.GetDTStart().IsZero() || set.GetRRule() == nil {
		t.Fatalf("incomplete RRULE Set: %q", value)
	}
	return set.GetRRule()
}

func requireShanghaiDTStart(t *testing.T, value string) {
	t.Helper()
	if !strings.Contains(value, "DTSTART;TZID=Asia/Shanghai:") {
		t.Fatalf("RRULE Set must pin DTSTART to Asia/Shanghai: %q", value)
	}
	if got := mustParseSetRule(t, value).OrigOptions.Dtstart.Location().String(); got != carbon.Shanghai {
		t.Fatalf("DTSTART location = %q, want %q", got, carbon.Shanghai)
	}
}

func TestTaskTypeDetection(t *testing.T) {
	tests := []struct {
		name string
		f    *IspTaskFields
		want string
	}{
		{"fixed", &IspTaskFields{FixedStartTime: "2025-07-09 00:00:00"}, "fixed"},
		{"cycle", &IspTaskFields{CycleMonth: "2", CycleWeek: "1", CycleExecuteTime: "20:00:00"}, "cycle"},
		{"interval", &IspTaskFields{IntervalNumber: "10", IntervalType: "1", IntervalExecuteTime: "08:59:00"}, "interval"},
		{"empty", &IspTaskFields{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.TaskType(); got != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}

func TestBuildFixedRRule(t *testing.T) {
	f := &IspTaskFields{FixedStartTime: "2025-07-09 00:00:00"}
	rruleStr := buildFixedRRule(f)
	requireShanghaiDTStart(t, rruleStr)

	rule := mustParseSetRule(t, rruleStr)
	if rule.OrigOptions.Freq != rrule.DAILY {
		t.Fatalf("expected DAILY, got %v", rule.OrigOptions.Freq)
	}
	if rule.OrigOptions.Count != 1 {
		t.Fatalf("expected COUNT=1, got %d", rule.OrigOptions.Count)
	}
	if got := rule.OrigOptions.Dtstart.Location().String(); got != carbon.Shanghai {
		t.Fatalf("DTSTART location = %q, want %q", got, carbon.Shanghai)
	}
}

func TestBuildCycleRRule(t *testing.T) {
	f := &IspTaskFields{
		CycleMonth:       "2",
		CycleWeek:        "1",
		CycleExecuteTime: "20:00:00",
		CycleStartTime:   "2026-01-01 00:00:00",
		CycleEndTime:     "2026-12-31 23:59:59",
	}
	rruleStr := buildCycleRRule(f)
	requireShanghaiDTStart(t, rruleStr)

	rule := mustParseSetRule(t, rruleStr)
	if rule.OrigOptions.Freq != rrule.WEEKLY {
		t.Fatalf("expected WEEKLY, got %v", rule.OrigOptions.Freq)
	}
	if len(rule.OrigOptions.Bymonth) != 1 || rule.OrigOptions.Bymonth[0] != 2 {
		t.Fatalf("expected BYMONTH=[2], got %v", rule.OrigOptions.Bymonth)
	}
	if len(rule.OrigOptions.Byweekday) != 1 || rule.OrigOptions.Byweekday[0] != rrule.MO {
		t.Fatalf("expected BYDAY=[MO], got %v", rule.OrigOptions.Byweekday)
	}
	if len(rule.OrigOptions.Byhour) != 1 || rule.OrigOptions.Byhour[0] != 20 {
		t.Fatalf("expected BYHOUR=[20], got %v", rule.OrigOptions.Byhour)
	}
}

func TestBuildCycleRRuleMultiple(t *testing.T) {
	f := &IspTaskFields{
		CycleMonth:       "1,2,5",
		CycleWeek:        "1,7",
		CycleExecuteTime: "08:00:00",
		CycleStartTime:   "2026-01-01 00:00:00",
	}
	rruleStr := buildCycleRRule(f)
	rule := mustParseSetRule(t, rruleStr)
	if len(rule.OrigOptions.Bymonth) != 3 {
		t.Fatalf("expected 3 months, got %d", len(rule.OrigOptions.Bymonth))
	}
	if len(rule.OrigOptions.Byweekday) != 2 {
		t.Fatalf("expected 2 weekdays, got %d", len(rule.OrigOptions.Byweekday))
	}
	if rule.OrigOptions.Byweekday[0] != rrule.MO || rule.OrigOptions.Byweekday[1] != rrule.SU {
		t.Fatalf("expected [MO, SU], got %v", rule.OrigOptions.Byweekday)
	}
}

func TestBuildIntervalRRuleHourly(t *testing.T) {
	f := &IspTaskFields{
		IntervalNumber:      "10",
		IntervalType:        string(IntervalHour),
		IntervalExecuteTime: "08:59:00",
		IntervalStartTime:   "2026-01-01 00:00:00",
	}
	rruleStr := buildIntervalRRule(f)
	requireShanghaiDTStart(t, rruleStr)
	rule := mustParseSetRule(t, rruleStr)
	if rule.OrigOptions.Freq != rrule.HOURLY {
		t.Fatalf("expected HOURLY, got %v", rule.OrigOptions.Freq)
	}
	if rule.OrigOptions.Interval != 10 {
		t.Fatalf("expected INTERVAL=10, got %d", rule.OrigOptions.Interval)
	}
}

func TestBuildIntervalRRuleDaily(t *testing.T) {
	f := &IspTaskFields{
		IntervalNumber:      "3",
		IntervalType:        string(IntervalDay),
		IntervalExecuteTime: "12:30:00",
		IntervalStartTime:   "2026-01-01 00:00:00",
	}
	rruleStr := buildIntervalRRule(f)
	requireShanghaiDTStart(t, rruleStr)
	rule := mustParseSetRule(t, rruleStr)
	if rule.OrigOptions.Freq != rrule.DAILY {
		t.Fatalf("expected DAILY, got %v", rule.OrigOptions.Freq)
	}
	if rule.OrigOptions.Interval != 3 {
		t.Fatalf("expected INTERVAL=3, got %d", rule.OrigOptions.Interval)
	}
}

func TestToRRuleStrDispatch(t *testing.T) {
	fixed := &IspTaskFields{FixedStartTime: "2025-07-09 00:00:00"}
	cycle := &IspTaskFields{CycleMonth: "2", CycleWeek: "1", CycleExecuteTime: "20:00:00", CycleStartTime: "2026-01-01 00:00:00"}
	interval := &IspTaskFields{IntervalNumber: "10", IntervalType: "1", IntervalExecuteTime: "08:59:00", IntervalStartTime: "2026-01-01 00:00:00"}

	if s := fixed.ToRRuleStr(); s == "" {
		t.Fatal("fixed task should have rrule")
	}
	if s := cycle.ToRRuleStr(); s == "" {
		t.Fatal("cycle task should have rrule")
	}
	if s := interval.ToRRuleStr(); s == "" {
		t.Fatal("interval task should have rrule")
	}
}

func TestToPriority(t *testing.T) {
	tests := []struct {
		priority string
		want     int
	}{
		{"1", 1}, {"2", 2}, {"3", 3}, {"4", 4}, {"", 1}, {"5", 1}, {"abc", 1},
	}
	for _, tt := range tests {
		f := &IspTaskFields{Priority: tt.priority}
		if got := f.ToPriority(); got != tt.want {
			t.Fatalf("priority=%s: expected %d, got %d", tt.priority, tt.want, got)
		}
	}
}

func TestToStatus(t *testing.T) {
	f0 := &IspTaskFields{IsEnable: "0"}
	if s := f0.ToStatus(); s != crontask.StatusEnabled {
		t.Fatalf("expected enabled, got %v", s)
	}
	f1 := &IspTaskFields{IsEnable: "1"}
	if s := f1.ToStatus(); s != crontask.StatusDisabled {
		t.Fatalf("expected disabled, got %v", s)
	}
}

func TestCalcInitNextRunCycle(t *testing.T) {
	now := carbon.Now(carbon.Shanghai)
	nextYear := now.AddYear().Year()

	f := &IspTaskFields{
		CycleMonth:       fmt.Sprintf("%d", int(now.Month())),
		CycleWeek:        fmt.Sprintf("%d", int(now.DayOfWeek())),
		CycleExecuteTime: "12:00:00",
		CycleStartTime:   now.StartOfDay().ToDateTimeString(),
		CycleEndTime:     fmt.Sprintf("%d-12-31 23:59:59", nextYear),
	}

	next, err := f.CalcInitNextRun()
	if err != nil {
		t.Fatalf("CalcInitNextRun error: %v", err)
	}
	if next.IsZero() {
		t.Fatal("expected non-zero next run for cycle task")
	}
	if next.Hour() != 12 || next.Minute() != 0 {
		t.Fatalf("expected 12:00, got %02d:%02d", next.Hour(), next.Minute())
	}
}

func TestCalcInitNextRunInterval(t *testing.T) {
	now := carbon.Now(carbon.Shanghai)
	start := now.StartOfDay().AddHours(8)
	end := now.StartOfDay().AddDay()

	f := &IspTaskFields{
		IntervalNumber:      "1",
		IntervalType:        string(IntervalHour),
		IntervalExecuteTime: fmt.Sprintf("%02d:00:00", start.Hour()),
		IntervalStartTime:   start.ToDateTimeString(),
		IntervalEndTime:     end.ToDateTimeString(),
	}

	next, err := f.CalcInitNextRun()
	if err != nil {
		t.Fatalf("CalcInitNextRun error: %v", err)
	}
	if next.IsZero() {
		t.Fatal("expected non-zero next run for interval task")
	}
	if next.Before(start.StdTime()) {
		t.Fatalf("expected >= start time, got %v", next)
	}
}

func TestFixedRRuleFiresOnce(t *testing.T) {
	f := &IspTaskFields{FixedStartTime: "2025-07-09 00:00:00"}
	rruleStr := f.ToRRuleStr()

	base := carbon.Parse("2025-07-09 00:00:00", carbon.Shanghai).StdTime()
	first := mustNextAfter(t, rruleStr, base.Add(-time.Second))
	if first.IsZero() {
		t.Fatal("expected first occurrence")
	}

	second := mustNextAfter(t, rruleStr, first)
	if !second.IsZero() {
		t.Fatal("expected no second occurrence after COUNT=1")
	}
}

func TestCycleRRuleNextOccurrence(t *testing.T) {
	now := carbon.Now()
	f := &IspTaskFields{
		CycleMonth:       fmt.Sprintf("%d", int(now.Month())),
		CycleWeek:        fmt.Sprintf("%d", int(now.DayOfWeek())),
		CycleExecuteTime: "20:00:00",
		CycleStartTime:   now.StartOfDay().ToDateTimeString(),
		CycleEndTime:     now.AddDay().EndOfDay().ToDateTimeString(),
	}

	rruleStr := f.ToRRuleStr()

	base := now.StartOfDay().StdTime()
	next := mustNextAfter(t, rruleStr, base)
	if next.IsZero() {
		t.Fatal("expected next occurrence")
	}
	if next.Hour() != 20 || next.Minute() != 0 {
		t.Fatalf("expected 20:00, got %02d:%02d", next.Hour(), next.Minute())
	}
}

func TestSerializeDeserializeExtra(t *testing.T) {
	f := &IspTaskFields{
		PatrolType:  "1",
		TaskCode:    "SIP25070409502403",
		TaskName:    "测试任务",
		Priority:    "1",
		DeviceLevel: 3,
		DeviceList:  "1000526,1001323",
		IsEnable:    "0",
		Creator:     "zxhc",
		CreateTime:  "2026-07-09 08:58:07",
	}

	data := SerializeExtra(f)
	parsed := DeserializeExtra(data)

	if parsed.TaskCode != f.TaskCode {
		t.Fatal("task_code mismatch")
	}
	if parsed.TaskName != f.TaskName {
		t.Fatal("task_name mismatch")
	}
	if parsed.Creator != f.Creator {
		t.Fatal("creator mismatch")
	}
	if parsed.CreateTime != f.CreateTime {
		t.Fatal("create_time mismatch")
	}
}

func TestInvalidTimeSkip(t *testing.T) {
	now := carbon.Now()
	today := now.ToDateString()
	tomorrow := now.AddDay().ToDateString()
	nextYear := now.AddYear().Year()

	f := &IspTaskFields{
		CycleMonth:       fmt.Sprintf("%d", int(now.Month())),
		CycleWeek:        fmt.Sprintf("%d", int(now.DayOfWeek())),
		CycleExecuteTime: "12:00:00",
		CycleStartTime:   now.StartOfDay().ToDateTimeString(),
		CycleEndTime:     fmt.Sprintf("%d-12-31 23:59:59", nextYear),
		InvalidStartTime: today + " 00:00:00",
		InvalidEndTime:   tomorrow + " 23:59:59",
	}

	rruleStr := f.ToRRuleStr()
	base := now.StartOfDay().StdTime()
	first := mustNextAfter(t, rruleStr, base)

	// should be today 12:00 (within invalid range)
	expectedToday := carbon.Parse(today+" 12:00:00", carbon.Shanghai).StdTime()
	if !first.Equal(expectedToday) {
		t.Fatalf("rrule should give today 12:00, got %v", first)
	}

	// 谓词排除无效区间，NextRuns 单趟推进到区间后首个有效点（下周）。
	runs, err := rrulex.NextRuns(rruleStr, base, false, 1, f.invalidTimePredicate())
	if err != nil {
		t.Fatalf("NextRuns: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 valid run, got %v", runs)
	}
	if !runs[0].After(parseTime(tomorrow + " 23:59:59")) {
		t.Fatalf("expected skip to next week, got %v", runs[0])
	}
}

func TestNoInvalidTimeNoSkip(t *testing.T) {
	now := carbon.Now()
	nextYear := now.AddYear().Year()

	f := &IspTaskFields{
		CycleMonth:       fmt.Sprintf("%d", int(now.Month())),
		CycleWeek:        fmt.Sprintf("%d", int(now.DayOfWeek())),
		CycleExecuteTime: "12:00:00",
		CycleStartTime:   now.StartOfDay().ToDateTimeString(),
		CycleEndTime:     fmt.Sprintf("%d-12-31 23:59:59", nextYear),
	}

	rruleStr := f.ToRRuleStr()
	base := now.StartOfDay().StdTime()
	first := mustNextAfter(t, rruleStr, base)

	// 无无效窗口：谓词为 nil，返回原候选。
	runs, err := rrulex.NextRuns(rruleStr, base, false, 1, f.invalidTimePredicate())
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || !runs[0].Equal(first) {
		t.Fatalf("expected no skip when no invalid time, got %v", runs)
	}
}

func TestCycleExecuteTimeShortStringNoPanic(t *testing.T) {
	f := &IspTaskFields{
		CycleMonth:       "2",
		CycleWeek:        "1",
		CycleExecuteTime: "08",
		CycleStartTime:   "2026-01-01 00:00:00",
	}
	s := buildCycleRRule(f)
	if s == "" {
		t.Fatal("expected rrule string for cycle task with short execute time")
	}
	// Should parse without panic: BYHOUR/BYMINUTE simply not set
	rule := mustParseSetRule(t, s)
	if len(rule.OrigOptions.Byhour) != 0 {
		t.Fatal("expected no BYHOUR for short execute time")
	}
}

func TestNewTaskConfigExpiredScheduleHasZeroNextRun(t *testing.T) {
	fields := &IspTaskFields{
		TaskCode:       "expired",
		TaskName:       "已结束任务",
		FixedStartTime: carbon.Now().SubHour().ToDateTimeString(),
		IsEnable:       "0",
	}

	cfg, err := NewTaskConfig(nil, fields)
	if err != nil {
		t.Fatalf("NewTaskConfig: %v", err)
	}
	if !cfg.NextRun.IsZero() {
		t.Fatalf("expected zero next run, got %v", cfg.NextRun)
	}
	if cfg.Status != crontask.StatusEnabled {
		t.Fatalf("expected enabled status to be preserved, got %v", cfg.Status)
	}
}

func TestNewTaskConfigReturnsScheduleError(t *testing.T) {
	_, err := NewTaskConfig(nil, &IspTaskFields{TaskCode: "invalid"})
	if err == nil {
		t.Fatal("expected invalid task type error")
	}
}

func TestInvalidTimePredicateReturnsEmptyWhenRuleExhausted(t *testing.T) {
	runAt := carbon.Now(carbon.Shanghai).AddHour().StartOfSecond()
	fields := &IspTaskFields{
		FixedStartTime:   runAt.ToDateTimeString(),
		InvalidStartTime: runAt.SubMinute().ToDateTimeString(),
		InvalidEndTime:   runAt.AddMinute().ToDateTimeString(),
	}
	task := &crontask.TaskConfig{
		RRuleStr: fields.ToRRuleStr(),
		Extra:    json.RawMessage(SerializeExtra(fields)),
	}

	pred := NewInvalidTimePredicate()
	if !pred(task, runAt.StdTime()) {
		t.Fatalf("expected candidate inside invalid window to be excluded")
	}

	// COUNT=1 且唯一候选落在无效区间内：推进耗尽返回空结果。
	runs, err := rrulex.NextRuns(task.RRuleStr, runAt.StdTime().Add(-time.Hour), false, 1, func(t time.Time) bool {
		return pred(task, t)
	})
	if err != nil {
		t.Fatalf("NextRuns: %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("expected empty runs when rule exhausted inside invalid window, got %v", runs)
	}
}

func TestNewInvalidTimePredicateNoWindow(t *testing.T) {
	pred := NewInvalidTimePredicate()
	now := time.Now()
	if pred(&crontask.TaskConfig{}, now) {
		t.Fatal("empty extra must not be filtered")
	}
	fields := &IspTaskFields{TaskCode: "no-window"}
	task := &crontask.TaskConfig{Extra: json.RawMessage(SerializeExtra(fields))}
	if pred(task, now) {
		t.Fatal("task without invalid window must not be filtered")
	}
}

func TestNewInvalidTimePredicateWindowBoundaries(t *testing.T) {
	start := parseTime("2026-08-13 10:00:00")
	end := parseTime("2026-08-13 12:00:00")
	tests := []struct {
		name string
		at   time.Time
		want bool
	}{
		{name: "before", at: start.Add(-time.Second), want: false},
		{name: "exact-start", at: start, want: true},
		{name: "interior", at: start.Add(time.Hour), want: true},
		{name: "exact-end", at: end, want: true},
		{name: "after", at: end.Add(time.Second), want: false},
	}
	task := &crontask.TaskConfig{Extra: json.RawMessage(SerializeExtra(&IspTaskFields{
		InvalidStartTime: "2026-08-13 10:00:00",
		InvalidEndTime:   "2026-08-13 12:00:00",
	}))}
	pred := NewInvalidTimePredicate()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pred(task, tt.at); got != tt.want {
				t.Fatalf("predicate(%v) = %v, want %v", tt.at, got, tt.want)
			}
		})
	}
}

func TestNewInvalidTimePredicateIgnoresIncompleteOrMalformedWindow(t *testing.T) {
	tests := []struct {
		name  string
		start string
		end   string
		extra json.RawMessage
	}{
		{name: "missing-start", end: "2026-08-13 12:00:00"},
		{name: "missing-end", start: "2026-08-13 10:00:00"},
		{name: "malformed-start", start: "not-a-time", end: "2026-08-13 12:00:00"},
		{name: "malformed-end", start: "2026-08-13 10:00:00", end: "not-a-time"},
		{name: "malformed-extra", extra: json.RawMessage("{")},
	}
	pred := NewInvalidTimePredicate()
	at := parseTime("2026-08-13 11:00:00")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extra := tt.extra
			if extra == nil {
				extra = json.RawMessage(SerializeExtra(&IspTaskFields{
					InvalidStartTime: tt.start,
					InvalidEndTime:   tt.end,
				}))
			}
			if pred(&crontask.TaskConfig{Extra: extra}, at) {
				t.Fatal("incomplete or malformed invalid window must not filter candidates")
			}
		})
	}
}

func TestNewInvalidTimePredicateConcurrent(t *testing.T) {
	runAt := carbon.Now(carbon.Shanghai).AddHour().StartOfSecond()
	fields := &IspTaskFields{
		FixedStartTime:   runAt.ToDateTimeString(),
		InvalidStartTime: runAt.SubMinute().ToDateTimeString(),
		InvalidEndTime:   runAt.AddMinute().ToDateTimeString(),
	}
	task := &crontask.TaskConfig{Extra: json.RawMessage(SerializeExtra(fields))}
	pred := NewInvalidTimePredicate()

	// 谓词无状态，并发调用结果一致（-race 兜底）。
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			for range 100 {
				if !pred(task, runAt.StdTime()) {
					t.Errorf("expected true for candidate inside window")
					return
				}
				if pred(task, runAt.AddDay().StdTime()) {
					t.Errorf("expected false for candidate outside window")
					return
				}
			}
		})
	}
	wg.Wait()
}
