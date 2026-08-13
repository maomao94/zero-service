package cronjob

import (
	"strings"
	"testing"
	"time"

	"zero-service/app/trigger/trigger"
	"zero-service/common/rrulex"

	"github.com/dromara/carbon/v2"
	"github.com/teambition/rrule-go"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestExactTimeRequestValidationAndJSONNames(t *testing.T) {
	rule := &trigger.PlanRulePb{Freq: 3, Hours: []int32{9}, Minutes: []int32{0}}
	validTimes := make([]string, 1000)
	for i := range validTimes {
		validTimes[i] = "2026-07-01 09:00:00"
	}

	request := &trigger.CalcPlanTaskDateReq{Rule: rule}
	if err := request.Validate(); err != nil {
		t.Fatalf("empty exact-time lists should be valid: %v", err)
	}
	request.SpecifiedTimes = validTimes
	request.ExcludedTimes = validTimes
	if err := request.Validate(); err != nil {
		t.Fatalf("1000 exact times should be valid: %v", err)
	}
	request.SpecifiedTimes = append(append([]string(nil), validTimes...), "2026-07-02 09:00:00")
	if err := request.Validate(); err == nil {
		t.Fatal("1001 specified times should be rejected")
	}
	request.SpecifiedTimes = validTimes
	request.ExcludedTimes = append(append([]string(nil), validTimes...), "2026-07-02 09:00:00")
	if err := request.Validate(); err == nil {
		t.Fatal("1001 excluded times should be rejected")
	}
	request.ExcludedTimes = []string{"2026-07-01 09:00"}
	if err := request.Validate(); err == nil {
		t.Fatal("an exact time whose length is not 19 should be rejected")
	}

	request.SpecifiedTimes = []string{"2026-07-01 09:00:00"}
	request.ExcludedTimes = []string{"2026-07-02 10:00:00"}
	encoded, err := protojson.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(encoded); !strings.Contains(got, `"specifiedTimes"`) || !strings.Contains(got, `"excludedTimes"`) {
		t.Fatalf("proto JSON must use exact-time camelCase names: %s", got)
	}
}

func TestCompileScheduleNextRunAndExcludeDate(t *testing.T) {
	location, err := time.LoadLocation(carbon.Shanghai)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 24, 10, 30, 0, 0, location)
	rule := &trigger.PlanRulePb{Freq: 3, Hours: []int32{11}, Minutes: []int32{0}}

	schedule, err := CompileSchedule(rule, "2026-07-01 00:00:00", "2026-07-31 23:59:59", []string{"2026-07-24"}, nil, nil, false, now)
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
	if err := rrulex.Validate(schedule.RRuleStr); err != nil {
		t.Fatalf("serialized RRULE cannot be parsed: %v", err)
	}
	if !strings.Contains(schedule.RRuleStr, "DTSTART;TZID=Asia/Shanghai:") ||
		!strings.Contains(schedule.RRuleStr, "EXDATE;TZID=Asia/Shanghai:") {
		t.Fatalf("serialized RRULE must pin DTSTART and EXDATE to Asia/Shanghai: %q", schedule.RRuleStr)
	}
	set, err := rrule.StrToRRuleSet(schedule.RRuleStr)
	if err != nil {
		t.Fatal(err)
	}
	if got := set.GetDTStart().Location().String(); got != carbon.Shanghai {
		t.Fatalf("DTSTART location = %q, want %q", got, carbon.Shanghai)
	}
	for _, date := range set.GetExDate() {
		if got := date.Location().String(); got != carbon.Shanghai {
			t.Fatalf("EXDATE location = %q, want %q", got, carbon.Shanghai)
		}
	}
}

func TestCompileScheduleSkipTimeFilterTriggersOnePastOccurrence(t *testing.T) {
	location, err := time.LoadLocation(carbon.Shanghai)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 24, 10, 30, 0, 0, location)
	rule := &trigger.PlanRulePb{Freq: 3, Hours: []int32{11}, Minutes: []int32{0}}

	schedule, err := CompileSchedule(rule, "2026-07-01 00:00:00", "2026-07-31 23:59:59", nil, nil, nil, true, now)
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

	schedule, err := CompileSchedule(rule, "2026-07-01 00:00:00", "2026-07-10 23:59:59", nil, nil, nil, false, now)
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

	schedule, err := CompileSchedule(rule, "", "", nil, nil, nil, false, now)
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

	schedule, err := CompileSchedule(rule, "2026-07-01 00:00:00", "2026-07-31 23:59:59", nil, nil, nil, false, now)
	if err != nil {
		t.Fatal(err)
	}
	if !schedule.NextRun.Equal(now) {
		t.Fatalf("next run = %v, want current occurrence %v", schedule.NextRun, now)
	}
}

