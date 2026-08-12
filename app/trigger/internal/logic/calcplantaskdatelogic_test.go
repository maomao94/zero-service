package logic

import (
	"context"
	"strings"
	"testing"
	"time"

	"zero-service/app/trigger/internal/cronjob"
	"zero-service/app/trigger/trigger"
	"zero-service/common/crontask"
)

func TestCalcPlanTaskDateReturnsScheduleDescription(t *testing.T) {
	result, err := NewCalcPlanTaskDateLogic(context.Background(), nil).CalcPlanTaskDate(&trigger.CalcPlanTaskDateReq{
		StartTime: "2026-07-27 00:00:00",
		EndTime:   "2026-07-29 23:59:59",
		Rule: &trigger.PlanRulePb{
			Freq:    3,
			Hours:   []int32{9},
			Minutes: []int32{30},
		},
		ExcludeDates: []string{"2026-07-28"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.PlanDates) != 2 {
		t.Fatalf("plan dates = %v, want two dates after exclusion", result.PlanDates)
	}
	for _, want := range []string{"DTSTART;TZID=Asia/Shanghai:", "RRULE:FREQ=DAILY", "EXDATE;TZID=Asia/Shanghai:"} {
		if !strings.Contains(result.RruleStr, want) {
			t.Fatalf("rruleStr = %q, want substring %q", result.RruleStr, want)
		}
	}
	for _, want := range []string{
		"每天 09:30 执行",
		"有效期：2026-07-27 00:00:00 至 2026-07-29 23:59:59",
		"从周期规则与额外候选的合并结果中排除：2026-07-28 09:30:00",
	} {
		if !strings.Contains(result.ScheduleDescription, want) {
			t.Fatalf("schedule description = %q, want substring %q", result.ScheduleDescription, want)
		}
	}
}

func TestCalcPlanTaskDateUsesPersistedScheduleCompiler(t *testing.T) {
	request := &trigger.CalcPlanTaskDateReq{
		StartTime: "2026-07-27 00:00:00",
		EndTime:   "2026-07-29 23:59:59",
		Rule: &trigger.PlanRulePb{
			Freq:    3,
			Hours:   []int32{9},
			Minutes: []int32{30},
		},
		ExcludeDates: []string{"2026-07-28"},
	}
	schedule, err := cronjob.CompileSchedule(request.Rule, request.StartTime, request.EndTime, request.ExcludeDates, request.SpecifiedTimes, request.ExcludedTimes, false, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	wantDescription, err := crontask.DescribeRRule(schedule.RRuleStr)
	if err != nil {
		t.Fatal(err)
	}

	result, err := NewCalcPlanTaskDateLogic(context.Background(), nil).CalcPlanTaskDate(request)
	if err != nil {
		t.Fatal(err)
	}
	if result.ScheduleDescription != wantDescription {
		t.Fatalf("description = %q, want compiler description %q", result.ScheduleDescription, wantDescription)
	}
	if result.RruleStr != schedule.RRuleStr {
		t.Fatalf("rruleStr = %q, want compiler output %q", result.RruleStr, schedule.RRuleStr)
	}
}

func TestCalcPlanTaskDateAppliesExactTimeSetSemantics(t *testing.T) {
	request := &trigger.CalcPlanTaskDateReq{
		StartTime: "2099-07-01 00:00:00",
		EndTime:   "2099-07-04 23:59:59",
		Rule: &trigger.PlanRulePb{
			Freq:    3,
			Hours:   []int32{9},
			Minutes: []int32{0},
		},
		ExcludeDates: []string{"2099-07-03"},
		SpecifiedTimes: []string{
			"2099-07-01 09:00:00", // Duplicate of the RRULE occurrence.
			"2099-07-01 09:00:00", // Duplicate RDATE input.
			"2099-07-02 12:34:56", // Precisely excluded below.
			"2099-07-03 12:34:56", // Excluded by the whole-day exclusion.
			"2099-07-04 12:34:56",
		},
		ExcludedTimes: []string{"2099-07-02 12:34:56"},
	}

	result, err := NewCalcPlanTaskDateLogic(context.Background(), nil).CalcPlanTaskDate(request)
	if err != nil {
		t.Fatal(err)
	}
	wantDates := []string{
		"2099-07-01 09:00:00",
		"2099-07-02 09:00:00",
		"2099-07-04 09:00:00",
		"2099-07-04 12:34:56",
	}
	if len(result.PlanDates) != len(wantDates) {
		t.Fatalf("plan dates = %v, want %v", result.PlanDates, wantDates)
	}
	for i := range wantDates {
		if result.PlanDates[i] != wantDates[i] {
			t.Fatalf("plan date[%d] = %q, want %q", i, result.PlanDates[i], wantDates[i])
		}
	}
	for _, want := range []string{"RDATE;TZID=Asia/Shanghai:", "EXDATE;TZID=Asia/Shanghai:"} {
		if !strings.Contains(result.RruleStr, want) {
			t.Fatalf("rruleStr = %q, want substring %q", result.RruleStr, want)
		}
	}
	for _, want := range []string{"额外纳入候选", "2099-07-04 12:34:56", "2099-07-02 12:34:56", "2099-07-03 12:34:56"} {
		if !strings.Contains(result.ScheduleDescription, want) {
			t.Fatalf("schedule description = %q, want substring %q", result.ScheduleDescription, want)
		}
	}

	schedule, err := cronjob.CompileSchedule(request.Rule, request.StartTime, request.EndTime, request.ExcludeDates, request.SpecifiedTimes, request.ExcludedTimes, false, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	wantDescription, err := crontask.DescribeRRule(schedule.RRuleStr)
	if err != nil {
		t.Fatal(err)
	}
	if result.RruleStr != schedule.RRuleStr || result.ScheduleDescription != wantDescription {
		t.Fatal("plan dates, description, and rruleStr must derive from the same compiled set")
	}
}
