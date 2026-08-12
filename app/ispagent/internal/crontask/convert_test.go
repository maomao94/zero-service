package crontask

import (
	"reflect"
	"testing"
	"time"

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
	lastScheduledRun := carbon.Now().SubHours(2).StdTime()

	cfg := &crontask.TaskConfig{
		TaskCode:         f.TaskCode,
		TaskName:         f.TaskName,
		RRuleStr:         f.ToRRuleStr(),
		Priority:         f.ToPriority(),
		LockTimeout:      90 * time.Second,
		Status:           f.ToStatus(),
		NextRun:          nextRun,
		LastRun:          lastRun,
		LastScheduledRun: lastScheduledRun,
		Extra:            []byte(extra),
	}

	gorm := fromTaskConfig(cfg)
	if got := toFields(gorm); !reflect.DeepEqual(got, f) {
		t.Fatalf("flattened fields mismatch: got %+v, want %+v", got, f)
	}
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
	if back.LockTimeout != cfg.LockTimeout {
		t.Fatalf("round-trip lock timeout mismatch: %v vs %v", back.LockTimeout, cfg.LockTimeout)
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
	if !back.LastScheduledRun.Equal(lastScheduledRun) {
		t.Fatalf("round-trip last_scheduled_run mismatch: %v", back.LastScheduledRun)
	}

	parsed := DeserializeExtra(string(back.Extra))
	if !reflect.DeepEqual(parsed, f) {
		t.Fatalf("runtime extra mismatch: got %+v, want %+v", parsed, f)
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
	if gorm.LastScheduledRun.Valid {
		t.Fatalf("expected invalid SQL last scheduled run, got %v", gorm.LastScheduledRun)
	}
	back := toTaskConfig(gorm)
	if !back.NextRun.IsZero() {
		t.Fatalf("expected zero next run, got %v", back.NextRun)
	}
	if !back.LastRun.IsZero() {
		t.Fatalf("expected zero last run, got %v", back.LastRun)
	}
	if !back.LastScheduledRun.IsZero() {
		t.Fatalf("expected zero last scheduled run, got %v", back.LastScheduledRun)
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
