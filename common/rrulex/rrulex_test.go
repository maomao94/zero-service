package rrulex

import (
	"strings"
	"testing"
	"time"

	"github.com/teambition/rrule-go"
)

func testRRuleSet(rule string) string {
	return "DTSTART:20260727T000000Z\nRRULE:" + rule
}

// refNextAfter 使用官方未平移的 ParseSet + set.After 做查询，作为平移后结果的差分参照。
func refNextAfter(t *testing.T, value string, after time.Time) time.Time {
	t.Helper()
	set, err := ParseSet(value)
	if err != nil {
		t.Fatal(err)
	}
	return set.After(after, false)
}

// refPreviewRuns 是未做起点平移的原始批量实现，作为平移后结果的差分参照。
func refPreviewRuns(t *testing.T, value string, after time.Time, count int) []time.Time {
	t.Helper()
	set, err := ParseSet(value)
	if err != nil {
		t.Fatal(err)
	}
	runs := make([]time.Time, 0, count)
	cursor := after
	for len(runs) < count {
		next := set.After(cursor, false)
		if next.IsZero() {
			break
		}
		runs = append(runs, next)
		cursor = next
	}
	return runs
}

// refNextRuns 使用官方未平移 Set 的单个迭代器，并保持 NextRuns 的谓词与边界判断顺序。
func refNextRuns(t *testing.T, value string, dt time.Time, inc bool, count int, invalid func(time.Time) bool) []time.Time {
	t.Helper()
	set, err := ParseSet(value)
	if err != nil {
		t.Fatal(err)
	}
	runs := make([]time.Time, 0, count)
	next := set.Iterator()
	for len(runs) < count {
		v, ok := next()
		if !ok {
			break
		}
		if invalid != nil && invalid(v) {
			continue
		}
		if inc && !v.Before(dt) || !inc && v.After(dt) {
			runs = append(runs, v)
			dt = v
		}
	}
	return runs
}

func requireSameRuns(t *testing.T, got, want []time.Time) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("runs = %v, want %v", got, want)
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Fatalf("runs[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestParseSetRejectsBareRRule(t *testing.T) {
	if _, err := ParseSet("FREQ=DAILY;BYHOUR=10;BYMINUTE=30;BYSECOND=0"); err == nil {
		t.Fatal("ParseSet must reject bare RRULE")
	}
	if err := Validate("FREQ=DAILY;BYHOUR=10;BYMINUTE=30;BYSECOND=0"); err == nil {
		t.Fatal("Validate must reject bare RRULE")
	}
}

func TestValidateEmptyRuleIsValid(t *testing.T) {
	// 空字符串表示一次性任务，是合法配置。
	if err := Validate(""); err != nil {
		t.Fatalf("Validate(empty) = %v, want nil", err)
	}
}

