package rrulex

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/teambition/rrule-go"
)

func TestDescribe(t *testing.T) {
	tests := []struct {
		name        string
		rule        string
		contains    []string
		notContains []string
	}{
		{
			name: "daily with timezone and until",
			rule: "DTSTART;TZID=Asia/Shanghai:20260727T000000\nRRULE:FREQ=DAILY;UNTIL=20261231T155959Z;BYHOUR=9;BYMINUTE=30;BYSECOND=0",
			contains: []string{
				"每天 09:30 执行",
				"周期规则有效期：2026-07-27 00:00:00 至 2026-12-31 23:59:59",
				"时区：Asia/Shanghai；遇不存在或重复的本地时间，以时区库实际解析结果为准",
			},
		},
		{
			name:     "monthly negative day interval",
			rule:     testRRuleSet("FREQ=MONTHLY;INTERVAL=2;BYMONTHDAY=-1,-3;BYHOUR=8;BYMINUTE=0;BYSECOND=0"),
			contains: []string{"按 2 个月间隔", "倒数第 3 天、最后一天", "08:00 执行"},
		},
		{
			name:     "weekly intersection",
			rule:     testRRuleSet("FREQ=WEEKLY;BYDAY=MO,FR;BYHOUR=9;BYMINUTE=15;BYSECOND=0"),
			contains: []string{"每周", "周一、周五", "09:15 执行"},
		},
		{
			name:     "ordinal weekday",
			rule:     testRRuleSet("FREQ=MONTHLY;BYDAY=1MO,-1FR;BYHOUR=10;BYMINUTE=0;BYSECOND=0"),
			contains: []string{"第 1 个周一", "倒数第 1 个周五", "10:00 执行"},
		},
		{
			name:     "hourly interval",
			rule:     testRRuleSet("FREQ=HOURLY;INTERVAL=3;BYMINUTE=5;BYSECOND=10"),
			contains: []string{"按 3 小时间隔", "时间条件：分钟为 05，秒为 10", "执行"},
		},
		{
			name:     "minutely",
			rule:     testRRuleSet("FREQ=MINUTELY;INTERVAL=10;BYSECOND=5"),
			contains: []string{"按 10 分钟间隔", "时间条件：秒为 05", "执行"},
		},
		{
			name: "minutely filtered to daily fixed times",
			rule: "DTSTART;TZID=Asia/Shanghai:20260727T000000\n" +
				"RRULE:FREQ=MINUTELY;UNTIL=20260731T155959Z;BYHOUR=10,13;BYMINUTE=0,1,2,3,4,5,6,7,8,9;BYSECOND=0",
			contains: []string{
				"每天 10:00、10:01、10:02、10:03、10:04、10:05、10:06、10:07、10:08、10:09、" +
					"13:00、13:01、13:02、13:03、13:04、13:05、13:06、13:07、13:08、13:09 执行",
				"周期规则有效期：2026-07-27 00:00:00 至 2026-07-31 23:59:59",
			},
		},
		{
			name:        "minutely interval retains phase semantics",
			rule:        testRRuleSet("FREQ=MINUTELY;INTERVAL=2;BYHOUR=10,13;BYMINUTE=0,1,2,3;BYSECOND=0"),
			contains:    []string{"按 2 分钟间隔", "时间条件：小时为 10、13，分钟为 00、01、02、03，秒为 00 执行"},
			notContains: []string{"每天"},
		},
		{
			name: "minutely fixed times use dtstart second default",
			rule: "DTSTART;TZID=Asia/Shanghai:20260727T000005\n" +
				"RRULE:FREQ=MINUTELY;BYHOUR=10;BYMINUTE=0,1",
			contains: []string{"每天 10:00:05、10:01:05 执行"},
		},
		{
			name:     "hourly filtered to daily fixed times",
			rule:     testRRuleSet("FREQ=HOURLY;BYHOUR=10,13;BYMINUTE=0,30;BYSECOND=0"),
			contains: []string{"每天 10:00、10:30、13:00、13:30 执行"},
		},
		{
			name: "hourly fixed times use dtstart minute and second defaults",
			rule: "DTSTART;TZID=Asia/Shanghai:20260727T001505\n" +
				"RRULE:FREQ=HOURLY;BYHOUR=10,13",
			contains: []string{"每天 10:15:05、13:15:05 执行"},
		},
		{
			name:        "hourly interval retains phase semantics",
			rule:        testRRuleSet("FREQ=HOURLY;INTERVAL=2;BYHOUR=10,13;BYMINUTE=0,30;BYSECOND=0"),
			contains:    []string{"按 2 小时间隔", "时间条件：小时为 10、13，分钟为 00、30，秒为 00 执行"},
			notContains: []string{"每天"},
		},
		{
			name:        "hourly interval with sparse hour filter",
			rule:        "DTSTART:20260727T010000Z\nRRULE:FREQ=HOURLY;INTERVAL=2;BYHOUR=1,5,7;BYMINUTE=0;BYSECOND=0",
			contains:    []string{"以开始时间为基准，按 2 小时间隔", "时间条件：小时为 01、05、07，分钟为 00，秒为 00 执行"},
			notContains: []string{"每 2 小时", "每天 01:00"},
		},
		{
			name:     "cartesian times",
			rule:     testRRuleSet("FREQ=DAILY;BYHOUR=8,9;BYMINUTE=0,30;BYSECOND=0"),
			contains: []string{"08:00、08:30、09:00、09:30 执行"},
		},
		{
			name:     "yearly filters and count",
			rule:     testRRuleSet("FREQ=YEARLY;INTERVAL=2;BYMONTH=1,6;BYMONTHDAY=1;COUNT=4;BYHOUR=0;BYMINUTE=0;BYSECOND=0"),
			contains: []string{"按 2 年间隔，1月、6月，且各指定月份的第 1 天", "周期规则最多生成 4 次"},
		},
		{
			name: "rrule set dates use dtstart timezone",
			rule: "DTSTART;TZID=Asia/Shanghai:20260727T090000\nRRULE:FREQ=DAILY;COUNT=2\n" +
				"RDATE:20260730T100000\nEXDATE:20260728T090000",
			contains: []string{
				"每天 09:00 执行", "周期规则最多生成 2 次",
				"额外纳入候选：2026-07-30 10:00:00", "从周期规则与额外候选的合并结果中排除：2026-07-28 09:00:00",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Describe(tt.rule)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Fatalf("Describe() = %q, want substring %q", got, want)
				}
			}
			for _, unwanted := range tt.notContains {
				if strings.Contains(got, unwanted) {
					t.Fatalf("Describe() = %q, must not contain %q", got, unwanted)
				}
			}
		})
	}
}

