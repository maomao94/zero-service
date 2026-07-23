package crontask

import (
	"testing"

	"zero-service/app/ispagent/model/gormmodel"
	"zero-service/common/crontask"

	"github.com/dromara/carbon/v2"
)

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

	nextRun := carbon.Now().AddDay().StdTime()
	lastRun := carbon.Now().SubHour().StdTime()

	cfg := &crontask.TaskConfig{
		TaskCode: f.TaskCode,
		TaskName: f.TaskName,
		RRuleStr: f.ToRRuleStr(),
		Priority: f.ToPriority(),
		Status:   f.ToStatus(),
		NextRun:  nextRun,
		LastRun:  lastRun,
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
	if !back.NextRun.Equal(nextRun) {
		t.Fatalf("round-trip next_run mismatch: %v", back.NextRun)
	}
	if !back.LastRun.Equal(lastRun) {
		t.Fatalf("round-trip last_run mismatch: %v", back.LastRun)
	}

	parsed := DeserializeExtra(string(back.Extra))
	if parsed.Creator != f.Creator {
		t.Fatal("round-trip creator mismatch")
	}
}

func TestConvertRoundTripZeroNextRun(t *testing.T) {
	cfg := &crontask.TaskConfig{
		TaskCode: "exhausted",
		TaskName: "已结束任务",
		Status:   crontask.StatusEnabled,
	}

	gorm := fromTaskConfig(cfg)
	if gorm.NextRun.Valid {
		t.Fatalf("expected invalid SQL time, got %v", gorm.NextRun)
	}
	if gorm.LastRun.Valid {
		t.Fatalf("expected invalid SQL last run, got %v", gorm.LastRun)
	}
	back := toTaskConfig(gorm)
	if !back.NextRun.IsZero() {
		t.Fatalf("expected zero next run, got %v", back.NextRun)
	}
	if !back.LastRun.IsZero() {
		t.Fatalf("expected zero last run, got %v", back.LastRun)
	}
}

func TestToFieldsRoundTripPriority(t *testing.T) {
	g := &gormmodel.GormTaskConfig{
		TaskCode:   "test-code",
		TaskName:   "test-name",
		Priority:   2,
		IsEnable:   "0",
		IspCreator: "creator-1",
	}
	f := toFields(g)
	if f.Priority != "2" {
		t.Fatalf("expected Priority='2', got '%s'", f.Priority)
	}
	if f.Creator != g.IspCreator {
		t.Fatalf("expected Creator='%s', got '%s'", g.IspCreator, f.Creator)
	}

	// round-trip: from fields back to GormTaskConfig via applyFields
	g2 := &gormmodel.GormTaskConfig{}
	applyFields(g2, f)
	if g2.IspCreator != g.IspCreator {
		t.Fatalf("applyFields round-trip creator mismatch")
	}
}