func TestParseSetSupportsCRLFRRuleSet(t *testing.T) {
	value := "DTSTART;TZID=Asia/Shanghai:20260727T090000\r\n" +
		"RRULE:FREQ=DAILY;COUNT=2\r\n" +
		"EXDATE;TZID=Asia/Shanghai:20260728T090000"
	after := time.Date(2026, 7, 27, 0, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	set, err := ParseSet(value)
	if err != nil {
		t.Fatal(err)
	}
	next := set.After(after, false)
	want := time.Date(2026, 7, 27, 9, 0, 0, 0, next.Location())
	if !next.Equal(want) {
		t.Fatalf("next = %v, want %v", next, want)
	}
	if err := Validate(value); err != nil {
		t.Fatalf("Validate(CRLF set): %v", err)
	}
}

func TestParseSetDoesNotTryToProveCalendarReachability(t *testing.T) {
	value := "DTSTART:20260201T090000Z\nRRULE:FREQ=MONTHLY;BYMONTH=2;BYMONTHDAY=30;UNTIL=20260301T000000Z"
	if _, err := ParseSet(value); err != nil {
		t.Fatalf("ParseSet must accept structurally valid but unreachable bounded rule: %v", err)
	}
	if err := Validate(value); err != nil {
		t.Fatalf("Validate must not perform unbounded reachability checks: %v", err)
	}
	runs, err := NextRuns(value, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), false, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 0 {
		t.Fatalf("unreachable bounded rule runs = %v, want exhausted", runs)
	}
}

// TestNextRunsMatchesUnshiftedAfter 证明平移后的批量迭代与官方未平移的单次查询结果一致。
func TestNextRunsMatchesUnshiftedAfter(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 8, 12, 15, 17, 54, 0, loc)
	rules := []struct {
		name string
		rule string
	}{
		{name: "daily-fixed-time", rule: "DTSTART;TZID=Asia/Shanghai:20260101T093000\nRRULE:FREQ=DAILY;BYHOUR=9;BYMINUTE=30;BYSECOND=0"},
		{name: "daily-bymonth", rule: "DTSTART;TZID=Asia/Shanghai:20260101T093000\nRRULE:FREQ=DAILY;BYMONTH=3;BYHOUR=9;BYMINUTE=30;BYSECOND=0"},
		{name: "weekly-byday", rule: "DTSTART;TZID=Asia/Shanghai:20260101T090000\nRRULE:FREQ=WEEKLY;BYDAY=MO,WE;BYHOUR=9;BYMINUTE=0;BYSECOND=0"},
		{name: "weekly-default-phase", rule: "DTSTART;TZID=Asia/Shanghai:20260101T090000\nRRULE:FREQ=WEEKLY;BYHOUR=9;BYMINUTE=0;BYSECOND=0"},
		{name: "monthly-day15", rule: "DTSTART;TZID=Asia/Shanghai:20260101T090000\nRRULE:FREQ=MONTHLY;BYMONTHDAY=15;BYHOUR=9;BYMINUTE=0;BYSECOND=0"},
		{name: "monthly-day31", rule: "DTSTART;TZID=Asia/Shanghai:20260101T090000\nRRULE:FREQ=MONTHLY;BYMONTHDAY=31;BYHOUR=9;BYMINUTE=0;BYSECOND=0"},
		{name: "yearly-bymonth", rule: "DTSTART;TZID=Asia/Shanghai:20260101T090000\nRRULE:FREQ=YEARLY;BYMONTH=1;BYMONTHDAY=1;BYHOUR=9;BYMINUTE=0;BYSECOND=0"},
		{name: "hourly-fixed-minute", rule: "DTSTART;TZID=Asia/Shanghai:20260101T093000\nRRULE:FREQ=HOURLY;BYMINUTE=30;BYSECOND=0"},
		{name: "minutely-fixed-second", rule: "DTSTART;TZID=Asia/Shanghai:20260101T093005\nRRULE:FREQ=MINUTELY;BYSECOND=0"},
		{name: "secondly-fixed-minute", rule: "DTSTART;TZID=Asia/Shanghai:20260101T093000\nRRULE:FREQ=SECONDLY;BYMINUTE=5;BYSECOND=0"},
		{name: "monthly-interval2", rule: "DTSTART;TZID=Asia/Shanghai:20260101T093000\nRRULE:FREQ=MONTHLY;INTERVAL=2;BYMONTHDAY=1;BYHOUR=9;BYMINUTE=30;BYSECOND=0"},
		{name: "daily-interval3", rule: "DTSTART;TZID=Asia/Shanghai:20260101T093000\nRRULE:FREQ=DAILY;INTERVAL=3;BYHOUR=9;BYMINUTE=30;BYSECOND=0"},
		{name: "weekly-interval2-byday", rule: "DTSTART;TZID=Asia/Shanghai:20260101T090000\nRRULE:FREQ=WEEKLY;INTERVAL=2;BYDAY=TH;BYHOUR=9;BYMINUTE=0;BYSECOND=0"},
		{name: "hourly-interval2", rule: "DTSTART;TZID=Asia/Shanghai:20260101T093000\nRRULE:FREQ=HOURLY;INTERVAL=2;BYMINUTE=30;BYSECOND=0"},
		{name: "minutely-interval5", rule: "DTSTART;TZID=Asia/Shanghai:20260101T093005\nRRULE:FREQ=MINUTELY;INTERVAL=5;BYSECOND=0"},
		{name: "yearly-interval2", rule: "DTSTART;TZID=Asia/Shanghai:20230101T090000\nRRULE:FREQ=YEARLY;INTERVAL=2;BYMONTH=1;BYMONTHDAY=1;BYHOUR=9;BYMINUTE=0;BYSECOND=0"},
		{name: "rdate-exdate", rule: "DTSTART;TZID=Asia/Shanghai:20260101T093000\nRRULE:FREQ=DAILY;BYHOUR=9;BYMINUTE=30;BYSECOND=0\nRDATE;TZID=Asia/Shanghai:20260115T100000\nEXDATE;TZID=Asia/Shanghai:20260201T093000"},
	}
	queries := []time.Time{now, now.Add(3 * time.Hour), now.Add(30 * 24 * time.Hour)}
	for _, tt := range rules {
		for _, q := range queries {
			want := refNextAfter(t, tt.rule, q)
			runs, err := NextRuns(tt.rule, q, false, 1, nil)
			if err != nil {
				t.Fatalf("%s: NextRuns failed: %v", tt.name, err)
			}
			if want.IsZero() {
				if len(runs) != 0 {
					t.Errorf("%s: NextRuns(%v) = %v, want exhausted", tt.name, q.Format("2006-01-02 15:04:05"), runs)
				}
				continue
			}
			if len(runs) != 1 || !runs[0].Equal(want) {
				t.Errorf("%s: NextRuns(%v) = %v, want %v", tt.name, q.Format("2006-01-02 15:04:05"), runs, want)
			}
		}
	}
}