func TestDescribeEmpty(t *testing.T) {
	got, err := Describe("")
	if err != nil || got != "" {
		t.Fatalf("Describe(empty) = %q, %v", got, err)
	}
}

func TestDescribeSupportsCRLFSet(t *testing.T) {
	description, err := Describe("DTSTART;TZID=Asia/Shanghai:20260727T090000\r\n" +
		"RRULE:FREQ=DAILY;COUNT=2\r\n" +
		"RDATE:20260730T100000\r\n" +
		"EXDATE:20260728T090000")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"每天 09:00 执行",
		"额外纳入候选：2026-07-30 10:00:00",
		"从周期规则与额外候选的合并结果中排除：2026-07-28 09:00:00",
	} {
		if !strings.Contains(description, want) {
			t.Fatalf("description = %q, want substring %q", description, want)
		}
	}
}

func TestDescribeErrors(t *testing.T) {
	if _, err := Describe("not-an-rrule"); err == nil {
		t.Fatal("invalid rule should fail")
	}
	if _, err := Describe("FREQ=DAILY;BYHOUR=9;BYMINUTE=0"); err == nil {
		t.Fatal("bare RRULE should fail")
	}
	if _, err := Describe("RRULE:FREQ=DAILY;BYHOUR=9;BYMINUTE=0"); err == nil {
		t.Fatal("RRULE without DTSTART should fail")
	}
	for _, rule := range []string{
		testRRuleSet("FREQ=YEARLY;BYYEARDAY=100"),
		testRRuleSet("FREQ=YEARLY;BYWEEKNO=2"),
		testRRuleSet("FREQ=YEARLY;BYEASTER=1"),
		testRRuleSet("FREQ=WEEKLY;BYDAY=1MO"),
		testRRuleSet("FREQ=MONTHLY;BYDAY=1MO,TU"),
		testRRuleSet("FREQ=YEARLY;BYDAY=1MO,TU"),
	} {
		if _, err := Describe(rule); !errors.Is(err, ErrUnsupportedDescription) {
			t.Fatalf("Describe(%q) error = %v, want ErrUnsupportedDescription", rule, err)
		}
	}
}

func TestDescribeExhaustedRulesUseConditionalWording(t *testing.T) {
	tests := []struct {
		name string
		rule string
		want string
	}{
		{
			name: "impossible calendar intersection",
			rule: "DTSTART:20260101T090000Z\n" +
				"RRULE:FREQ=YEARLY;UNTIL=20261231T090000Z;BYMONTH=2;BYMONTHDAY=30",
			want: "执行（仅在上述条件形成匹配候选时）",
		},
		{
			name: "until before dtstart",
			rule: "DTSTART:20260727T090000Z\n" +
				"RRULE:FREQ=DAILY;UNTIL=20260726T090000Z",
			want: "周期规则边界已倒置：开始于 2026-07-27 09:00:00，截止于 2026-07-26 09:00:00",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description, err := Describe(tt.rule)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(description, tt.want) {
				t.Fatalf("Describe() = %q, want %q", description, tt.want)
			}
			if strings.Contains(description, "每天 09:00 执行，") || strings.Contains(description, "周期规则有效期：2026-07-27") {
				t.Fatalf("Describe() makes an unconditional or inverted validity claim: %q", description)
			}

			set, err := rrule.StrToRRuleSet(tt.rule)
			if err != nil {
				t.Fatal(err)
			}
			if occurrences := set.All(); len(occurrences) != 0 {
				t.Fatalf("occurrences = %v, want exhausted rule", occurrences)
			}
		})
	}
}

