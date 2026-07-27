package logic

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"zero-service/app/ispagent/model/gormmodel"
)

func TestToTaskConfigItemPbIncludesSchedule(t *testing.T) {
	nextRun := time.Date(2026, 7, 28, 9, 30, 0, 0, time.Local)
	record := &gormmodel.GormTaskConfig{
		TaskCode: "TASK001",
		RRuleStr: "DTSTART;TZID=Asia/Shanghai:20260727T000000\n" +
			"RRULE:FREQ=DAILY;BYHOUR=9;BYMINUTE=30;BYSECOND=0",
		NextRun: sql.NullTime{Time: nextRun, Valid: true},
	}
	item, err := toTaskConfigItemPb(record)
	if err != nil {
		t.Fatal(err)
	}
	if item.RruleStr != record.RRuleStr || !strings.Contains(item.ScheduleDescription, "每天 09:30 执行") {
		t.Fatalf("unexpected schedule view: %+v", item)
	}
	if item.NextRun == "" {
		t.Fatal("nextRun must be converted")
	}
}
