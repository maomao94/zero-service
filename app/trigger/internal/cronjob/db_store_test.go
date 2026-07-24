package cronjob

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/crontask"
	"zero-service/common/gormx"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDBStoreClaimCompleteAndExtraRoundTrip(t *testing.T) {
	db := newCronJobTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	now := time.Date(2026, 7, 24, 10, 0, 0, 0, time.Local)
	ruleJSON, _ := json.Marshal(&trigger.PlanRulePb{Freq: 3, Hours: []int32{11}, Minutes: []int32{0}})
	bizExtra := json.RawMessage(`{"source":"test"}`)
	extra, err := MarshalExtra(&CronJobExtra{
		DeptCode:     "D001",
		Type:         "inspection",
		StartTime:    "2026-07-01 00:00:00",
		EndTime:      "2026-07-31 23:59:59",
		Rule:         ruleJSON,
		ExcludeDates: []string{"2026-07-26"},
		BizExtra:     bizExtra,
		Ext1:         "ext",
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg := &crontask.TaskConfig{
		TaskCode:    "CRON001",
		TaskName:    "测试周期任务",
		RRuleStr:    "FREQ=DAILY",
		Priority:    5,
		LockTimeout: 2 * time.Minute,
		Payload:     json.RawMessage(`{"id":1}`),
		Extra:       extra,
		Status:      crontask.StatusEnabled,
		NextRun:     now.Add(-time.Minute),
	}
	if err := store.Insert(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.ID == "" {
		t.Fatal("expected generated JobId")
	}

	claim, err := store.LockAndFetch(context.Background(), now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !claim.Task.NextRun.Equal(now.Add(-time.Minute)) {
		t.Fatalf("scheduled time = %v, want original due time", claim.Task.NextRun)
	}
	if claim.Task.LockTimeout != 2*time.Minute {
		t.Fatalf("lock timeout = %v, want %v", claim.Task.LockTimeout, 2*time.Minute)
	}
	if want := now.Add(2 * time.Minute); !claim.LockedUntil.Equal(want) {
		t.Fatalf("locked until = %v, want task-specific %v", claim.LockedUntil, want)
	}
	parsed, err := ParseExtra(claim.Task.Extra)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.DeptCode != "D001" || string(parsed.BizExtra) != string(bizExtra) || len(parsed.ExcludeDates) != 1 {
		t.Fatalf("unexpected rebuilt extra: %+v", parsed)
	}

	if err := store.Complete(context.Background(), cfg.ID, claim.LockedUntil, time.Time{}, now); err != nil {
		t.Fatal(err)
	}
	var job gormmodel.CronJob
	if err := db.Where("id = ?", cfg.ID).First(&job).Error; err != nil {
		t.Fatal(err)
	}
	if job.NextRun.Valid {
		t.Fatalf("expected SQL NULL next_run, got %v", job.NextRun)
	}
	if !job.LastRun.Valid || !job.LastRun.Time.Equal(now) {
		t.Fatalf("last run = %v, want %v", job.LastRun, now)
	}
	if !job.StartTime.Valid || !job.EndTime.Valid || !job.ExcludeDates.Valid {
		t.Fatalf("expected supplied business fields to be non-NULL: start=%v end=%v exclude=%v", job.StartTime, job.EndTime, job.ExcludeDates)
	}
	if job.LockTimeout != int64((2*time.Minute)/time.Millisecond) {
		t.Fatalf("persisted lock timeout = %d, want %d", job.LockTimeout, int64((2*time.Minute)/time.Millisecond))
	}
	if job.ScheduledTime.Valid {
		t.Fatalf("completed task must clear scheduled_time: %v", job.ScheduledTime)
	}
}

func TestDBStoreRetryKeepsOriginalScheduledTime(t *testing.T) {
	db := newCronJobTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	now := time.Date(2026, 7, 24, 10, 0, 0, 0, time.Local)
	originalScheduledTime := now.Add(-time.Minute)
	config := cronJobTestConfig(t, originalScheduledTime)
	if err := store.Insert(context.Background(), config); err != nil {
		t.Fatal(err)
	}

	firstClaim, err := store.LockAndFetch(context.Background(), now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !firstClaim.Task.NextRun.Equal(originalScheduledTime) {
		t.Fatalf("first scheduled time = %v, want %v", firstClaim.Task.NextRun, originalScheduledTime)
	}
	secondClaim, err := store.LockAndFetch(context.Background(), firstClaim.LockedUntil, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !secondClaim.Task.NextRun.Equal(originalScheduledTime) {
		t.Fatalf("retry scheduled time = %v, want stable %v", secondClaim.Task.NextRun, originalScheduledTime)
	}
	if err := store.Complete(context.Background(), config.ID, secondClaim.LockedUntil, now.Add(time.Hour), now); err != nil {
		t.Fatal(err)
	}

	var job gormmodel.CronJob
	if err := db.Where("id = ?", config.ID).First(&job).Error; err != nil {
		t.Fatal(err)
	}
	if job.ScheduledTime.Valid {
		t.Fatalf("completed retry must clear scheduled_time: %v", job.ScheduledTime)
	}
}

func TestDBStoreOptionalBusinessFieldsPersistAsNull(t *testing.T) {
	db := newCronJobTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	config := cronJobTestConfig(t, time.Now().Add(time.Hour))
	extra, err := ParseExtra(config.Extra)
	if err != nil {
		t.Fatal(err)
	}
	extra.StartTime = ""
	extra.EndTime = ""
	extra.ExcludeDates = nil
	config.Extra, err = MarshalExtra(extra)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Insert(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	var job gormmodel.CronJob
	if err := db.Where("id = ?", config.ID).First(&job).Error; err != nil {
		t.Fatal(err)
	}
	if job.StartTime.Valid || job.EndTime.Valid || job.ExcludeDates.Valid {
		t.Fatalf("optional fields must be SQL NULL: start=%v end=%v exclude=%v", job.StartTime, job.EndTime, job.ExcludeDates)
	}

	loaded, err := store.GetByID(context.Background(), config.ID)
	if err != nil {
		t.Fatal(err)
	}
	loadedExtra, err := ParseExtra(loaded.Extra)
	if err != nil {
		t.Fatal(err)
	}
	if loadedExtra.StartTime != "" || loadedExtra.EndTime != "" || len(loadedExtra.ExcludeDates) != 0 {
		t.Fatalf("unexpected optional field round trip: %+v", loadedExtra)
	}
}

func TestDBStoreCompleteRejectsLostClaim(t *testing.T) {
	db := newCronJobTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	now := time.Date(2026, 7, 24, 10, 0, 0, 0, time.Local)
	cfg := cronJobTestConfig(t, now.Add(-time.Minute))
	if err := store.Insert(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	claim, err := store.LockAndFetch(context.Background(), now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&gormmodel.CronJob{}).Where("id = ?", cfg.ID).Update("next_run", claim.LockedUntil.Add(time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	if err := store.Complete(context.Background(), cfg.ID, claim.LockedUntil, now.Add(time.Hour), now); !errors.Is(err, crontask.ErrNotFound) {
		t.Fatalf("expected lost claim, got %v", err)
	}
}

func TestDBStoreCompleteAllowsConcurrentDisable(t *testing.T) {
	db := newCronJobTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	now := time.Date(2026, 7, 24, 10, 0, 0, 0, time.Local)
	cfg := cronJobTestConfig(t, now.Add(-time.Minute))
	if err := store.Insert(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	claim, err := store.LockAndFetch(context.Background(), now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Disable(context.Background(), cfg.ID); err != nil {
		t.Fatal(err)
	}
	nextRun := now.Add(time.Hour)
	if err := store.Complete(context.Background(), cfg.ID, claim.LockedUntil, nextRun, now); err != nil {
		t.Fatalf("disabled in-flight task should complete: %v", err)
	}
	loaded, err := store.GetByID(context.Background(), cfg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != crontask.StatusDisabled || !loaded.NextRun.Equal(nextRun) || !loaded.LastRun.Equal(now) {
		t.Fatalf("unexpected completed disabled task: %+v", loaded)
	}
}

func TestDBStoreDisableIsIdempotentAndRejectsMissingJob(t *testing.T) {
	db := newCronJobTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	cfg := cronJobTestConfig(t, time.Now().Add(time.Hour))
	if err := store.Insert(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if err := store.Disable(context.Background(), cfg.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.Disable(context.Background(), cfg.ID); err != nil {
		t.Fatalf("repeated disable: %v", err)
	}
	if err := store.Disable(context.Background(), "missing"); !errors.Is(err, crontask.ErrUpdate) {
		t.Fatalf("disable missing job = %v, want ErrUpdate", err)
	}
}

func TestDBStoreEnableRecalculatesOnceAndClearsInFlightSchedule(t *testing.T) {
	db := newCronJobTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	now := time.Now()
	cfg := cronJobTestConfig(t, now.Add(-time.Minute))
	if err := store.Insert(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	claim, err := store.LockAndFetch(context.Background(), now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Disable(context.Background(), cfg.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.Enable(context.Background(), cfg.ID); err != nil {
		t.Fatal(err)
	}

	var enabled gormmodel.CronJob
	if err := db.Where("id = ?", cfg.ID).First(&enabled).Error; err != nil {
		t.Fatal(err)
	}
	if enabled.Status != int(crontask.StatusEnabled) || !enabled.NextRun.Valid || !enabled.NextRun.Time.After(now) {
		t.Fatalf("unexpected enabled job: %+v", enabled)
	}
	if enabled.ScheduledTime.Valid {
		t.Fatalf("enable must clear in-flight scheduled time: %v", enabled.ScheduledTime)
	}
	firstNextRun := enabled.NextRun.Time
	if err := store.Enable(context.Background(), cfg.ID); err != nil {
		t.Fatal(err)
	}
	enabled = gormmodel.CronJob{}
	if err := db.Where("id = ?", cfg.ID).First(&enabled).Error; err != nil {
		t.Fatal(err)
	}
	if !enabled.NextRun.Time.Equal(firstNextRun) {
		t.Fatalf("repeated enable changed next_run: first=%v second=%v", firstNextRun, enabled.NextRun.Time)
	}
	if err := store.Complete(context.Background(), cfg.ID, claim.LockedUntil, now.Add(time.Hour), now); !errors.Is(err, crontask.ErrNotFound) {
		t.Fatalf("enable should invalidate the previous claim, got %v", err)
	}
}

func TestDBStoreRejectsDuplicateTaskCode(t *testing.T) {
	db := newCronJobTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	now := time.Now()
	first := cronJobTestConfig(t, now)
	second := cronJobTestConfig(t, now)
	if err := store.Insert(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := store.Insert(context.Background(), second); !errors.Is(err, crontask.ErrDuplicate) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestDBStoreDeleteIsIdempotent(t *testing.T) {
	db := newCronJobTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	cfg := cronJobTestConfig(t, time.Now().Add(time.Hour))
	if err := store.Insert(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(context.Background(), cfg.ID); err != nil {
		t.Fatalf("first delete: %v", err)
	}
	if err := store.Delete(context.Background(), cfg.ID); err != nil {
		t.Fatalf("repeated delete: %v", err)
	}
}

func TestDBStoreListByStatuses(t *testing.T) {
	db := newCronJobTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	first := cronJobTestConfig(t, time.Now().Add(time.Hour))
	first.TaskCode = "LIST-ENABLED"
	second := cronJobTestConfig(t, time.Now().Add(time.Hour))
	second.TaskCode = "LIST-DISABLED"
	second.Status = crontask.StatusDisabled
	if err := store.Insert(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := store.Insert(context.Background(), second); err != nil {
		t.Fatal(err)
	}

	all, err := store.List(context.Background(), crontask.ListCondition{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("all jobs = %d, want 2", len(all))
	}
	enabled, err := store.List(context.Background(), crontask.ListCondition{Statuses: []crontask.TaskStatus{crontask.StatusEnabled}})
	if err != nil {
		t.Fatal(err)
	}
	if len(enabled) != 1 || enabled[0].TaskCode != first.TaskCode {
		t.Fatalf("unexpected enabled jobs: %+v", enabled)
	}
	both, err := store.List(context.Background(), crontask.ListCondition{
		Statuses: []crontask.TaskStatus{crontask.StatusEnabled, crontask.StatusDisabled},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(both) != 2 {
		t.Fatalf("both statuses = %d, want 2", len(both))
	}
}

func cronJobTestConfig(t *testing.T, nextRun time.Time) *crontask.TaskConfig {
	t.Helper()
	ruleJSON, _ := json.Marshal(&trigger.PlanRulePb{Freq: 3, Hours: []int32{11}, Minutes: []int32{0}})
	extra, err := MarshalExtra(&CronJobExtra{
		DeptCode:  "D001",
		Type:      "test",
		StartTime: "2026-07-01 00:00:00",
		EndTime:   "2026-07-31 23:59:59",
		Rule:      ruleJSON,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &crontask.TaskConfig{
		TaskCode: "SAME",
		TaskName: "same",
		RRuleStr: "FREQ=DAILY",
		Extra:    extra,
		Status:   crontask.StatusEnabled,
		NextRun:  nextRun,
	}
}

func newCronJobTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&parseTime=true&_loc=auto"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&gormmodel.CronJob{}); err != nil {
		t.Fatal(err)
	}
	return db
}