func TestDescribeRejectsDuplicateClockValuesWithBySetPos(t *testing.T) {
	duplicateRule := "DTSTART:20260701T000000Z\n" +
		"RRULE:FREQ=MONTHLY;COUNT=2;BYDAY=MO;BYHOUR=9,9;BYMINUTE=0;BYSECOND=0;BYSETPOS=2"
	if _, err := Describe(duplicateRule); !errors.Is(err, ErrUnsupportedDescription) {
		t.Fatalf("Describe() error = %v, want ErrUnsupportedDescription", err)
	}

	uniqueRule := strings.Replace(duplicateRule, "BYHOUR=9,9", "BYHOUR=9", 1)
	duplicateSet, err := rrule.StrToRRuleSet(duplicateRule)
	if err != nil {
		t.Fatal(err)
	}
	uniqueSet, err := rrule.StrToRRuleSet(uniqueRule)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	duplicateOccurrences := duplicateSet.Between(start, end, true)
	uniqueOccurrences := uniqueSet.Between(start, end, true)
	if len(duplicateOccurrences) != 1 || len(uniqueOccurrences) != 1 {
		t.Fatalf("duplicate occurrences = %v, unique occurrences = %v", duplicateOccurrences, uniqueOccurrences)
	}
	if duplicateOccurrences[0].Equal(uniqueOccurrences[0]) {
		t.Fatalf("duplicate clock value must change BYSETPOS selection: duplicate = %v, unique = %v", duplicateOccurrences, uniqueOccurrences)
	}
	if duplicateOccurrences[0].Day() != 6 || uniqueOccurrences[0].Day() != 13 {
		t.Fatalf("duplicate occurrence = %v, unique occurrence = %v; want first and second Monday", duplicateOccurrences[0], uniqueOccurrences[0])
	}

	// At SECONDLY frequency BYSECOND filters the current period; it does not
	// expand the single-element timeset indexed by BYSETPOS.
	filterRule := "DTSTART:20260727T091005Z\n" +
		"RRULE:FREQ=SECONDLY;COUNT=2;BYSECOND=5,5;BYSETPOS=1"
	if _, err := Describe(filterRule); err != nil {
		t.Fatalf("Describe() rejected duplicate filter values that do not change BYSETPOS slots: %v", err)
	}
	filterSet, err := rrule.StrToRRuleSet(filterRule)
	if err != nil {
		t.Fatal(err)
	}
	filterOccurrences := filterSet.All()
	if len(filterOccurrences) != 2 || filterOccurrences[0].Second() != 5 || filterOccurrences[1].Second() != 5 {
		t.Fatalf("filter occurrences = %v, want two :05 occurrences", filterOccurrences)
	}
}

func TestDescribeSupportsSecondlyBySetPos(t *testing.T) {
	for _, tt := range []struct {
		name     string
		position int
		want     string
	}{
		{name: "first candidate", position: 1, want: "第 1 个执行（仅在该位置存在候选时）"},
		{name: "last candidate", position: -1, want: "最后一个执行（仅在该位置存在候选时）"},
		{name: "nonexistent second candidate", position: 2, want: "第 2 个执行（仅在该位置存在候选时）"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rule := fmt.Sprintf("DTSTART:20260727T091005Z\nRRULE:FREQ=SECONDLY;INTERVAL=2;COUNT=2;BYSETPOS=%d", tt.position)
			description, err := Describe(rule)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(description, "按 2 秒间隔；每个周期先按上述条件形成候选，再选择"+tt.want) {
				t.Fatalf("Describe() = %q, want position-aware conditional wording %q", description, tt.want)
			}
		})
	}
	// Only positions 1 and -1 select SECONDLY's single candidate. Do not iterate
	// position 2: rrule-go checks UNTIL only after BYSETPOS selects a candidate,
	// so a nonexistent position scans until its maximum year even with a bound.
	rule := "DTSTART:20260727T091005Z\n" +
		"RRULE:FREQ=SECONDLY;INTERVAL=2;COUNT=2;BYSETPOS=1"
	set, err := rrule.StrToRRuleSet(rule)
	if err != nil {
		t.Fatal(err)
	}
	wantOccurrences := []time.Time{
		time.Date(2026, 7, 27, 9, 10, 5, 0, time.UTC),
		time.Date(2026, 7, 27, 9, 10, 7, 0, time.UTC),
	}
	got := set.All()
	if len(got) != len(wantOccurrences) {
		t.Fatalf("occurrences = %v, want %v", got, wantOccurrences)
	}
	for i := range wantOccurrences {
		if !got[i].Equal(wantOccurrences[i]) {
			t.Fatalf("occurrences[%d] = %v, want %v", i, got[i], wantOccurrences[i])
		}
	}
}

func TestDescribeSetDatesDescribeCombinedSetSemantics(t *testing.T) {
	rule := "DTSTART:20260727T090000Z\n" +
		"RRULE:FREQ=DAILY;COUNT=1\n" +
		"RDATE:20260727T090000Z,20260730T100000Z\n" +
		"EXDATE:20260727T090000Z,20260730T100000Z"
	description, err := Describe(rule)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"额外纳入候选：2026-07-27 09:00:00 +00:00、2026-07-30 10:00:00 +00:00",
		"从周期规则与额外候选的合并结果中排除：2026-07-27 09:00:00 +00:00、2026-07-30 10:00:00 +00:00",
	} {
		if !strings.Contains(description, want) {
			t.Fatalf("Describe() = %q, want %q", description, want)
		}
	}
	if strings.Contains(description, "额外执行") {
		t.Fatalf("Describe() must not promise RDATE execution: %q", description)
	}

	set, err := rrule.StrToRRuleSet(rule)
	if err != nil {
		t.Fatal(err)
	}
	if occurrences := set.All(); len(occurrences) != 0 {
		t.Fatalf("occurrences = %v, want empty combined set", occurrences)
	}
}

func TestDescribeRejectsUnsafeNegativeMonthlyOrdinalWeekdays(t *testing.T) {
	for _, rule := range []string{
		"DTSTART:20260101T090000Z\nRRULE:FREQ=MONTHLY;COUNT=1;BYDAY=-6MO",
		"DTSTART:20260101T090000Z\nRRULE:FREQ=YEARLY;COUNT=1;BYMONTH=1;BYDAY=-6MO",
	} {
		if _, err := Describe(rule); !errors.Is(err, ErrUnsupportedDescription) {
			t.Fatalf("Describe(%q) error = %v, want ErrUnsupportedDescription", rule, err)
		}
	}
}

