package crontask

import (
	"fmt"
	"testing"
	"time"

	"zero-service/common/crontask"

	"github.com/dromara/carbon/v2"
	"github.com/teambition/rrule-go"
)

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

	rule, err := rrule.StrToRRule(rruleStr)
	if err != nil {
		t.Fatalf("parse rrule failed: %v, str=%s", err, rruleStr)
	}
	if rule.OrigOptions.Freq != rrule.DAILY {
		t.Fatalf("expected DAILY, got %v", rule.OrigOptions.Freq)
	}
	if rule.OrigOptions.Count != 1 {
		t.Fatalf("expected COUNT=1, got %d", rule.OrigOptions.Count)
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

	rule, err := rrule.StrToRRule(rruleStr)
	if err != nil {
		t.Fatalf("parse rrule failed: %v, str=%s", err, rruleStr)
	}
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
	}
	rruleStr := buildCycleRRule(f)
	rule, err := rrule.StrToRRule(rruleStr)
	if err != nil {
		t.Fatalf("parse rrule failed: %v", err)
	}
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
	}
	rruleStr := buildIntervalRRule(f)
	rule, err := rrule.StrToRRule(rruleStr)
	if err != nil {
		t.Fatalf("parse rrule failed: %v", err)
	}
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
	}
	rruleStr := buildIntervalRRule(f)
	rule, err := rrule.StrToRRule(rruleStr)
	if err != nil {
		t.Fatalf("parse rrule failed: %v", err)
	}
	if rule.OrigOptions.Freq != rrule.DAILY {
		t.Fatalf("expected DAILY, got %v", rule.OrigOptions.Freq)
	}
	if rule.OrigOptions.Interval != 3 {
		t.Fatalf("expected INTERVAL=3, got %d", rule.OrigOptions.Interval)
	}
}

func TestToRRuleStrDispatch(t *testing.T) {
	fixed := &IspTaskFields{FixedStartTime: "2025-07-09 00:00:00"}
	cycle := &IspTaskFields{CycleMonth: "2", CycleWeek: "1", CycleExecuteTime: "20:00:00"}
	interval := &IspTaskFields{IntervalNumber: "10", IntervalType: "1", IntervalExecuteTime: "08:59:00"}

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
	now := carbon.Now()
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
	now := carbon.Now()
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
	rule, _ := rrule.StrToRRule(rruleStr)

	base := carbon.Parse("2025-07-09 00:00:00").StdTime()
	first := rule.After(base.Add(-time.Second), false)
	if first.IsZero() {
		t.Fatal("expected first occurrence")
	}

	second := rule.After(first, false)
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
	rule, err := rrule.StrToRRule(rruleStr)
	if err != nil {
		t.Fatalf("parse rrule: %v", err)
	}

	base := now.StartOfDay().StdTime()
	next := rule.After(base, false)
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

func TestConvertRoundTrip(t *testing.T) {
	f := &IspTaskFields{
		PatrolType:  "1",
		TaskCode:    "SIP-test-001",
		TaskName:    "测试",
		Priority:    "2",
		DeviceLevel: 3,
		DeviceList:  "1000526",
		IsEnable:    "0",
		Creator:     "zxhc",
		CreateTime:  "2026-07-09 08:58:07",
	}
	extra := SerializeExtra(f)

	nextRun, err := f.CalcInitNextRun()
	if err != nil {
		nextRun = carbon.Now().AddYears(100).StdTime()
	}

	cfg := &crontask.TaskConfig{
		TaskCode: f.TaskCode,
		TaskName: f.TaskName,
		RRuleStr: f.ToRRuleStr(),
		Priority: f.ToPriority(),
		Status:   f.ToStatus(),
		NextRun:  nextRun,
		Extra:    []byte(extra),
		Version:  1,
	}

	gorm := fromTaskConfig(cfg)
	back := toTaskConfig(gorm)

	if back.TaskCode != cfg.TaskCode {
		t.Fatal("round-trip task_code mismatch")
	}
	if back.RRuleStr != cfg.RRuleStr {
		t.Fatalf("round-trip rrule mismatch: %s vs %s", back.RRuleStr, cfg.RRuleStr)
	}
	if back.Priority != cfg.Priority {
		t.Fatal("round-trip priority mismatch")
	}
	if back.Status != cfg.Status {
		t.Fatal("round-trip status mismatch")
	}

	parsed := DeserializeExtra(string(back.Extra))
	if parsed.Creator != f.Creator {
		t.Fatal("round-trip creator mismatch")
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
	rule, err := rrule.StrToRRule(rruleStr)
	if err != nil {
		t.Fatalf("parse rrule: %v", err)
	}

	base := now.StartOfDay().StdTime()
	first := rule.After(base, false)

	// should be today 12:00 (within invalid range)
	expectedToday := carbon.Parse(today + " 12:00:00").StdTime()
	if !first.Equal(expectedToday) {
		t.Fatalf("rrule should give today 12:00, got %v", first)
	}

	// skipInvalidTime should skip to next week
	skipped := f.skipInvalidTime(rule, first)
	if !skipped.After(parseTime(tomorrow + " 23:59:59")) {
		t.Fatalf("expected skip to next week, got %v", skipped)
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
	rule, _ := rrule.StrToRRule(rruleStr)

	base := now.StartOfDay().StdTime()
	first := rule.After(base, false)

	skipped := f.skipInvalidTime(rule, first)
	if !skipped.Equal(first) {
		t.Fatal("expected no skip when no invalid time")
	}
}

func TestCycleExecuteTimeShortStringNoPanic(t *testing.T) {
	f := &IspTaskFields{
		CycleMonth:       "2",
		CycleWeek:        "1",
		CycleExecuteTime: "08",
	}
	s := buildCycleRRule(f)
	if s == "" {
		t.Fatal("expected rrule string for cycle task with short execute time")
	}
	// Should parse without panic: BYHOUR/BYMINUTE simply not set
	rule, err := rrule.StrToRRule(s)
	if err != nil {
		t.Fatalf("parse rrule: %v", err)
	}
	if len(rule.OrigOptions.Byhour) != 0 {
		t.Fatal("expected no BYHOUR for short execute time")
	}
}

func TestToFieldsRoundTripPriority(t *testing.T) {
	g := &GormTaskConfig{
		TaskCode:   "test-code",
		TaskName:   "test-name",
		Priority:   2,
		IsEnable:   "0",
		IspCreator: "creator-1",
	}
	f := g.toFields()
	if f.Priority != "2" {
		t.Fatalf("expected Priority='2', got '%s'", f.Priority)
	}
	if f.Creator != g.IspCreator {
		t.Fatalf("expected Creator='%s', got '%s'", g.IspCreator, f.Creator)
	}

	// round-trip: from fields back to GormTaskConfig via applyFields
	g2 := &GormTaskConfig{}
	g2.applyFields(f)
	if g2.IspCreator != g.IspCreator {
		t.Fatalf("applyFields round-trip creator mismatch")
	}
}
