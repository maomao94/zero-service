package cronjob

import (
	"testing"
	"time"

	"zero-service/app/trigger/trigger"
	"zero-service/common/crontask"

	"github.com/dromara/carbon/v2"
)

func TestCompileScheduleNextRunAndExcludeDate(t *testing.T) {
	location, err := time.LoadLocation(carbon.Shanghai)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 24, 10, 30, 0, 0, location)
	rule := &trigger.PlanRulePb{Freq: 3, Hours: []int32{11}, Minutes: []int32{0}}

	schedule, err := CompileSchedule(rule, "2026-07-01 00:00:00", "2026-07-31 23:59:59", []string{"2026-07-24"}, false, now)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 7, 25, 11, 0, 0, 0, location)
	if !schedule.NextRun.Equal(want) {
		t.Fatalf("next run = %v, want %v", schedule.NextRun, want)
	}
	if schedule.RRuleStr == "" {
		t.Fatal("expected serialized RRULE set")
	}
	if err := crontask.ValidateRRule(schedule.RRuleStr); err != nil {
		t.Fatalf("serialized RRULE cannot be parsed: %v", err)
	}
}

func TestCompileScheduleSkipTimeFilterTriggersOnePastOccurrence(t *testing.T) {
	location, err := time.LoadLocation(carbon.Shanghai)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 24, 10, 30, 0, 0, location)
	rule := &trigger.PlanRulePb{Freq: 3, Hours: []int32{11}, Minutes: []int32{0}}

	schedule, err := CompileSchedule(rule, "2026-07-01 00:00:00", "2026-07-31 23:59:59", nil, true, now)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 7, 23, 11, 0, 0, 0, location)
	if !schedule.NextRun.Equal(want) {
		t.Fatalf("next run = %v, want previous occurrence %v", schedule.NextRun, want)
	}
}

func TestCompileScheduleExhaustedReturnsZero(t *testing.T) {
	location, err := time.LoadLocation(carbon.Shanghai)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 24, 10, 30, 0, 0, location)
	rule := &trigger.PlanRulePb{Freq: 3, Hours: []int32{11}, Minutes: []int32{0}}

	schedule, err := CompileSchedule(rule, "2026-07-01 00:00:00", "2026-07-10 23:59:59", nil, false, now)
	if err != nil {
		t.Fatal(err)
	}
	if !schedule.NextRun.IsZero() {
		t.Fatalf("expected exhausted schedule, got %v", schedule.NextRun)
	}
}

func TestCompileScheduleDefaultsToCurrentYear(t *testing.T) {
	location, err := time.LoadLocation(carbon.Shanghai)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 24, 10, 30, 0, 0, location)
	rule := &trigger.PlanRulePb{Freq: 3, Hours: []int32{11}, Minutes: []int32{0}}

	schedule, err := CompileSchedule(rule, "", "", nil, false, now)
	if err != nil {
		t.Fatal(err)
	}
	if schedule.StartTime.Year() != 2026 || schedule.EndTime.Year() != 2026 {
		t.Fatalf("default range = %v..%v, want current year", schedule.StartTime, schedule.EndTime)
	}
}

func TestCompileScheduleIncludesOccurrenceAtCurrentSecond(t *testing.T) {
	location, err := time.LoadLocation(carbon.Shanghai)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 24, 11, 0, 0, 0, location)
	rule := &trigger.PlanRulePb{Freq: 3, Hours: []int32{11}, Minutes: []int32{0}}

	schedule, err := CompileSchedule(rule, "2026-07-01 00:00:00", "2026-07-31 23:59:59", nil, false, now)
	if err != nil {
		t.Fatal(err)
	}
	if !schedule.NextRun.Equal(now) {
		t.Fatalf("next run = %v, want current occurrence %v", schedule.NextRun, now)
	}
}
