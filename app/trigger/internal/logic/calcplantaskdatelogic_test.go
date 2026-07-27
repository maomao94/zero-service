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
		"排除执行：2026-07-28 09:30:00",
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
	schedule, err := cronjob.CompileSchedule(request.Rule, request.StartTime, request.EndTime, request.ExcludeDates, false, time.Now())
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