func TestCronJobScheduleAllowsOneHundredYearsButPlanDoesNot(t *testing.T) {
	location, err := time.LoadLocation(carbon.Shanghai)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 24, 10, 30, 0, 0, location)
	rule := &trigger.PlanRulePb{Freq: 0, Month: []int32{1}, Day: []int32{1}, Hours: []int32{9}, Minutes: []int32{0}}

	if _, err := CompileSchedule(rule, "2026-01-01 00:00:00", "2126-01-01 00:00:00", nil, nil, nil, false, now); err == nil {
		t.Fatal("plan schedule should keep the three-year limit")
	}
	schedule, err := CompileCronJobSchedule(rule, "2026-01-01 00:00:00", "2126-01-01 00:00:00", nil, nil, nil, false, now)
	if err != nil {
		t.Fatalf("cron job schedule should allow a 100-year range: %v", err)
	}
	if schedule.EndTime.Year() != 2126 {
		t.Fatalf("cron job end year = %d, want 2126", schedule.EndTime.Year())
	}
	if _, err := CompileCronJobSchedule(rule, "2026-01-01 00:00:00", "2126-01-01 00:00:01", nil, nil, nil, false, now); err == nil {
		t.Fatal("cron job schedule should reject ranges longer than 100 years")
	}
}

func TestCompileScheduleExactTimesAcceptRangeBoundaries(t *testing.T) {
	location, err := time.LoadLocation(carbon.Shanghai)
	if err != nil {
		t.Fatal(err)
	}
	rule := &trigger.PlanRulePb{Freq: 3, Hours: []int32{9}, Minutes: []int32{0}}
	schedule, err := CompileSchedule(
		rule,
		"2026-07-01 00:00:00",
		"2026-07-31 23:59:59",
		nil,
		[]string{"2026-07-01 00:00:00", "2026-07-31 23:59:59"},
		[]string{"2026-07-01 00:00:00", "2026-07-31 23:59:59"},
		false,
		time.Date(2026, 6, 30, 0, 0, 0, 987654321, location),
	)
	if err != nil {
		t.Fatal(err)
	}
	set, err := rrule.StrToRRuleSet(schedule.RRuleStr)
	if err != nil {
		t.Fatal(err)
	}
	want := []time.Time{
		time.Date(2026, 7, 1, 0, 0, 0, 0, location),
		time.Date(2026, 7, 31, 23, 59, 59, 0, location),
	}
	if got := set.GetRDate(); len(got) != len(want) || !got[0].Equal(want[0]) || !got[1].Equal(want[1]) {
		t.Fatalf("RDATE = %v, want %v", got, want)
	}
	if got := set.GetExDate(); len(got) != len(want) || !got[0].Equal(want[0]) || !got[1].Equal(want[1]) {
		t.Fatalf("EXDATE = %v, want %v", got, want)
	}
	if !strings.Contains(schedule.RRuleStr, "RDATE;TZID=Asia/Shanghai:") {
		t.Fatalf("serialized RRULE must pin RDATE to Asia/Shanghai: %q", schedule.RRuleStr)
	}
}

func TestCompileScheduleExactTimesRejectInvalidValues(t *testing.T) {
	location, err := time.LoadLocation(carbon.Shanghai)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, location)
	rule := &trigger.PlanRulePb{Freq: 3, Hours: []int32{9}, Minutes: []int32{0}}
	tests := []struct {
		name           string
		specifiedTimes []string
		excludedTimes  []string
	}{
		{name: "specified before range", specifiedTimes: []string{"2026-06-30 23:59:59"}},
		{name: "specified after range", specifiedTimes: []string{"2026-08-01 00:00:00"}},
		{name: "excluded before range", excludedTimes: []string{"2026-06-30 23:59:59"}},
		{name: "excluded after range", excludedTimes: []string{"2026-08-01 00:00:00"}},
		{name: "specified malformed", specifiedTimes: []string{"2026-07-01T12:00:00"}},
		{name: "specified fractional second", specifiedTimes: []string{"2026-07-01 12:00:00.1"}},
		{name: "excluded invalid date", excludedTimes: []string{"2026-07-32 12:00:00"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := CompileSchedule(
				rule,
				"2026-07-01 00:00:00",
				"2026-07-31 23:59:59",
				nil,
				test.specifiedTimes,
				test.excludedTimes,
				false,
				now,
			); err == nil {
				t.Fatal("expected exact time validation error")
			}
		})
	}
}