func TestNextRunsAllFrequenciesMatchUnshiftedSet(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		dtstart time.Time
		after   time.Time
	}{
		{name: "yearly", value: "DTSTART:20200115T091011Z\nRRULE:FREQ=YEARLY;INTERVAL=2", dtstart: time.Date(2020, 1, 15, 9, 10, 11, 0, time.UTC), after: time.Date(2025, 8, 12, 12, 0, 0, 0, time.UTC)},
		{name: "monthly", value: "DTSTART:20240115T091011Z\nRRULE:FREQ=MONTHLY;INTERVAL=3", dtstart: time.Date(2024, 1, 15, 9, 10, 11, 0, time.UTC), after: time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)},
		{name: "weekly", value: "DTSTART:20240101T091011Z\nRRULE:FREQ=WEEKLY;INTERVAL=2;BYDAY=MO,WE", dtstart: time.Date(2024, 1, 1, 9, 10, 11, 0, time.UTC), after: time.Date(2024, 3, 14, 12, 0, 0, 0, time.UTC)},
		{name: "daily", value: "DTSTART:20240101T091011Z\nRRULE:FREQ=DAILY;INTERVAL=3", dtstart: time.Date(2024, 1, 1, 9, 10, 11, 0, time.UTC), after: time.Date(2024, 2, 14, 12, 0, 0, 0, time.UTC)},
		{name: "hourly", value: "DTSTART:20240101T091011Z\nRRULE:FREQ=HOURLY;INTERVAL=5", dtstart: time.Date(2024, 1, 1, 9, 10, 11, 0, time.UTC), after: time.Date(2024, 1, 3, 12, 0, 0, 0, time.UTC)},
		{name: "minutely", value: "DTSTART:20240101T091011Z\nRRULE:FREQ=MINUTELY;INTERVAL=7", dtstart: time.Date(2024, 1, 1, 9, 10, 11, 0, time.UTC), after: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
		{name: "secondly", value: "DTSTART:20240101T091011Z\nRRULE:FREQ=SECONDLY;INTERVAL=11", dtstart: time.Date(2024, 1, 1, 9, 10, 11, 0, time.UTC), after: time.Date(2024, 1, 1, 9, 12, 0, 0, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queries := []struct {
				name string
				dt   time.Time
			}{
				{name: "before-dtstart", dt: tt.dtstart.Add(-time.Second)},
				{name: "at-dtstart", dt: tt.dtstart},
				{name: "after-dtstart", dt: tt.after},
			}
			for _, query := range queries {
				for _, inc := range []bool{false, true} {
					t.Run(query.name+map[bool]string{false: "-exclusive", true: "-inclusive"}[inc], func(t *testing.T) {
						want := refNextRuns(t, tt.value, query.dt, inc, 4, nil)
						got, err := NextRuns(tt.value, query.dt, inc, 4, nil)
						if err != nil {
							t.Fatal(err)
						}
						requireSameRuns(t, got, want)
					})
				}
			}
		})
	}
}

func TestNextRunsSetComponentsAndBoundsMatchUnshiftedSet(t *testing.T) {
	tests := []struct {
		name  string
		value string
		after time.Time
		count int
	}{
		{
			name: "rdate-exdate",
			value: "DTSTART:20260101T090000Z\nRRULE:FREQ=DAILY;INTERVAL=2;UNTIL=20260112T090000Z\n" +
				"RDATE:20260106T120000Z\nEXDATE:20260105T090000Z",
			after: time.Date(2026, 1, 3, 9, 0, 0, 0, time.UTC),
			count: 6,
		},
		{
			name:  "count-fallback-and-exhaustion",
			value: "DTSTART:20260101T090000Z\nRRULE:FREQ=DAILY;COUNT=3",
			after: time.Date(2025, 12, 31, 9, 0, 0, 0, time.UTC),
			count: 8,
		},
		{
			name:  "until-exhaustion",
			value: "DTSTART:20260101T090000Z\nRRULE:FREQ=HOURLY;INTERVAL=2;UNTIL=20260101T150000Z",
			after: time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC),
			count: 8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, inc := range []bool{false, true} {
				want := refNextRuns(t, tt.value, tt.after, inc, tt.count, nil)
				got, err := NextRuns(tt.value, tt.after, inc, tt.count, nil)
				if err != nil {
					t.Fatal(err)
				}
				requireSameRuns(t, got, want)
			}
		})
	}
}