func TestDescribeUsesNormalizedOptions(t *testing.T) {
	tests := []struct {
		name        string
		rule        string
		description string
		occurrences []time.Time
	}{
		{
			name:        "yearly date defaults",
			rule:        testRRuleSet("FREQ=YEARLY;COUNT=2;BYSETPOS=1"),
			description: "每年7月，且各指定月份的第 27 天 00:00；每个周期先按上述条件形成候选，再选择第 1 个执行",
			occurrences: []time.Time{
				time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC),
				time.Date(2027, 7, 27, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:        "monthly date default",
			rule:        testRRuleSet("FREQ=MONTHLY;COUNT=2;BYSETPOS=1"),
			description: "每月第 27 天 00:00；每个周期先按上述条件形成候选，再选择第 1 个执行",
			occurrences: []time.Time{
				time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC),
				time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:        "weekly weekday default",
			rule:        testRRuleSet("FREQ=WEEKLY;COUNT=2;BYSETPOS=1"),
			description: "以周一为一周起始，每周周一 00:00；每个周期先按上述条件形成候选，再选择第 1 个执行",
			occurrences: []time.Time{
				time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC),
				time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:        "daily clock defaults",
			rule:        testRRuleSet("FREQ=DAILY;COUNT=2;BYSETPOS=1"),
			description: "每天 00:00；每个周期先按上述条件形成候选，再选择第 1 个执行",
			occurrences: []time.Time{
				time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC),
				time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description, err := Describe(tt.rule)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(description, tt.description) {
				t.Fatalf("Describe() = %q, want substring %q", description, tt.description)
			}

			set, err := rrule.StrToRRuleSet(tt.rule)
			if err != nil {
				t.Fatal(err)
			}
			got := set.All()
			if len(got) != len(tt.occurrences) {
				t.Fatalf("occurrences = %v, want %v", got, tt.occurrences)
			}
			for i := range tt.occurrences {
				if !got[i].Equal(tt.occurrences[i]) {
					t.Fatalf("occurrences[%d] = %v, want %v", i, got[i], tt.occurrences[i])
				}
			}
		})
	}
}

func TestDescribeBySetPosFollowsDateAndTimeExpansion(t *testing.T) {
	rule := "DTSTART:20260701T000000Z\n" +
		"RRULE:FREQ=MONTHLY;COUNT=2;BYDAY=MO,TU;BYHOUR=9,17;BYMINUTE=0;BYSECOND=0;BYSETPOS=2"

	description, err := Describe(rule)
	if err != nil {
		t.Fatal(err)
	}
	want := "每月周一、周二 09:00、17:00；每个周期先按上述条件形成候选，再选择第 2 个执行"
	if !strings.Contains(description, want) {
		t.Fatalf("Describe() = %q, want substring %q", description, want)
	}

	set, err := rrule.StrToRRuleSet(rule)
	if err != nil {
		t.Fatal(err)
	}
	got := set.Between(
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		true,
	)
	wantOccurrences := []time.Time{
		time.Date(2026, 7, 6, 17, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 3, 17, 0, 0, 0, time.UTC),
	}
	if len(got) != len(wantOccurrences) {
		t.Fatalf("occurrences = %v, want %v", got, wantOccurrences)
	}
	for i := range wantOccurrences {
		if !got[i].Equal(wantOccurrences[i]) {
			t.Fatalf("occurrences[%d] = %v, want %v", i, got[i], wantOccurrences[i])
		}
	}
}

func TestRenderSetPositionsUsesSelectionOrderAndNaturalConjunction(t *testing.T) {
	got := renderSetPositions([]int{-1, 3, -3, 1, 3})
	want := "第 1 个、第 3 个、倒数第 3 个和最后一个"
	if got != want {
		t.Fatalf("renderSetPositions() = %q, want %q", got, want)
	}
}