func TestCompileScheduleExactTimeSetSemantics(t *testing.T) {
	location, err := time.LoadLocation(carbon.Shanghai)
	if err != nil {
		t.Fatal(err)
	}
	rule := &trigger.PlanRulePb{Freq: 3, Hours: []int32{9}, Minutes: []int32{0}}
	schedule, err := CompileSchedule(
		rule,
		"2026-07-01 00:00:00",
		"2026-07-04 23:59:59",
		[]string{"2026-07-03"},
		[]string{
			"2026-07-01 09:00:00", // Duplicate of the RRULE occurrence.
			"2026-07-01 09:00:00", // Duplicate RDATE input.
			"2026-07-02 12:34:56", // Precisely excluded below.
			"2026-07-03 12:34:56", // Excluded by the whole-day exclusion.
			"2026-07-04 12:34:56",
		},
		[]string{"2026-07-02 12:34:56"},
		false,
		time.Date(2026, 6, 30, 0, 0, 0, 0, location),
	)
	if err != nil {
		t.Fatal(err)
	}
	set, err := rrule.StrToRRuleSet(schedule.RRuleStr)
	if err != nil {
		t.Fatal(err)
	}
	got := set.All()
	want := []time.Time{
		time.Date(2026, 7, 1, 9, 0, 0, 0, location),
		time.Date(2026, 7, 2, 9, 0, 0, 0, location),
		time.Date(2026, 7, 4, 9, 0, 0, 0, location),
		time.Date(2026, 7, 4, 12, 34, 56, 0, location),
	}
	if len(got) != len(want) {
		t.Fatalf("occurrences = %v, want %v", got, want)
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Fatalf("occurrence[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestCompileScheduleIntervalPeriodStep(t *testing.T) {
	location, err := time.LoadLocation(carbon.Shanghai)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 3, 20, 10, 30, 0, 0, location)
	rule := &trigger.PlanRulePb{
		Freq: 1, Day: []int32{5}, Hours: []int32{9}, Minutes: []int32{0}, Interval: 3,
	}

	schedule, err := CompileSchedule(rule, "2026-01-05 00:00:00", "2026-12-31 23:59:59", nil, nil, nil, false, now)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(schedule.RRuleStr, "INTERVAL=3") {
		t.Fatalf("serialized RRULE must contain INTERVAL=3: %q", schedule.RRuleStr)
	}
	want := time.Date(2026, 4, 5, 9, 0, 0, 0, location)
	if !schedule.NextRun.Equal(want) {
		t.Fatalf("next run = %v, want %v", schedule.NextRun, want)
	}

	set, err := rrule.StrToRRuleSet(schedule.RRuleStr)
	if err != nil {
		t.Fatal(err)
	}
	got := set.Between(
		time.Date(2026, 1, 1, 0, 0, 0, 0, location),
		time.Date(2026, 12, 31, 23, 59, 59, 0, location),
		true,
	)
	wantDates := []time.Time{
		time.Date(2026, 1, 5, 9, 0, 0, 0, location),
		time.Date(2026, 4, 5, 9, 0, 0, 0, location),
		time.Date(2026, 7, 5, 9, 0, 0, 0, location),
		time.Date(2026, 10, 5, 9, 0, 0, 0, location),
	}
	if len(got) != len(wantDates) {
		t.Fatalf("occurrences = %v, want %v", got, wantDates)
	}
	for i := range wantDates {
		if !got[i].Equal(wantDates[i]) {
			t.Fatalf("occurrence[%d] = %v, want %v", i, got[i], wantDates[i])
		}
	}

	description, err := rrulex.Describe(schedule.RRuleStr)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(description, "按 3 个月间隔") {
		t.Fatalf("description must mention 3-month interval: %q", description)
	}
}

func TestCompileScheduleIntervalDefaultsToOne(t *testing.T) {
	location, err := time.LoadLocation(carbon.Shanghai)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, location)
	// interval=0（老客户端缺省）必须归一化为 1，行为与未传一致。
	rule := &trigger.PlanRulePb{Freq: 3, Hours: []int32{9}, Minutes: []int32{0}}
	schedule, err := CompileSchedule(rule, "2026-07-01 00:00:00", "2026-07-31 23:59:59", nil, nil, nil, false, now)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(schedule.RRuleStr, "INTERVAL=") {
		t.Fatalf("default interval must not be serialized: %q", schedule.RRuleStr)
	}
	want := time.Date(2026, 7, 1, 9, 0, 0, 0, location)
	if !schedule.NextRun.Equal(want) {
		t.Fatalf("next run = %v, want %v", schedule.NextRun, want)
	}

	// rule JSON 往返必须保留 interval。
	ruleWithInterval := &trigger.PlanRulePb{Freq: 1, Day: []int32{5}, Hours: []int32{9}, Minutes: []int32{0}, Interval: 3}
	encoded, err := protojson.Marshal(ruleWithInterval)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"interval":3`) {
		t.Fatalf("rule JSON must carry interval: %s", encoded)
	}
	var decoded trigger.PlanRulePb
	if err := protojson.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Interval != 3 {
		t.Fatalf("round-trip interval = %d, want 3", decoded.Interval)
	}
}