func TestNextRunsMatchesReference(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 8, 12, 15, 17, 54, 0, loc)
	rule := "DTSTART;TZID=Asia/Shanghai:20260101T093000\nRRULE:FREQ=DAILY;BYHOUR=9;BYMINUTE=30;BYSECOND=0\nRDATE;TZID=Asia/Shanghai:20260115T100000\nEXDATE;TZID=Asia/Shanghai:20260201T093000"

	want := refPreviewRuns(t, rule, now, 7)
	got, err := NextRuns(rule, now, false, 7, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("runs = %v, want %v", got, want)
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Fatalf("runs[%d] = %v, want %v", i, got[i], want[i])
		}
	}

	if runs, err := NextRuns(rule, now, false, 0, nil); err != nil || len(runs) != 0 {
		t.Fatalf("NextRuns count=0 = %v, %v; want empty, nil", runs, err)
	}
	if _, err := NextRuns("", now, false, 5, nil); err == nil {
		t.Fatal("NextRuns must reject empty rule")
	}
}

func TestNextRuns(t *testing.T) {
	after := time.Date(2026, 7, 27, 8, 0, 0, 0, time.UTC)
	rule := "DTSTART:20260727T080000Z\nRRULE:FREQ=HOURLY;COUNT=8"

	// nil 谓词：直接收集候选，序列严格递增。
	got, err := NextRuns(rule, after, false, 4, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("nil predicate runs = %v, want 4", got)
	}
	for i, dt := range got {
		if i > 0 && !dt.After(got[i-1]) {
			t.Fatalf("runs must be strictly increasing, got %v", got)
		}
	}

	// 谓词跳过中间候选：只推进已接受结果，游标严格递增。
	filtered, err := NextRuns(rule, after, false, 3, func(t time.Time) bool {
		return t.Hour() >= 9 && t.Hour() <= 11
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 3 {
		t.Fatalf("filtered runs = %v, want 3", filtered)
	}
	for i, dt := range filtered {
		if i > 0 && !dt.After(filtered[i-1]) {
			t.Fatalf("filtered runs must be strictly increasing, got %v", filtered)
		}
	}
	wantHours := []int{12, 13, 14}
	for i, h := range wantHours {
		if filtered[i].Hour() != h {
			t.Fatalf("filtered[%d] hour = %d, want %d", i, filtered[i].Hour(), h)
		}
	}

	// 谓词永久排除：耗尽返回已收集结果（空）和 nil error。
	exhausted, err := NextRuns(rule, after, false, 3, func(time.Time) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	if len(exhausted) != 0 {
		t.Fatalf("exhausted runs = %v, want empty", exhausted)
	}
}

func TestNextRunsEvaluatesInvalidPredicateBeforeIncBoundary(t *testing.T) {
	rule := "DTSTART:20260727T080000Z\nRRULE:FREQ=HOURLY;COUNT=4"
	dt := time.Date(2026, 7, 27, 9, 0, 0, 0, time.UTC)
	var evaluated []time.Time
	runs, err := NextRuns(rule, dt, false, 1, func(candidate time.Time) bool {
		evaluated = append(evaluated, candidate)
		return false
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || !runs[0].Equal(dt.Add(time.Hour)) {
		t.Fatalf("runs = %v, want [%v]", runs, dt.Add(time.Hour))
	}
	wantEvaluated := []time.Time{dt.Add(-time.Hour), dt, dt.Add(time.Hour)}
	requireSameRuns(t, evaluated, wantEvaluated)
}

// TestNextRunsIncBoundary 验证边界语义与官方 after 一致：
// inc=true 接受不早于 dt 的候选（含 dt 本身），inc=false 只接受严格晚于 dt 的候选。
func TestNextRunsIncBoundary(t *testing.T) {
	rule := "DTSTART:20260727T080000Z\nRRULE:FREQ=HOURLY;COUNT=4"
	at := time.Date(2026, 7, 27, 9, 0, 0, 0, time.UTC)

	// inc=false：跳过恰好等于 dt 的候选。
	exclusive, err := NextRuns(rule, at, false, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(exclusive) != 1 || !exclusive[0].Equal(at.Add(time.Hour)) {
		t.Fatalf("exclusive runs = %v, want [%v]", exclusive, at.Add(time.Hour))
	}

	// inc=true：接受恰好等于 dt 的候选。
	inclusive, err := NextRuns(rule, at, true, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(inclusive) != 1 || !inclusive[0].Equal(at) {
		t.Fatalf("inclusive runs = %v, want [%v]", inclusive, at)
	}

	// inc=true 且 dt 落在两个候选之间：仍取下一个不早于 dt 的候选。
	between := at.Add(30 * time.Minute)
	inclusive, err = NextRuns(rule, between, true, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []time.Time{at.Add(time.Hour), at.Add(2 * time.Hour)}
	if len(inclusive) != 2 || !inclusive[0].Equal(want[0]) || !inclusive[1].Equal(want[1]) {
		t.Fatalf("inclusive runs = %v, want %v", inclusive, want)
	}
}

func TestShiftSetForQueryFallbackRules(t *testing.T) {
	after := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	stringRules := []string{
		"DTSTART:20260101T090000Z\nRRULE:FREQ=DAILY;COUNT=3;BYHOUR=9",
		"DTSTART:20260101T090000Z\nRRULE:FREQ=YEARLY;BYWEEKNO=1;BYDAY=MO",
		"DTSTART:20260101T090000Z\nRRULE:FREQ=YEARLY;BYYEARDAY=1",
		"DTSTART:20260101T090000Z\nRRULE:FREQ=MONTHLY;BYMONTHDAY=1,15;BYSETPOS=1",
		"DTSTART:20200229T090000Z\nRRULE:FREQ=YEARLY",
	}
	for _, rule := range stringRules {
		set, err := ParseSet(rule)
		if err != nil {
			t.Fatalf("parse %q: %v", rule, err)
		}
		if shifted := ShiftSetForQuery(set, after); shifted != nil {
			t.Errorf("rule %q must fall back to original set", rule)
		}
		want := refNextAfter(t, rule, after)
		runs, err := NextRuns(rule, after, false, 1, nil)
		if err != nil {
			t.Fatalf("NextRuns failed for %q: %v", rule, err)
		}
		if want.IsZero() {
			if len(runs) != 0 {
				t.Errorf("rule %q: NextRuns = %v, want exhausted", rule, runs)
			}
			continue
		}
		if len(runs) != 1 || !runs[0].Equal(want) {
			t.Errorf("rule %q: NextRuns = %v, want %v", rule, runs, want)
		}
	}
	clampedRule := "DTSTART:20240131T090000Z\nRRULE:FREQ=MONTHLY"
	clampedSet, err := ParseSet(clampedRule)
	if err != nil {
		t.Fatal(err)
	}
	clampedAfter := time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)
	if shifted := ShiftSetForQuery(clampedSet, clampedAfter); shifted != nil {
		t.Fatalf("AddDate-clamped rule must fall back, got shifted DTSTART %v", shifted.GetDTStart())
	}
	got, err := NextRuns(clampedRule, clampedAfter, false, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	requireSameRuns(t, got, refNextRuns(t, clampedRule, clampedAfter, false, 2, nil))

	dtstart := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	optionRules := []rrule.ROption{
		{Freq: rrule.MONTHLY, Dtstart: dtstart, Bymonthday: []int{1, 15}, Bysetpos: []int{1}},
		{Freq: rrule.YEARLY, Dtstart: dtstart, Byeaster: []int{1}},
	}
	for _, option := range optionRules {
		rule, err := rrule.NewRRule(option)
		if err != nil {
			t.Fatalf("build rule %v: %v", option, err)
		}
		set := &rrule.Set{}
		set.RRule(rule)
		if shifted := ShiftSetForQuery(set, after); shifted != nil {
			t.Errorf("option %v must fall back to original set", option)
		}
	}
}

func TestShiftSetForQueryAnchorAlignment(t *testing.T) {
	tests := []struct {
		name  string
		value string
		after time.Time
		want  time.Time
	}{
		{name: "yearly-interval2", value: "DTSTART:20200115T090000Z\nRRULE:FREQ=YEARLY;INTERVAL=2", after: time.Date(2025, 8, 12, 0, 0, 0, 0, time.UTC), want: time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)},
		{name: "monthly-interval3", value: "DTSTART:20240115T090000Z\nRRULE:FREQ=MONTHLY;INTERVAL=3", after: time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC), want: time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)},
		{name: "weekly-interval2", value: "DTSTART:20240101T090000Z\nRRULE:FREQ=WEEKLY;INTERVAL=2", after: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), want: time.Date(2024, 1, 29, 9, 0, 0, 0, time.UTC)},
		{name: "daily-interval3", value: "DTSTART:20240101T090000Z\nRRULE:FREQ=DAILY;INTERVAL=3", after: time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC), want: time.Date(2024, 1, 10, 9, 0, 0, 0, time.UTC)},
		{name: "hourly-interval5", value: "DTSTART:20240101T091000Z\nRRULE:FREQ=HOURLY;INTERVAL=5", after: time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC), want: time.Date(2024, 1, 2, 10, 10, 0, 0, time.UTC)},
		{name: "minutely-interval7", value: "DTSTART:20240101T091011Z\nRRULE:FREQ=MINUTELY;INTERVAL=7", after: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC), want: time.Date(2024, 1, 1, 9, 59, 11, 0, time.UTC)},
		{name: "secondly-interval11", value: "DTSTART:20240101T091011Z\nRRULE:FREQ=SECONDLY;INTERVAL=11", after: time.Date(2024, 1, 1, 9, 12, 0, 0, time.UTC), want: time.Date(2024, 1, 1, 9, 11, 50, 0, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set, err := ParseSet(tt.value)
			if err != nil {
				t.Fatal(err)
			}
			shifted := ShiftSetForQuery(set, tt.after)
			if shifted == nil {
				t.Fatal("expected query set to be shifted")
			}
			anchor := shifted.GetDTStart()
			if anchor.After(tt.after) {
				t.Fatalf("shifted DTSTART %v is after query point %v", anchor, tt.after)
			}
			if !anchor.Equal(tt.want) {
				t.Fatalf("shifted DTSTART = %v, want interval-aligned %v", anchor, tt.want)
			}
			got, err := NextRuns(tt.value, tt.after, false, 4, nil)
			if err != nil {
				t.Fatal(err)
			}
			requireSameRuns(t, got, refNextRuns(t, tt.value, tt.after, false, 4, nil))
		})
	}
}

func TestShiftSetForQueryNearDtStartIsCorrect(t *testing.T) {
	// DTSTART 与查询点同在一个周期内：无需平移，行为与原始一致。
	after := time.Date(2026, 8, 12, 10, 15, 0, 0, time.UTC)
	rule := "DTSTART:20260812T093000Z\nRRULE:FREQ=HOURLY;BYMINUTE=0"
	set, err := ParseSet(rule)
	if err != nil {
		t.Fatal(err)
	}
	if shifted := ShiftSetForQuery(set, after); shifted != nil {
		t.Fatalf("DTSTART within one period must not be shifted, got %v", shifted.GetDTStart())
	}
	want := refNextAfter(t, rule, after)
	runs, err := NextRuns(rule, after, false, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || !runs[0].Equal(want) {
		t.Fatalf("NextRuns = %v, want %v", runs, want)
	}

	// anchor 恰好落在查询点上的平移也必须保持结果一致。
	after = time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	rule = "DTSTART:20260812T090000Z\nRRULE:FREQ=HOURLY;BYMINUTE=0"
	want = refNextAfter(t, rule, after)
	runs, err = NextRuns(rule, after, false, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || !runs[0].Equal(want) {
		t.Fatalf("NextRuns = %v, want %v", runs, want)
	}
}

func TestNextRunsAcrossDSTMatchesUnshiftedSet(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("location unavailable: %v", err)
	}
	tests := []struct {
		name  string
		rule  string
		query time.Time
	}{
		{name: "spring-daily", rule: "DTSTART;TZID=America/New_York:20260306T090000\nRRULE:FREQ=DAILY;INTERVAL=2", query: time.Date(2026, 3, 10, 8, 30, 0, 0, loc)},
		{name: "spring-weekly", rule: "DTSTART;TZID=America/New_York:20260223T090000\nRRULE:FREQ=WEEKLY;INTERVAL=2", query: time.Date(2026, 3, 23, 8, 30, 0, 0, loc)},
		{name: "fall-daily", rule: "DTSTART;TZID=America/New_York:20261030T090000\nRRULE:FREQ=DAILY;INTERVAL=1", query: time.Date(2026, 11, 2, 8, 30, 0, 0, loc)},
		{name: "fall-weekly", rule: "DTSTART;TZID=America/New_York:20261005T090000\nRRULE:FREQ=WEEKLY;INTERVAL=1", query: time.Date(2026, 11, 2, 8, 30, 0, 0, loc)},
		{name: "fall-hourly", rule: "DTSTART;TZID=America/New_York:20261030T090000\nRRULE:FREQ=HOURLY;BYMINUTE=0;BYSECOND=0", query: time.Date(2026, 11, 2, 9, 30, 0, 0, loc)},
		{name: "fall-minutely", rule: "DTSTART;TZID=America/New_York:20261030T090000\nRRULE:FREQ=MINUTELY;BYSECOND=0", query: time.Date(2026, 11, 1, 5, 30, 0, 0, loc)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set, parseErr := ParseSet(tt.rule)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			shifted := ShiftSetForQuery(set, tt.query)
			if strings.Contains(tt.name, "daily") || strings.Contains(tt.name, "weekly") {
				if shifted != nil {
					t.Fatalf("calendar frequency across DST must fall back, got shifted DTSTART %v", shifted.GetDTStart())
				}
			} else if shifted != nil && shifted.GetDTStart().After(tt.query) {
				t.Fatalf("shifted DTSTART %v is after query point %v", shifted.GetDTStart(), tt.query)
			}
			for _, inc := range []bool{false, true} {
				want := refNextRuns(t, tt.rule, tt.query, inc, 4, nil)
				runs, err := NextRuns(tt.rule, tt.query, inc, 4, nil)
				if err != nil {
					t.Fatalf("%s: NextRuns failed: %v", tt.name, err)
				}
				requireSameRuns(t, runs, want)
			}
		})
	}
}

func TestShiftDtStartByPeriod(t *testing.T) {
	loc := time.UTC
	date := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, loc) }
	datetime := func(y int, m time.Month, d, hh, mm, ss int) time.Time {
		return time.Date(y, m, d, hh, mm, ss, 0, loc)
	}
	tests := []struct {
		name     string
		dtstart  time.Time
		after    time.Time
		freq     rrule.Frequency
		interval int
		want     time.Time
		wantOK   bool
	}{
		// YEARLY：年差整对齐，直接在 after 之前。
		{name: "yearly-simple", dtstart: date(2020, 6, 15), after: date(2026, 8, 12), freq: rrule.YEARLY, interval: 1, want: date(2026, 6, 15), wantOK: true},
		// YEARLY：+6 年越过 after（9 月晚于 8 月），回退一个间隔到上一个相位 2025-09-15。
		{name: "yearly-retreat-previous-phase", dtstart: date(2020, 9, 15), after: date(2026, 8, 12), freq: rrule.YEARLY, interval: 1, want: date(2025, 9, 15), wantOK: true},
		// YEARLY INTERVAL=2：5 年对齐到 4 年。
		{name: "yearly-interval2-floor", dtstart: date(2020, 6, 15), after: date(2025, 8, 12), freq: rrule.YEARLY, interval: 2, want: date(2024, 6, 15), wantOK: true},
		// YEARLY 闰日：AddDate 进位到 2026-03-01，月/日相位被破坏 → 放弃平移。
		{name: "yearly-feb29-clamp", dtstart: date(2020, 2, 29), after: date(2026, 8, 12), freq: rrule.YEARLY, interval: 1, want: time.Time{}, wantOK: false},
		// YEARLY：未跨满一个间隔（years=0）→ 不平移。
		{name: "yearly-not-yet-one-period", dtstart: date(2020, 6, 15), after: date(2020, 8, 12), freq: rrule.YEARLY, interval: 1, want: time.Time{}, wantOK: false},
		// YEARLY：回退到 dtstart 自身（2026-06-15 越过、2025-06-15 即起点）→ Equal 兜底 false。
		{name: "yearly-retreat-to-dtstart", dtstart: date(2020, 6, 15), after: date(2021, 3, 1), freq: rrule.YEARLY, interval: 1, want: time.Time{}, wantOK: false},

		// MONTHLY：+31 月=2026-08-15 越过 → 回退到 2026-07-15。
		{name: "monthly-retreat-previous-phase", dtstart: date(2024, 1, 15), after: date(2026, 8, 12), freq: rrule.MONTHLY, interval: 1, want: date(2026, 7, 15), wantOK: true},
		// MONTHLY INTERVAL=3：31 月对齐到 30 月（07-15 未越过）。
		{name: "monthly-interval3-floor", dtstart: date(2024, 1, 15), after: date(2026, 8, 12), freq: rrule.MONTHLY, interval: 3, want: date(2026, 7, 15), wantOK: true},
		// MONTHLY 月末：1/31 +2 月=3/31 越过，+1 月=3/02 相位破坏 → 放弃。
		{name: "monthly-day31-clamp", dtstart: date(2024, 1, 31), after: date(2024, 3, 5), freq: rrule.MONTHLY, interval: 1, want: time.Time{}, wantOK: false},
		// MONTHLY：回退到 dtstart（2024-02-15 越过，2024-01-15 即起点）→ Equal 兜底 false。
		{name: "monthly-retreat-to-dtstart", dtstart: date(2024, 1, 15), after: date(2024, 2, 1), freq: rrule.MONTHLY, interval: 1, want: time.Time{}, wantOK: false},
		// MONTHLY：月相位对、日号 15 保持不变 → 平移成功。
		{name: "monthly-day15-preserved", dtstart: date(2024, 1, 15), after: date(2024, 4, 20), freq: rrule.MONTHLY, interval: 1, want: date(2024, 4, 15), wantOK: true},

		// WEEKLY：2024-01-01(周一) 起 954 天对齐到 952 天=136 周，保持周一相位。
		{name: "weekly-954d-floor", dtstart: date(2024, 1, 1), after: date(2026, 8, 12), freq: rrule.WEEKLY, interval: 1, want: date(2026, 8, 10), wantOK: true},
		// WEEKLY INTERVAL=2：954 对齐到 952=偶数周。
		{name: "weekly-interval2-floor", dtstart: date(2024, 1, 1), after: date(2026, 8, 12), freq: rrule.WEEKLY, interval: 2, want: date(2026, 8, 10), wantOK: true},
		// WEEKLY：不足一周即不平移。
		{name: "weekly-too-early", dtstart: date(2024, 1, 1), after: date(2024, 1, 3), freq: rrule.WEEKLY, interval: 1, want: time.Time{}, wantOK: false},
		// WEEKLY：对齐到最近周一（2024-01-08）。
		{name: "weekly-floor-monday", dtstart: date(2024, 1, 1), after: date(2024, 1, 10), freq: rrule.WEEKLY, interval: 1, want: date(2024, 1, 8), wantOK: true},

		// DAILY：954 天恰在 after，整对齐不动。
		{name: "daily-exact", dtstart: date(2024, 1, 1), after: date(2026, 8, 12), freq: rrule.DAILY, interval: 1, want: date(2026, 8, 12), wantOK: true},
		// DAILY INTERVAL=3：955 天对齐到 954 天（divisible-by-3 相位保留）。
		{name: "daily-interval3-floor", dtstart: date(2024, 1, 1), after: date(2026, 8, 13), freq: rrule.DAILY, interval: 3, want: date(2026, 8, 12), wantOK: true},
		// DAILY：不足一个间隔不平移。
		{name: "daily-too-early", dtstart: date(2024, 1, 1), after: date(2024, 1, 2), freq: rrule.DAILY, interval: 100, want: time.Time{}, wantOK: false},

		// HOURLY：652 小时对齐，锚点 14:00，分/秒相位保持。
		{name: "hourly-retreat-previous-phase", dtstart: datetime(2024, 1, 1, 10, 0, 0), after: datetime(2024, 1, 28, 14, 30, 0), freq: rrule.HOURLY, interval: 1, want: datetime(2024, 1, 28, 14, 0, 0), wantOK: true},
		// HOURLY INTERVAL=6：652 对齐到 648。
		{name: "hourly-interval6-floor", dtstart: datetime(2024, 1, 1, 10, 0, 0), after: datetime(2024, 1, 28, 14, 30, 0), freq: rrule.HOURLY, interval: 6, want: datetime(2024, 1, 28, 10, 0, 0), wantOK: true},
		// HOURLY：小时差不足则不平移。
		{name: "hourly-too-early", dtstart: datetime(2024, 1, 1, 10, 0, 0), after: datetime(2024, 1, 1, 10, 59, 0), freq: rrule.HOURLY, interval: 1, want: time.Time{}, wantOK: false},

		// MINUTELY：17.5 分钟截断到 17，对齐到 15。
		{name: "minutely-interval5-floor", dtstart: datetime(2024, 1, 1, 10, 0, 0), after: datetime(2024, 1, 1, 10, 17, 30), freq: rrule.MINUTELY, interval: 5, want: datetime(2024, 1, 1, 10, 15, 0), wantOK: true},
		// MINUTELY：秒相位保持（从 :15 起的 15 分钟对齐)。
		{name: "minutely-second-phase-preserved", dtstart: datetime(2024, 1, 1, 10, 0, 15), after: datetime(2024, 1, 1, 10, 17, 30), freq: rrule.MINUTELY, interval: 5, want: datetime(2024, 1, 1, 10, 15, 15), wantOK: true},

		// SECONDLY：50 秒对齐到 45。
		{name: "secondly-interval15-floor", dtstart: datetime(2024, 1, 1, 10, 0, 0), after: datetime(2024, 1, 1, 10, 0, 50), freq: rrule.SECONDLY, interval: 15, want: datetime(2024, 1, 1, 10, 0, 45), wantOK: true},

		// 前置守卫：after 不晚于 dtstart 或 dtstart 为零。
		{name: "after-not-after-dtstart", dtstart: date(2026, 8, 12), after: date(2026, 8, 12), freq: rrule.DAILY, interval: 1, want: time.Time{}, wantOK: false},
		{name: "zero-dtstart", dtstart: time.Time{}, after: date(2026, 8, 12), freq: rrule.DAILY, interval: 1, want: time.Time{}, wantOK: false},
		// interval < 1 归一化为 1。
		{name: "interval-zero-normalized", dtstart: date(2024, 1, 1), after: date(2024, 1, 10), freq: rrule.DAILY, interval: 0, want: date(2024, 1, 10), wantOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := shiftDtStartByPeriod(tt.dtstart, tt.after, tt.freq, tt.interval)
			if ok != tt.wantOK {
				t.Fatalf("shiftDtStartByPeriod(%v, %v, %d, %d) ok = %v, want %v", tt.dtstart.Format("2006-01-02 15:04:05"), tt.after.Format("2006-01-02 15:04:05"), tt.freq, tt.interval, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if !got.Equal(tt.want) {
				t.Errorf("shiftDtStartByPeriod(%v, %v, %d, %d) = %v, want %v", tt.dtstart.Format("2006-01-02 15:04:05"), tt.after.Format("2006-01-02 15:04:05"), tt.freq, tt.interval, got.Format("2006-01-02 15:04:05"), tt.want.Format("2006-01-02 15:04:05"))
			}
			if got.After(tt.after) {
				t.Errorf("shiftDtStartByPeriod shifted %v must not be after query point %v", got, tt.after)
			}
		})
	}
}