func TestDescribeFrequencyAndPhaseMatrix(t *testing.T) {
	tests := []struct {
		name        string
		rule        string
		description string
		occurrences []time.Time
	}{
		{
			name:        "yearly defaults from dtstart",
			rule:        "DTSTART:20260727T091005Z\nRRULE:FREQ=YEARLY;COUNT=2",
			description: "每年7月，且各指定月份的第 27 天 09:10:05 执行",
			occurrences: []time.Time{time.Date(2026, 7, 27, 9, 10, 5, 0, time.UTC), time.Date(2027, 7, 27, 9, 10, 5, 0, time.UTC)},
		},
		{
			name:        "monthly defaults from dtstart",
			rule:        "DTSTART:20260727T091005Z\nRRULE:FREQ=MONTHLY;COUNT=2",
			description: "每月第 27 天 09:10:05 执行",
			occurrences: []time.Time{time.Date(2026, 7, 27, 9, 10, 5, 0, time.UTC), time.Date(2026, 8, 27, 9, 10, 5, 0, time.UTC)},
		},
		{
			name:        "weekly defaults from dtstart",
			rule:        "DTSTART:20260727T091005Z\nRRULE:FREQ=WEEKLY;COUNT=2",
			description: "每周周一 09:10:05 执行",
			occurrences: []time.Time{time.Date(2026, 7, 27, 9, 10, 5, 0, time.UTC), time.Date(2026, 8, 3, 9, 10, 5, 0, time.UTC)},
		},
		{
			name:        "daily interval phase",
			rule:        "DTSTART:20260727T091005Z\nRRULE:FREQ=DAILY;INTERVAL=2;COUNT=2",
			description: "按 2 天间隔 09:10:05 执行",
			occurrences: []time.Time{time.Date(2026, 7, 27, 9, 10, 5, 0, time.UTC), time.Date(2026, 7, 29, 9, 10, 5, 0, time.UTC)},
		},
		{
			name:        "hourly interval phase",
			rule:        "DTSTART:20260727T091005Z\nRRULE:FREQ=HOURLY;INTERVAL=2;COUNT=2",
			description: "按 2 小时间隔 时间条件：分钟为 10，秒为 05 执行",
			occurrences: []time.Time{time.Date(2026, 7, 27, 9, 10, 5, 0, time.UTC), time.Date(2026, 7, 27, 11, 10, 5, 0, time.UTC)},
		},
		{
			name:        "minutely interval phase",
			rule:        "DTSTART:20260727T091005Z\nRRULE:FREQ=MINUTELY;INTERVAL=2;COUNT=2",
			description: "按 2 分钟间隔 时间条件：秒为 05 执行",
			occurrences: []time.Time{time.Date(2026, 7, 27, 9, 10, 5, 0, time.UTC), time.Date(2026, 7, 27, 9, 12, 5, 0, time.UTC)},
		},
		{
			name:        "secondly interval phase",
			rule:        "DTSTART:20260727T091005Z\nRRULE:FREQ=SECONDLY;INTERVAL=2;COUNT=2",
			description: "按 2 秒间隔执行",
			occurrences: []time.Time{time.Date(2026, 7, 27, 9, 10, 5, 0, time.UTC), time.Date(2026, 7, 27, 9, 10, 7, 0, time.UTC)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description, err := Describe(tt.rule)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(description, tt.description) {
				t.Fatalf("description = %q, want substring %q", description, tt.description)
			}

			set, err := rrule.StrToRRuleSet(tt.rule)
			if err != nil {
				t.Fatal(err)
			}
			got := set.All()
			if len(got) != len(tt.occurrences) {
				t.Fatalf("occurrences = %v, want %v", got, tt.occurrences)
			}
			for i := range tt.occurrences {
				if !got[i].Equal(tt.occurrences[i]) {
					t.Fatalf("occurrences[%d] = %v, want %v", i, got[i], tt.occurrences[i])
				}
			}
		})
	}
}

func TestDescribeCountAndUntilIsAnUpperBound(t *testing.T) {
	rule := "DTSTART:20260727T090000Z\n" +
		"RRULE:FREQ=DAILY;COUNT=10;UNTIL=20260728T090000Z"

	description, err := Describe(rule)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(description, "周期规则最多生成 10 次") {
		t.Fatalf("Describe() = %q, want COUNT upper-bound wording", description)
	}
	if strings.Contains(description, "共执行 10 次") || strings.Contains(description, "周期规则生成 10 次") {
		t.Fatalf("Describe() = %q, must not claim ten generated occurrences", description)
	}

	set, err := rrule.StrToRRuleSet(rule)
	if err != nil {
		t.Fatal(err)
	}
	occurrences := set.All()
	if len(occurrences) != 2 {
		t.Fatalf("actual occurrence count = %d, want 2: %v", len(occurrences), occurrences)
	}
}

func TestDescribePreservesDistinctDatesAcrossDSTFold(t *testing.T) {
	rule := "DTSTART;TZID=America/New_York:20261031T000000\n" +
		"RRULE:FREQ=DAILY;COUNT=1\n" +
		"RDATE:20261101T053000Z,20261101T063000Z,20261101T053000Z"

	description, err := Describe(rule)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"2026-11-01 01:30:00 -04:00",
		"2026-11-01 01:30:00 -05:00",
	} {
		if strings.Count(description, want) != 1 {
			t.Fatalf("Describe() = %q, want exactly one %q", description, want)
		}
	}
}

