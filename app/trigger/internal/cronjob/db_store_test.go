package cronjob

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/crontask"
	"zero-service/common/gormx"
	"zero-service/common/rrulex"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDBStoreClaimCompleteAndExtraRoundTrip(t *testing.T) {
	db := newCronJobTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	now := time.Date(2026, 7, 24, 10, 0, 0, 0, time.Local)
	ruleJSON, _ := json.Marshal(&trigger.PlanRulePb{Freq: 3, Hours: []int32{11}, Minutes: []int32{0}})
	extra, err := MarshalExtra(&CronJobExtra{
		DeptCode:     "D001",
		Type:         "inspection",
		Rule:         ruleJSON,
		ExcludeDates: []string{"2026-07-26"},
		Ext1:         "ext",
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg := &crontask.TaskConfig{
		TaskCode:    "CRON001",
		TaskName:    "测试周期任务",
		RRuleStr:    "DTSTART:20260701T000000Z\nRRULE:FREQ=DAILY",
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
	if !claim.Task.ScheduledTime.Equal(now.Add(-time.Minute)) {
		t.Fatalf("scheduled time = %v, want original due time", claim.Task.ScheduledTime)
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
	if parsed.DeptCode != "D001" || len(parsed.ExcludeDates) != 1 {
		t.Fatalf("unexpected rebuilt extra: %+v", parsed)
	}

	if err := store.Complete(context.Background(), cfg.ID, claim.LockedUntil, crontask.Completion{
		LastRun:          now,
		LastScheduledRun: claim.Task.ScheduledTime,
	}); err != nil {
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
	if !job.LastScheduledRun.Valid || !job.LastScheduledRun.Time.Equal(claim.Task.ScheduledTime) {
		t.Fatalf("last scheduled run = %v, want %v", job.LastScheduledRun, claim.Task.ScheduledTime)
	}
	loaded, err := store.GetByID(context.Background(), cfg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.LastScheduledRun.Equal(claim.Task.ScheduledTime) {
		t.Fatalf("loaded last scheduled run = %v, want %v", loaded.LastScheduledRun, claim.Task.ScheduledTime)
	}
	if !job.ExcludeDates.Valid {
		t.Fatalf("expected exclude dates to be non-NULL: %v", job.ExcludeDates)
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
	if !firstClaim.Task.ScheduledTime.Equal(originalScheduledTime) {
		t.Fatalf("first scheduled time = %v, want %v", firstClaim.Task.ScheduledTime, originalScheduledTime)
	}
	secondClaim, err := store.LockAndFetch(context.Background(), firstClaim.LockedUntil, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !secondClaim.Task.ScheduledTime.Equal(originalScheduledTime) {
		t.Fatalf("retry scheduled time = %v, want stable %v", secondClaim.Task.ScheduledTime, originalScheduledTime)
	}
	if err := store.Complete(context.Background(), config.ID, secondClaim.LockedUntil, crontask.Completion{
		NextRun:          now.Add(time.Hour),
		LastRun:          now,
		LastScheduledRun: secondClaim.Task.ScheduledTime,
	}); err != nil {
		t.Fatal(err)
	}

	var job gormmodel.CronJob
	if err := db.Where("id = ?", config.ID).First(&job).Error; err != nil {
		t.Fatal(err)
	}
	if job.ScheduledTime.Valid {
		t.Fatalf("completed retry must clear scheduled_time: %v", job.ScheduledTime)
	}
	if !job.LastScheduledRun.Valid || !job.LastScheduledRun.Time.Equal(originalScheduledTime) {
		t.Fatalf("last scheduled run = %v, want stable %v", job.LastScheduledRun, originalScheduledTime)
	}
}

func TestDBStoreCompletionProgressesFromPersistedExactTimeSet(t *testing.T) {
	db := newCronJobTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	now := time.Now().Truncate(time.Second)
	ruleCursor := now.UTC()
	specified := ruleCursor.Add(-time.Hour)
	excluded := ruleCursor.Add(time.Hour)
	wantNext := ruleCursor.Add(2 * time.Hour)
	cfg := cronJobTestConfig(t, now.Add(-time.Minute))
	cfg.TaskCode = "COMPLETE-EXACT-TIME"
	cfg.RRuleStr = "DTSTART:" + specified.Format("20060102T150405Z") + "\n" +
		"RRULE:FREQ=HOURLY;COUNT=4\n" +
		"RDATE:" + specified.Format("20060102T150405Z") + "\n" +
		"EXDATE:" + excluded.Format("20060102T150405Z")
	if err := store.Insert(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}

	claim, err := store.LockAndFetch(context.Background(), now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	set, err := rrulex.ParseSet(claim.Task.RRuleStr)
	if err != nil {
		t.Fatal(err)
	}
	nextRun := set.After(ruleCursor, false)
	if !nextRun.Equal(wantNext) {
		t.Fatalf("completion progression = %v, want %v after exact EXDATE", nextRun, wantNext)
	}
	if err := store.Complete(context.Background(), cfg.ID, claim.LockedUntil, crontask.Completion{
		NextRun:          nextRun,
		LastRun:          now,
		LastScheduledRun: claim.Task.ScheduledTime,
	}); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.GetByID(context.Background(), cfg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.NextRun.Equal(wantNext) || loaded.RRuleStr != cfg.RRuleStr {
		t.Fatalf("completed exact-time schedule changed unexpectedly: %+v", loaded)
	}
}

func TestDBStoreUpdateRejectsInFlightTask(t *testing.T) {
	db := newCronJobTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	now := time.Date(2026, 7, 24, 10, 0, 0, 0, time.Local)
	config := cronJobTestConfig(t, now.Add(-time.Minute))
	if err := store.Insert(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	claim, err := store.LockAndFetch(context.Background(), now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	claim.Task.TaskName = "updated"
	extra, err := ParseExtra(claim.Task.Extra)
	if err != nil {
		t.Fatal(err)
	}
	extra.SpecifiedTimes = []string{"2026-07-28 12:00:00"}
	extra.ExcludedTimes = []string{"2026-07-29 12:00:00"}
	claim.Task.Extra, err = MarshalExtra(extra)
	if err != nil {
		t.Fatal(err)
	}
	claim.Task.RRuleStr = "DTSTART:20260701T000000Z\nRRULE:FREQ=DAILY\nRDATE:20260728T120000Z\nEXDATE:20260729T120000Z"
	if err := store.Update(context.Background(), claim.Task); !errors.Is(err, crontask.ErrUpdate) {
		t.Fatalf("in-flight update = %v, want ErrUpdate", err)
	}

	var updated gormmodel.CronJob
	if err := db.Where("id = ?", config.ID).First(&updated).Error; err != nil {
		t.Fatal(err)
	}
	if !updated.ScheduledTime.Valid || !updated.ScheduledTime.Time.Equal(claim.Task.ScheduledTime) {
		t.Fatalf("scheduled_time = %v, want in-flight %v", updated.ScheduledTime, claim.Task.ScheduledTime)
	}
	if !updated.NextRun.Valid || !updated.NextRun.Time.Equal(claim.LockedUntil) {
		t.Fatalf("next_run = %v, want lease %v", updated.NextRun, claim.LockedUntil)
	}
	if updated.TaskName == claim.Task.TaskName {
		t.Fatalf("in-flight task configuration changed: %+v", updated)
	}
	if updated.SpecifiedTimes.Valid || updated.ExcludedTimes.Valid || strings.Contains(updated.RRuleStr, "RDATE") || strings.Contains(updated.RRuleStr, "EXDATE") {
		t.Fatalf("in-flight exact-time configuration changed partially: %+v", updated)
	}
}

func TestDBStoreUpdateOwnsOnlyConfigurationFields(t *testing.T) {
	db := newCronJobTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	now := time.Date(2026, 7, 24, 10, 0, 0, 0, time.Local)
	config := cronJobTestConfig(t, now.Add(time.Hour))
	if err := store.Insert(context.Background(), config); err != nil {
		t.Fatal(err)
	}

	lastRun := now.Add(-time.Hour)
	lastScheduledRun := now.Add(-2 * time.Hour)
	if err := db.Model(&gormmodel.CronJob{}).Where("id = ?", config.ID).Updates(map[string]interface{}{
		"status":             int(crontask.StatusDisabled),
		"last_run":           lastRun,
		"last_scheduled_run": lastScheduledRun,
	}).Error; err != nil {
		t.Fatal(err)
	}

	updated, err := store.GetByID(context.Background(), config.ID)
	if err != nil {
		t.Fatal(err)
	}
	updated.TaskCode = "must-not-change"
	updated.TaskName = "updated"
	updated.Payload = nil
	extra, err := ParseExtra(updated.Extra)
	if err != nil {
		t.Fatal(err)
	}
	extra.GroupId = "must-not-change"
	extra.DeptCode = "must-not-change"
	extra.Type = "must-not-change"
	extra.Description = ""
	extra.ExcludeDates = nil
	extra.Ext1 = ""
	updated.Extra, err = MarshalExtra(extra)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(context.Background(), updated); err != nil {
		t.Fatal(err)
	}

	var record gormmodel.CronJob
	if err := db.Where("id = ?", config.ID).First(&record).Error; err != nil {
		t.Fatal(err)
	}
	if record.TaskCode != config.TaskCode || record.GroupId != "G001" || record.DeptCode != "D001" || record.Type != "test" ||
		record.Status != int(crontask.StatusDisabled) {
		t.Fatalf("protected identity/state changed: %+v", record)
	}
	if !record.LastRun.Valid || !record.LastRun.Time.Equal(lastRun) ||
		!record.LastScheduledRun.Valid || !record.LastScheduledRun.Time.Equal(lastScheduledRun) {
		t.Fatalf("execution history changed: last_run=%v last_scheduled_run=%v", record.LastRun, record.LastScheduledRun)
	}
	if record.TaskName != "updated" || record.Payload != "" || record.Description != "" ||
		record.StartTime.Valid || record.EndTime.Valid || record.ExcludeDates.Valid || record.Ext1 != "" {
		t.Fatalf("configuration fields were not updated/cleared: %+v", record)
	}
}

func TestDBStoreUpdateMissingReturnsErrUpdate(t *testing.T) {
	store := NewDBStore(&gormx.DB{DB: newCronJobTestDB(t)})
	config := cronJobTestConfig(t, time.Now().Add(time.Hour))
	config.ID = "missing"
	if err := store.Update(context.Background(), config); !errors.Is(err, crontask.ErrUpdate) {
		t.Fatalf("missing update = %v, want ErrUpdate", err)
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
	extra.ExcludeDates = nil
	extra.SpecifiedTimes = nil
	extra.ExcludedTimes = nil
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
	if job.StartTime.Valid || job.EndTime.Valid || job.ExcludeDates.Valid || job.SpecifiedTimes.Valid || job.ExcludedTimes.Valid {
		t.Fatalf("optional fields must be SQL NULL: start=%v end=%v exclude=%v specified=%v excluded=%v", job.StartTime, job.EndTime, job.ExcludeDates, job.SpecifiedTimes, job.ExcludedTimes)
	}

	loaded, err := store.GetByID(context.Background(), config.ID)
	if err != nil {
		t.Fatal(err)
	}
	loadedExtra, err := ParseExtra(loaded.Extra)
	if err != nil {
		t.Fatal(err)
	}
	if len(loadedExtra.ExcludeDates) != 0 {
		t.Fatalf("unexpected optional field round trip: %+v", loadedExtra)
	}
}

func TestDBStoreExactTimesReplaceAndClear(t *testing.T) {
	db := newCronJobTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	config := cronJobTestConfig(t, time.Now().Add(time.Hour))
	extra, err := ParseExtra(config.Extra)
	if err != nil {
		t.Fatal(err)
	}
	extra.SpecifiedTimes = []string{"2026-07-25 09:00:00"}
	extra.ExcludedTimes = []string{"2026-07-26 09:00:00"}
	config.RRuleStr = "DTSTART:20260701T000000Z\nRRULE:FREQ=DAILY\nRDATE:20260725T090000Z\nEXDATE:20260726T090000Z"
	config.Extra, err = MarshalExtra(extra)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Insert(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.GetByID(context.Background(), config.ID)
	if err != nil {
		t.Fatal(err)
	}
	loadedExtra, err := ParseExtra(loaded.Extra)
	if err != nil {
		t.Fatal(err)
	}
	if len(loadedExtra.SpecifiedTimes) != 1 || len(loadedExtra.ExcludedTimes) != 1 {
		t.Fatalf("exact times did not round trip: %+v", loadedExtra)
	}

	loadedExtra.SpecifiedTimes = []string{"2026-07-27 10:00:00"}
	loadedExtra.ExcludedTimes = nil
	loaded.Extra, err = MarshalExtra(loadedExtra)
	if err != nil {
		t.Fatal(err)
	}
	loaded.RRuleStr = "DTSTART:20260701T000000Z\nRRULE:FREQ=DAILY\nRDATE:20260727T100000Z"
	if err := store.Update(context.Background(), loaded); err != nil {
		t.Fatal(err)
	}
	var record gormmodel.CronJob
	if err := db.Where("id = ?", config.ID).First(&record).Error; err != nil {
		t.Fatal(err)
	}
	if !record.SpecifiedTimes.Valid || record.SpecifiedTimes.String != `["2026-07-27 10:00:00"]` || record.ExcludedTimes.Valid {
		t.Fatalf("replacement/clear not persisted atomically: %+v", record)
	}
	if !strings.Contains(record.RRuleStr, "RDATE:20260727T100000Z") ||
		strings.Contains(record.RRuleStr, "20260725T090000Z") || strings.Contains(record.RRuleStr, "EXDATE") {
		t.Fatalf("compiled set was not replaced with exact-time configuration: %q", record.RRuleStr)
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
	if err := store.Complete(context.Background(), cfg.ID, claim.LockedUntil, crontask.Completion{NextRun: now.Add(time.Hour), LastRun: now}); !errors.Is(err, crontask.ErrNotFound) {
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
	if err := store.Complete(context.Background(), cfg.ID, claim.LockedUntil, crontask.Completion{NextRun: nextRun, LastRun: now}); err != nil {
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
	if err := store.Complete(context.Background(), cfg.ID, claim.LockedUntil, crontask.Completion{NextRun: now.Add(time.Hour), LastRun: now}); !errors.Is(err, crontask.ErrNotFound) {
		t.Fatalf("enable should invalidate the previous claim, got %v", err)
	}
}

func TestDBStoreEnableRecalculatesFromPersistedExactTimeSet(t *testing.T) {
	db := newCronJobTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	now := time.Now().UTC().Truncate(time.Second)
	specified := now.Add(2 * time.Hour)
	excluded := now.Add(time.Hour)
	periodicStart := now.Add(24 * time.Hour)
	cfg := cronJobTestConfig(t, excluded)
	cfg.TaskCode = "ENABLE-EXACT-TIME"
	cfg.Status = crontask.StatusDisabled
	cfg.RRuleStr = "DTSTART:" + periodicStart.Format("20060102T150405Z") + "\n" +
		"RRULE:FREQ=DAILY;COUNT=2\n" +
		"RDATE:" + excluded.Format("20060102T150405Z") + "," + specified.Format("20060102T150405Z") + "\n" +
		"EXDATE:" + excluded.Format("20060102T150405Z")
	if err := store.Insert(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}

	if err := store.Enable(context.Background(), cfg.ID); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.GetByID(context.Background(), cfg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.NextRun.Equal(specified) {
		t.Fatalf("enable next run = %v, want persisted RDATE %v", loaded.NextRun, specified)
	}
}

func TestDBStoreEnablePreservesOneShotSchedule(t *testing.T) {
	db := newCronJobTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	scheduled := time.Now().Add(time.Hour).Truncate(time.Second)
	cfg := cronJobTestConfig(t, scheduled)
	cfg.TaskCode = "ONE-SHOT-ENABLE"
	cfg.RRuleStr = ""
	if err := store.Insert(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if err := store.Disable(context.Background(), cfg.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.Enable(context.Background(), cfg.ID); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetByID(context.Background(), cfg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.NextRun.Equal(scheduled) {
		t.Fatalf("one-shot next run = %v, want preserved %v", got.NextRun, scheduled)
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

func TestDBStoreGetByIDAndProto(t *testing.T) {
	db := newCronJobTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	baseTime := time.Date(2026, 7, 27, 10, 0, 0, 0, time.Local)

	newConfig := func(taskCode, taskName, deptCode, taskType, groupID string, status crontask.TaskStatus, nextRun time.Time) *crontask.TaskConfig {
		config := cronJobTestConfig(t, nextRun)
		config.TaskCode = taskCode
		config.TaskName = taskName
		config.Status = status
		config.Payload = json.RawMessage(`{"job":"` + taskCode + `"}`)
		extra, err := ParseExtra(config.Extra)
		if err != nil {
			t.Fatal(err)
		}
		extra.DeptCode = deptCode
		extra.Type = taskType
		extra.GroupId = groupID
		config.Extra, err = MarshalExtra(extra)
		if err != nil {
			t.Fatal(err)
		}
		return config
	}

	alpha := newConfig("JOB-ALPHA", "巡检甲", "D001", "inspection", "G001", crontask.StatusEnabled, time.Time{})
	deleted := newConfig("JOB-DELETED", "已删除", "D001", "inspection", "G001", crontask.StatusEnabled, baseTime.Add(2*time.Hour))
	for i, config := range []*crontask.TaskConfig{alpha, deleted} {
		if err := store.Insert(context.Background(), config); err != nil {
			t.Fatal(err)
		}
		createdAt := baseTime.Add(time.Duration(i) * time.Hour)
		if err := db.Model(&gormmodel.CronJob{}).
			Where("id = ?", config.ID).
			Updates(map[string]interface{}{"create_time": createdAt, "update_time": createdAt}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Delete(context.Background(), deleted.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := store.GetByID(context.Background(), deleted.ID); !errors.Is(err, crontask.ErrNotFound) {
		t.Fatalf("deleted job lookup = %v, want ErrNotFound", err)
	}

	alphaRecord, err := store.GetByID(context.Background(), alpha.ID)
	if err != nil {
		t.Fatal(err)
	}
	pbJob, err := ToProto(alphaRecord)
	if err != nil {
		t.Fatal(err)
	}
	if pbJob.JobId != alpha.ID || pbJob.TaskCode != alpha.TaskCode || pbJob.NextRun != "" || pbJob.LastRun != "" {
		t.Fatalf("unexpected proto identity or nullable times: %+v", pbJob)
	}
	wantAuditTime := baseTime.Format(dateTimeLayout)
	if pbJob.CreateTime != wantAuditTime || pbJob.UpdateTime != wantAuditTime {
		t.Fatalf("unexpected proto audit times: %+v", pbJob)
	}
	if pbJob.Rule == nil || pbJob.Rule.Freq != 3 {
		t.Fatalf("unexpected proto business fields: %+v", pbJob)
	}
	if pbJob.RruleStr != alphaRecord.RRuleStr || pbJob.ScheduleDescription == "" {
		t.Fatalf("unexpected proto schedule fields: %+v", pbJob)
	}
}

func cronJobTestConfig(t *testing.T, nextRun time.Time) *crontask.TaskConfig {
	t.Helper()
	ruleJSON, _ := json.Marshal(&trigger.PlanRulePb{Freq: 3, Hours: []int32{11}, Minutes: []int32{0}})
	extra, err := MarshalExtra(&CronJobExtra{
		DeptCode: "D001",
		Type:     "test",
		GroupId:  "G001",
		Rule:     ruleJSON,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &crontask.TaskConfig{
		TaskCode: "SAME",
		TaskName: "same",
		RRuleStr: "DTSTART:20260701T000000Z\nRRULE:FREQ=DAILY",
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
	if err := db.AutoMigrate(&gormmodel.CronJob{}, &gormmodel.CronExecLog{}); err != nil {
		t.Fatal(err)
	}
	return db
}
