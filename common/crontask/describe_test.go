package crontask

import (
	"errors"
	"strings"
	"testing"
)

func TestDescribeRRule(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		contains []string
	}{
		{
			name: "daily with timezone and until",
			rule: "DTSTART;TZID=Asia/Shanghai:20260727T000000\nRRULE:FREQ=DAILY;UNTIL=20261231T155959Z;BYHOUR=9;BYMINUTE=30;BYSECOND=0",
			contains: []string{
				"每天 09:30 执行",
				"有效期：2026-07-27 00:00:00 至 2026-12-31 23:59:59",
			},
		},
		{
			name:     "monthly negative day interval",
			rule:     testRRuleSet("FREQ=MONTHLY;INTERVAL=2;BYMONTHDAY=-1,-3;BYHOUR=8;BYMINUTE=0;BYSECOND=0"),
			contains: []string{"每 2 个月", "倒数第 3 天、最后一天", "08:00 执行"},
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
			contains: []string{"每 3 小时", "分钟=05", "秒=10", "执行"},
		},
		{
			name:     "minutely",
			rule:     testRRuleSet("FREQ=MINUTELY;INTERVAL=10;BYSECOND=5"),
			contains: []string{"每 10 分钟", "秒=05", "执行"},
		},
		{
			name:     "cartesian times",
			rule:     testRRuleSet("FREQ=DAILY;BYHOUR=8,9;BYMINUTE=0,30;BYSECOND=0"),
			contains: []string{"08:00、08:30、09:00、09:30 执行"},
		},
		{
			name:     "yearly filters and count",
			rule:     testRRuleSet("FREQ=YEARLY;INTERVAL=2;BYMONTH=1,6;BYMONTHDAY=1;COUNT=4;BYHOUR=0;BYMINUTE=0;BYSECOND=0"),
			contains: []string{"每 2 年", "1月、6月，且第 1 天", "共执行 4 次"},
		},
		{
			name: "rrule set dates use dtstart timezone",
			rule: "DTSTART;TZID=Asia/Shanghai:20260727T090000\nRRULE:FREQ=DAILY;COUNT=2\n" +
				"RDATE:20260730T100000\nEXDATE:20260728T090000",
			contains: []string{
				"每天 09:00 执行", "周期规则生成 2 次",
				"额外执行：2026-07-30 10:00:00", "排除执行：2026-07-28 09:00:00",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DescribeRRule(tt.rule)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Fatalf("DescribeRRule() = %q, want substring %q", got, want)
				}
			}
		})
	}
}

func TestDescribeRRuleEmpty(t *testing.T) {
	got, err := DescribeRRule("")
	if err != nil || got != "" {
		t.Fatalf("DescribeRRule(empty) = %q, %v", got, err)
	}
}

func TestDescribeRRuleSupportsCRLFSet(t *testing.T) {
	description, err := DescribeRRule("DTSTART;TZID=Asia/Shanghai:20260727T090000\r\n" +
		"RRULE:FREQ=DAILY;COUNT=2\r\n" +
		"RDATE:20260730T100000\r\n" +
		"EXDATE:20260728T090000")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"每天 09:00 执行",
		"额外执行：2026-07-30 10:00:00",
		"排除执行：2026-07-28 09:00:00",
	} {
		if !strings.Contains(description, want) {
			t.Fatalf("description = %q, want substring %q", description, want)
		}
	}
}

func TestDescribeRRuleErrors(t *testing.T) {
	if _, err := DescribeRRule("not-an-rrule"); err == nil {
		t.Fatal("invalid rule should fail")
	}
	if _, err := DescribeRRule("FREQ=DAILY;BYHOUR=9;BYMINUTE=0"); err == nil {
		t.Fatal("bare RRULE should fail")
	}
	for _, rule := range []string{
		testRRuleSet("FREQ=YEARLY;BYYEARDAY=100"),
		testRRuleSet("FREQ=YEARLY;BYWEEKNO=2"),
		testRRuleSet("FREQ=YEARLY;BYEASTER=1"),
		testRRuleSet("FREQ=MONTHLY;BYSETPOS=1"),
		testRRuleSet("FREQ=WEEKLY;BYDAY=1MO"),
	} {
		if _, err := DescribeRRule(rule); !errors.Is(err, ErrUnsupportedDescription) {
			t.Fatalf("DescribeRRule(%q) error = %v, want ErrUnsupportedDescription", rule, err)
		}
	}
}