func TestDescribeWeeklyWKSTMatchesOccurrences(t *testing.T) {
	tests := []struct {
		name string
		wkst string
		want []time.Time
	}{
		{
			name: "sunday week start",
			wkst: "SU",
			want: []time.Time{
				time.Date(2024, 1, 14, 9, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 28, 9, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 29, 9, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "monday week start",
			wkst: "MO",
			want: []time.Time{
				time.Date(2024, 1, 7, 9, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 21, 9, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 29, 9, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "default monday week start",
			want: []time.Time{
				time.Date(2024, 1, 7, 9, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 21, 9, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 29, 9, 0, 0, 0, time.UTC),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wkstPart := ""
			if tt.wkst != "" {
				wkstPart = ";WKST=" + tt.wkst
			}
			rule := "DTSTART:20240103T090000Z\nRRULE:FREQ=WEEKLY;INTERVAL=2;COUNT=8" + wkstPart + ";BYDAY=SU,MO"
			description, err := Describe(rule)
			if err != nil {
				t.Fatal(err)
			}
			wantStart := map[string]string{"SU": "以周日为一周起始", "MO": "以周一为一周起始", "": "以周一为一周起始"}[tt.wkst]
			if !strings.Contains(description, wantStart) {
				t.Fatalf("description = %q, want %q", description, wantStart)
			}
			set, err := rrule.StrToRRuleSet(rule)
			if err != nil {
				t.Fatal(err)
			}
			got := set.Between(tt.want[0].Add(-24*time.Hour), tt.want[len(tt.want)-1].Add(24*time.Hour), true)
			if len(got) < len(tt.want) {
				t.Fatalf("occurrences = %v, want at least %v", got, tt.want)
			}
			for i := range tt.want {
				if !got[i].Equal(tt.want[i]) {
					t.Fatalf("occurrences[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestDescribeDSTGapUsesLibraryOccurrenceAndWarns(t *testing.T) {
	rule := "DTSTART;TZID=America/New_York:20240309T023000\nRRULE:FREQ=DAILY;COUNT=4"
	description, err := Describe(rule)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"每天 02:30 执行",
		"时区：America/New_York；遇不存在或重复的本地时间，以时区库实际解析结果为准",
	} {
		if !strings.Contains(description, want) {
			t.Fatalf("description = %q, want %q", description, want)
		}
	}
	set, err := rrule.StrToRRuleSet(rule)
	if err != nil {
		t.Fatal(err)
	}
	got := set.All()
	want := []time.Time{
		time.Date(2024, 3, 9, 2, 30, 0, 0, time.FixedZone("EST", -5*60*60)),
		time.Date(2024, 3, 10, 1, 30, 0, 0, time.FixedZone("EST", -5*60*60)),
		time.Date(2024, 3, 11, 2, 30, 0, 0, time.FixedZone("EDT", -4*60*60)),
		time.Date(2024, 3, 12, 2, 30, 0, 0, time.FixedZone("EDT", -4*60*60)),
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Fatalf("occurrences[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestDescribeDateFilterAndRDateBoundarySemantics(t *testing.T) {
	tests := []struct {
		rule        string
		want        string
		occurrences []time.Time
	}{
		{
			rule: "DTSTART:20240115T090000Z\nRRULE:FREQ=YEARLY;COUNT=4;BYMONTHDAY=1",
			want: "每年各月的第 1 天",
			occurrences: []time.Time{
				time.Date(2024, 2, 1, 9, 0, 0, 0, time.UTC), time.Date(2024, 3, 1, 9, 0, 0, 0, time.UTC),
				time.Date(2024, 4, 1, 9, 0, 0, 0, time.UTC), time.Date(2024, 5, 1, 9, 0, 0, 0, time.UTC),
			},
		},
		{
			rule: "DTSTART:20240115T090000Z\nRRULE:FREQ=YEARLY;COUNT=4;BYDAY=MO",
			want: "每年每个周一",
			occurrences: []time.Time{
				time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC), time.Date(2024, 1, 22, 9, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 29, 9, 0, 0, 0, time.UTC), time.Date(2024, 2, 5, 9, 0, 0, 0, time.UTC),
			},
		},
		{
			rule: "DTSTART:20240115T090000Z\nRRULE:FREQ=DAILY;COUNT=4;BYMONTHDAY=1",
			want: "以每天为周期生成候选，并仅限每月第 1 天",
			occurrences: []time.Time{
				time.Date(2024, 2, 1, 9, 0, 0, 0, time.UTC), time.Date(2024, 3, 1, 9, 0, 0, 0, time.UTC),
				time.Date(2024, 4, 1, 9, 0, 0, 0, time.UTC), time.Date(2024, 5, 1, 9, 0, 0, 0, time.UTC),
			},
		},
	}
	for _, tt := range tests {
		description, err := Describe(tt.rule)
		if err != nil || !strings.Contains(description, tt.want) {
			t.Fatalf("Describe(%q) = %q, %v; want %q", tt.rule, description, err, tt.want)
		}
		set, err := rrule.StrToRRuleSet(tt.rule)
		if err != nil {
			t.Fatal(err)
		}
		got := set.All()
		for i := range tt.occurrences {
			if !got[i].Equal(tt.occurrences[i]) {
				t.Fatalf("occurrences[%d] = %v, want %v", i, got[i], tt.occurrences[i])
			}
		}
	}
	rule := "DTSTART:20240110T090000Z\nRRULE:FREQ=DAILY;UNTIL=20240112T090000Z\nRDATE:20240101T090000Z,20240201T090000Z"
	description, err := Describe(rule)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(description, "周期规则有效期：2024-01-10 09:00:00 至 2024-01-12 09:00:00") ||
		!strings.Contains(description, "额外纳入候选：2024-01-01 09:00:00 +00:00、2024-02-01 09:00:00 +00:00") {
		t.Fatalf("description = %q, want rule-only boundary and out-of-bound RDATE", description)
	}
}

func TestDescribeFollowsLibraryNormalization(t *testing.T) {
	for _, rule := range []string{
		testRRuleSet("FREQ=DAILY;COUNT=0;INTERVAL=0;BYDAY=0MO"),
		testRRuleSet("FREQ=DAILY;COUNT=-1"),
		testRRuleSet("FREQ=DAILY;COUNT=2;UNTIL=20260728T000000Z"),
	} {
		if description, err := Describe(rule); err != nil || description == "" {
			t.Fatalf("Describe(%q) = %q, %v; want library-normalized description", rule, description, err)
		}
	}
}

func TestDescribeOccurrenceDifferentials(t *testing.T) {
	tests := []struct {
		name        string
		rule        string
		description string
		occurrences []string
	}{
		{
			name: "hourly fixed times retain date filters and dtstart cutoff",
			rule: "DTSTART:20260727T120000Z\n" +
				"RRULE:FREQ=HOURLY;COUNT=5;BYMONTH=7;BYMONTHDAY=27,28;BYDAY=MO,TU;BYHOUR=9,13;BYMINUTE=30;BYSECOND=0",
			description: "以每天为周期生成候选，并仅限7月，且仅限每月第 27 天、第 28 天，且仅限周一、周二 09:30、13:30 执行",
			occurrences: []string{
				"2026-07-27T13:30:00Z",
				"2026-07-28T09:30:00Z",
				"2026-07-28T13:30:00Z",
				"2027-07-27T09:30:00Z",
				"2027-07-27T13:30:00Z",
			},
		},
		{
			name: "yearly ordinal weekday is scoped to year with monthday intersection",
			rule: "DTSTART:20240101T090000Z\n" +
				"RRULE:FREQ=YEARLY;COUNT=3;BYMONTHDAY=1,2,3,4,5,6,7;BYDAY=1MO;BYHOUR=9;BYMINUTE=0;BYSECOND=0",
			description: "仅限第 1 个周一",
			occurrences: []string{
				"2024-01-01T09:00:00Z",
				"2025-01-06T09:00:00Z",
				"2026-01-05T09:00:00Z",
			},
		},
		{
			name: "yearly ordinal weekday is scoped to each selected month",
			rule: "DTSTART:20240101T090000Z\n" +
				"RRULE:FREQ=YEARLY;COUNT=4;BYMONTH=1,2;BYMONTHDAY=1,2,3,4,5,6,7;BYDAY=1MO;BYHOUR=9;BYMINUTE=0;BYSECOND=0",
			description: "仅限各指定月份的第 1 个周一",
			occurrences: []string{
				"2024-01-01T09:00:00Z",
				"2024-02-05T09:00:00Z",
				"2025-01-06T09:00:00Z",
				"2025-02-03T09:00:00Z",
			},
		},
		{
			name: "bysetpos indexes the date and time cartesian product",
			rule: "DTSTART:20260701T000000Z\n" +
				"RRULE:FREQ=MONTHLY;COUNT=4;BYDAY=MO,TU;BYHOUR=9,17;BYMINUTE=0;BYSECOND=0;BYSETPOS=2,-1",
			description: "每个周期先按上述条件形成候选，再选择第 2 个和最后一个执行",
			occurrences: []string{
				"2026-07-06T17:00:00Z",
				"2026-07-28T17:00:00Z",
				"2026-08-03T17:00:00Z",
				"2026-08-31T17:00:00Z",
			},
		},
		{
			name: "weekly bysetpos uses wkst even without interval",
			rule: "DTSTART:20240101T090000Z\n" +
				"RRULE:FREQ=WEEKLY;COUNT=3;WKST=SU;BYDAY=SU,MO;BYSETPOS=1;BYHOUR=9;BYMINUTE=0;BYSECOND=0",
			description: "以周日为一周起始，每周周一、周日 09:00；每个周期先按上述条件形成候选，再选择第 1 个执行",
			occurrences: []string{
				"2024-01-01T09:00:00Z",
				"2024-01-07T09:00:00Z",
				"2024-01-14T09:00:00Z",
			},
		},
		{
			name: "yearly bymonth supplies a describable bysetpos candidate set",
			rule: "DTSTART:20240115T090000Z\n" +
				"RRULE:FREQ=YEARLY;COUNT=3;BYMONTH=1,2;BYSETPOS=1;BYHOUR=9;BYMINUTE=0;BYSECOND=0",
			description: "每年1月、2月，且各指定月份的第 15 天 09:00；每个周期先按上述条件形成候选，再选择第 1 个执行",
			occurrences: []string{
				"2024-01-15T09:00:00Z",
				"2025-01-15T09:00:00Z",
				"2026-01-15T09:00:00Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description, err := Describe(tt.rule)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(description, tt.description) {
				t.Fatalf("description = %q, want %q", description, tt.description)
			}

			set, err := rrule.StrToRRuleSet(tt.rule)
			if err != nil {
				t.Fatal(err)
			}
			got := set.All()
			if len(got) != len(tt.occurrences) {
				t.Fatalf("occurrences = %v, want %v", got, tt.occurrences)
			}
			for i, want := range tt.occurrences {
				if got[i].Format(time.RFC3339) != want {
					t.Fatalf("occurrences[%d] = %s, want %s", i, got[i].Format(time.RFC3339), want)
				}
			}
		})
	}
}

func TestDescribeOptionFieldMatrix(t *testing.T) {
	tests := []struct {
		name        string
		rule        string
		want        string
		unsupported bool
	}{
		{name: "freq yearly and dtstart", rule: "DTSTART:20260727T091005Z\nRRULE:FREQ=YEARLY;COUNT=1", want: "每年"},
		{name: "freq monthly and interval", rule: testRRuleSet("FREQ=MONTHLY;INTERVAL=2;COUNT=1"), want: "按 2 个月间隔"},
		{name: "freq weekly and wkst", rule: testRRuleSet("FREQ=WEEKLY;INTERVAL=2;WKST=SU;COUNT=1"), want: "以周日为一周起始"},
		{name: "freq daily and until", rule: "DTSTART:20260727T000000Z\nRRULE:FREQ=DAILY;UNTIL=20260728T000000Z", want: "周期规则有效期"},
		{name: "freq hourly", rule: testRRuleSet("FREQ=HOURLY;COUNT=1"), want: "每小时"},
		{name: "freq minutely", rule: testRRuleSet("FREQ=MINUTELY;COUNT=1"), want: "每分钟"},
		{name: "freq secondly", rule: testRRuleSet("FREQ=SECONDLY;COUNT=1"), want: "每秒"},
		{name: "bysetpos", rule: testRRuleSet("FREQ=MONTHLY;COUNT=1;BYDAY=MO;BYSETPOS=1"), want: "第 1 个"},
		{name: "bymonth", rule: testRRuleSet("FREQ=YEARLY;COUNT=1;BYMONTH=2"), want: "2月"},
		{name: "bymonthday", rule: testRRuleSet("FREQ=MONTHLY;COUNT=1;BYMONTHDAY=-1"), want: "最后一天"},
		{name: "byweekday", rule: testRRuleSet("FREQ=WEEKLY;COUNT=1;BYDAY=FR"), want: "周五"},
		{name: "clock dimensions", rule: testRRuleSet("FREQ=DAILY;COUNT=1;BYHOUR=9;BYMINUTE=10;BYSECOND=5"), want: "09:10:05"},
		{name: "byyearday", rule: testRRuleSet("FREQ=YEARLY;BYYEARDAY=100"), unsupported: true},
		{name: "byweekno", rule: testRRuleSet("FREQ=YEARLY;BYWEEKNO=2"), unsupported: true},
		{name: "byeaster", rule: testRRuleSet("FREQ=YEARLY;BYEASTER=1"), unsupported: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description, err := Describe(tt.rule)
			if tt.unsupported {
				if !errors.Is(err, ErrUnsupportedDescription) {
					t.Fatalf("error = %v, want ErrUnsupportedDescription", err)
				}
				return
			}
			if err != nil || !strings.Contains(description, tt.want) {
				t.Fatalf("Describe() = %q, %v; want %q", description, err, tt.want)
			}
		})
	}
}

func TestDescribeYearlyWordingDoesNotRepeatScope(t *testing.T) {
	for _, rule := range []string{
		"DTSTART:20240101T090000Z\nRRULE:FREQ=YEARLY;BYMONTHDAY=1;COUNT=1",
		"DTSTART:20240101T090000Z\nRRULE:FREQ=YEARLY;BYMONTHDAY=1,2,3,4,5,6,7;BYDAY=1MO;COUNT=1",
	} {
		description, err := Describe(rule)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(description, "每年 每年内") {
			t.Fatalf("description repeats YEARLY scope: %q", description)
		}
	}
}

func TestDescribeGroupsCommonWeekdaySets(t *testing.T) {
	tests := []struct {
		name        string
		rule        string
		want        string
		occurrences []string
	}{
		{
			name: "weekly weekdays",
			rule: testRRuleSet("FREQ=WEEKLY;COUNT=5;BYDAY=MO,TU,WE,TH,FR;BYHOUR=9;BYMINUTE=0;BYSECOND=0"),
			want: "每周工作日 09:00 执行",
			occurrences: []string{
				"2026-07-27T09:00:00Z", "2026-07-28T09:00:00Z", "2026-07-29T09:00:00Z",
				"2026-07-30T09:00:00Z", "2026-07-31T09:00:00Z",
			},
		},
		{
			name: "daily weekdays filter",
			rule: testRRuleSet("FREQ=DAILY;COUNT=5;BYDAY=MO,TU,WE,TH,FR;BYHOUR=9;BYMINUTE=0;BYSECOND=0"),
			want: "以每天为周期生成候选，并仅限工作日 09:00 执行",
			occurrences: []string{
				"2026-07-27T09:00:00Z", "2026-07-28T09:00:00Z", "2026-07-29T09:00:00Z",
				"2026-07-30T09:00:00Z", "2026-07-31T09:00:00Z",
			},
		},
		{
			name: "daily all weekdays remains visible",
			rule: testRRuleSet("FREQ=DAILY;COUNT=3;BYDAY=MO,TU,WE,TH,FR,SA,SU;BYHOUR=9;BYMINUTE=0;BYSECOND=0"),
			want: "每天，仅限周一至周日 09:00 执行",
			occurrences: []string{
				"2026-07-27T09:00:00Z", "2026-07-28T09:00:00Z", "2026-07-29T09:00:00Z",
			},
		},
		{
			name: "weekly weekend",
			rule: testRRuleSet("FREQ=WEEKLY;COUNT=2;BYDAY=SA,SU;BYHOUR=9;BYMINUTE=0;BYSECOND=0"),
			want: "每周周末 09:00 执行",
			occurrences: []string{
				"2026-08-01T09:00:00Z", "2026-08-02T09:00:00Z",
			},
		},
		{
			name: "yearly weekdays",
			rule: testRRuleSet("FREQ=YEARLY;BYDAY=MO,TU,WE,TH,FR;BYHOUR=9;BYMINUTE=0;BYSECOND=0;COUNT=5"),
			want: "每年工作日 09:00 执行",
			occurrences: []string{
				"2026-07-27T09:00:00Z", "2026-07-28T09:00:00Z", "2026-07-29T09:00:00Z",
				"2026-07-30T09:00:00Z", "2026-07-31T09:00:00Z",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description, err := Describe(tt.rule)
			if err != nil || !strings.Contains(description, tt.want) {
				t.Fatalf("Describe() = %q, %v; want %q", description, err, tt.want)
			}
			set, err := rrule.StrToRRuleSet(tt.rule)
			if err != nil {
				t.Fatal(err)
			}
			got := set.All()
			if len(got) != len(tt.occurrences) {
				t.Fatalf("occurrences = %v, want %v", got, tt.occurrences)
			}
			for i, want := range tt.occurrences {
				if got[i].Format(time.RFC3339) != want {
					t.Fatalf("occurrences[%d] = %s, want %s", i, got[i].Format(time.RFC3339), want)
				}
			}
		})
	}
}
