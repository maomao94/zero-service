package crontask

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"zero-service/app/ispagent/model/gormmodel"
	commoncrontask "zero-service/common/crontask"
	"zero-service/common/gormx"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDBStoreLockAndFetchUsesSQLiteRandomFunction(t *testing.T) {
	db := newDBStoreTestDB(t)

	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.Local)
	if err := db.Create(&gormmodel.GormTaskConfig{
		TaskCode: "EXHAUSTED",
		TaskName: "已结束任务",
		Priority: 100,
		Status:   int(commoncrontask.StatusEnabled),
		NextRun:  sql.NullTime{},
	}).Error; err != nil {
		t.Fatalf("insert exhausted task config: %v", err)
	}
	if err := db.Create(&gormmodel.GormTaskConfig{
		TaskCode:    "TASK001",
		TaskName:    "测试任务",
		Priority:    1,
		LockTimeout: int64((2 * time.Minute) / time.Millisecond),
		Status:      int(commoncrontask.StatusEnabled),
		NextRun:     sql.NullTime{Time: now.Add(-time.Minute), Valid: true},
	}).Error; err != nil {
		t.Fatalf("insert task config: %v", err)
	}

	store := NewDBStore(&gormx.DB{DB: db})
	claim, err := store.LockAndFetch(context.Background(), now, 30*time.Second)
	if err != nil {
		t.Fatalf("LockAndFetch: %v", err)
	}
	got := claim.Task
	originalScheduledRun := got.ScheduledTime
	if got.TaskCode != "TASK001" {
		t.Fatalf("task code = %q, want TASK001", got.TaskCode)
	}
	if !got.ScheduledTime.Equal(now.Add(-time.Minute)) {
		t.Fatalf("scheduled time = %v, want original due time", got.ScheduledTime)
	}
	if got.LockTimeout != 2*time.Minute {
		t.Fatalf("lock timeout = %v, want %v", got.LockTimeout, 2*time.Minute)
	}
	if want := now.Add(2 * time.Minute); !claim.LockedUntil.Equal(want) {
		t.Fatalf("locked until = %v, want task-specific %v", claim.LockedUntil, want)
	}

	if err := store.Complete(context.Background(), got.ID, claim.LockedUntil, commoncrontask.Completion{
		LastRun:          now,
		LastScheduledRun: originalScheduledRun,
	}); err != nil {
		t.Fatalf("Complete(zero): %v", err)
	}
	var updated gormmodel.GormTaskConfig
	if err := db.Where("id = ?", got.ID).First(&updated).Error; err != nil {
		t.Fatalf("reload task config: %v", err)
	}
	if updated.NextRun.Valid {
		t.Fatalf("expected SQL NULL next_run, got %v", updated.NextRun)
	}
	if !updated.LastRun.Valid || !updated.LastRun.Time.Equal(now) {
		t.Fatalf("last_run = %v, want %v", updated.LastRun, now)
	}
	if !updated.LastScheduledRun.Valid || !updated.LastScheduledRun.Time.Equal(originalScheduledRun) {
		t.Fatalf("last_scheduled_run = %v, want %v", updated.LastScheduledRun, originalScheduledRun)
	}

	got.NextRun = time.Time{}
	if err := store.Update(context.Background(), got); err != nil {
		t.Fatalf("Update with zero next run: %v", err)
	}
	updated = gormmodel.GormTaskConfig{}
	if err := db.Where("id = ?", got.ID).First(&updated).Error; err != nil {
		t.Fatalf("reload updated task config: %v", err)
	}
	if updated.NextRun.Valid {
		t.Fatalf("expected full update to persist SQL NULL next_run, got %v", updated.NextRun)
	}
	if !updated.LastRun.Valid || !updated.LastRun.Time.Equal(now) {
		t.Fatalf("full update changed last_run: %v", updated.LastRun)
	}
	if !updated.LastScheduledRun.Valid || !updated.LastScheduledRun.Time.Equal(originalScheduledRun) {
		t.Fatalf("full update changed last_scheduled_run: %v", updated.LastScheduledRun)
	}
	if _, err := store.LockAndFetch(context.Background(), now, 30*time.Second); err != commoncrontask.ErrNotFound {
		t.Fatalf("expected NULL next_run tasks not to be fetched, got %v", err)
	}
}

func TestDBStoreRetryKeepsOriginalScheduledTime(t *testing.T) {
	db := newDBStoreTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.Local)
	originalScheduledRun := now.Add(-time.Minute)
	record := &gormmodel.GormTaskConfig{
		TaskCode: "RETRY",
		Status:   int(commoncrontask.StatusEnabled),
		NextRun:  sql.NullTime{Time: originalScheduledRun, Valid: true},
	}
	if err := db.Create(record).Error; err != nil {
		t.Fatal(err)
	}

	firstClaim, err := store.LockAndFetch(context.Background(), now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	secondClaim, err := store.LockAndFetch(context.Background(), firstClaim.LockedUntil, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !secondClaim.Task.ScheduledTime.Equal(originalScheduledRun) {
		t.Fatalf("retry scheduled run = %v, want %v", secondClaim.Task.ScheduledTime, originalScheduledRun)
	}
	if err := store.Complete(context.Background(), record.Id, secondClaim.LockedUntil, commoncrontask.Completion{
		LastRun:          now,
		LastScheduledRun: secondClaim.Task.ScheduledTime,
	}); err != nil {
		t.Fatal(err)
	}

	var completed gormmodel.GormTaskConfig
	if err := db.Where("id = ?", record.Id).First(&completed).Error; err != nil {
		t.Fatal(err)
	}
	if completed.ScheduledTime.Valid {
		t.Fatalf("completed retry must clear scheduled_time: %v", completed.ScheduledTime)
	}
	if !completed.LastScheduledRun.Valid || !completed.LastScheduledRun.Time.Equal(originalScheduledRun) {
		t.Fatalf("last scheduled run = %v, want %v", completed.LastScheduledRun, originalScheduledRun)
	}
}

func TestDBStoreUpdatePreservesInFlightScheduledTime(t *testing.T) {
	db := newDBStoreTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.Local)
	record := &gormmodel.GormTaskConfig{
		TaskCode: "IN-FLIGHT",
		Status:   int(commoncrontask.StatusEnabled),
		NextRun:  sql.NullTime{Time: now.Add(-time.Minute), Valid: true},
	}
	if err := db.Create(record).Error; err != nil {
		t.Fatal(err)
	}
	claim, err := store.LockAndFetch(context.Background(), now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	claim.Task.TaskName = "updated"
	if err := store.Update(context.Background(), claim.Task); err != nil {
		t.Fatal(err)
	}

	var updated gormmodel.GormTaskConfig
	if err := db.Where("id = ?", record.Id).First(&updated).Error; err != nil {
		t.Fatal(err)
	}
	if !updated.ScheduledTime.Valid || !updated.ScheduledTime.Time.Equal(claim.Task.ScheduledTime) {
		t.Fatalf("scheduled_time = %v, want in-flight %v", updated.ScheduledTime, claim.Task.ScheduledTime)
	}
	if !updated.NextRun.Valid || !updated.NextRun.Time.Equal(claim.LockedUntil) {
		t.Fatalf("next_run = %v, want lease %v", updated.NextRun, claim.LockedUntil)
	}
}

func TestDBStoreLockAndFetchLocksOnlySelectedTask(t *testing.T) {
	db := newDBStoreTestDB(t)
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.Local)
	tasks := []gormmodel.GormTaskConfig{
		{
			TaskCode: "HIGH",
			Priority: 2,
			Status:   int(commoncrontask.StatusEnabled),
			NextRun:  sql.NullTime{Time: now.Add(-time.Minute), Valid: true},
		},
		{
			TaskCode: "LOW",
			Priority: 1,
			Status:   int(commoncrontask.StatusEnabled),
			NextRun:  sql.NullTime{Time: now.Add(-2 * time.Minute), Valid: true},
		},
	}
	if err := db.Create(&tasks).Error; err != nil {
		t.Fatalf("insert task configs: %v", err)
	}

	store := NewDBStore(&gormx.DB{DB: db})
	claim, err := store.LockAndFetch(context.Background(), now, 30*time.Second)
	if err != nil {
		t.Fatalf("LockAndFetch: %v", err)
	}
	got := claim.Task
	if got.TaskCode != "HIGH" {
		t.Fatalf("task code = %q, want HIGH", got.TaskCode)
	}
	if want := now.Add(30 * time.Second); !claim.LockedUntil.Equal(want) {
		t.Fatalf("locked until = %v, want default %v", claim.LockedUntil, want)
	}

	var low gormmodel.GormTaskConfig
	if err := db.Where("task_code = ?", "LOW").First(&low).Error; err != nil {
		t.Fatalf("reload low priority task: %v", err)
	}
	if !low.NextRun.Valid || !low.NextRun.Time.Equal(now.Add(-2*time.Minute)) {
		t.Fatalf("unselected task next_run changed: %v", low.NextRun)
	}
}

func TestDBStoreGetByID(t *testing.T) {
	db := newDBStoreTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	record := &gormmodel.GormTaskConfig{
		TaskCode: "GET-BY-ID",
		TaskName: "按 ID 查询",
		Status:   int(commoncrontask.StatusEnabled),
	}
	if err := db.Create(record).Error; err != nil {
		t.Fatal(err)
	}

	task, err := store.GetByID(context.Background(), record.Id)
	if err != nil {
		t.Fatal(err)
	}
	if task.ID != record.Id || task.TaskCode != record.TaskCode || task.Status != commoncrontask.StatusEnabled {
		t.Fatalf("unexpected task: %+v", task)
	}
	if task.CreateTime.IsZero() || task.UpdateTime.IsZero() {
		t.Fatalf("audit times not mapped: %+v", task)
	}
	if _, err := store.GetByID(context.Background(), "missing"); !errors.Is(err, commoncrontask.ErrNotFound) {
		t.Fatalf("missing task error = %v, want ErrNotFound", err)
	}
	if err := store.Delete(context.Background(), record.Id); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetByID(context.Background(), record.Id); !errors.Is(err, commoncrontask.ErrNotFound) {
		t.Fatalf("deleted task error = %v, want ErrNotFound", err)
	}
}

func TestDBStoreCompleteUsesLeaseTokenNotStatus(t *testing.T) {
	db := newDBStoreTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.Local)
	record := &gormmodel.GormTaskConfig{
		TaskCode: "COMPLETE",
		TaskName: "完成测试",
		Status:   int(commoncrontask.StatusEnabled),
		NextRun:  sql.NullTime{Time: now.Add(-time.Minute), Valid: true},
	}
	if err := db.Create(record).Error; err != nil {
		t.Fatal(err)
	}
	claim, err := store.LockAndFetch(context.Background(), now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Disable(context.Background(), record.Id); err != nil {
		t.Fatal(err)
	}
	nextRun := now.Add(time.Hour)
	if err := store.Complete(context.Background(), record.Id, claim.LockedUntil, commoncrontask.Completion{NextRun: nextRun, LastRun: now}); err != nil {
		t.Fatalf("disabled in-flight task should complete: %v", err)
	}

	if err := db.Model(&gormmodel.GormTaskConfig{}).
		Where("id = ?", record.Id).
		Update("next_run", nextRun.Add(time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	if err := store.Complete(context.Background(), record.Id, nextRun, commoncrontask.Completion{NextRun: now.Add(2 * time.Hour), LastRun: now}); !errors.Is(err, commoncrontask.ErrNotFound) {
		t.Fatalf("expected lost lease error, got %v", err)
	}
}

func TestDBStoreEnableDisableAreIdempotent(t *testing.T) {
	db := newDBStoreTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	now := time.Now()
	record := &gormmodel.GormTaskConfig{
		TaskCode: "ENABLE",
		TaskName: "启停测试",
		RRuleStr: "DTSTART:20260701T000000Z\nRRULE:FREQ=DAILY",
		Status:   int(commoncrontask.StatusEnabled),
		NextRun:  sql.NullTime{Time: now.Add(time.Hour), Valid: true},
	}
	if err := db.Create(record).Error; err != nil {
		t.Fatal(err)
	}
	if err := store.Disable(context.Background(), record.Id); err != nil {
		t.Fatal(err)
	}
	if err := store.Disable(context.Background(), record.Id); err != nil {
		t.Fatalf("repeated disable: %v", err)
	}
	if err := store.Disable(context.Background(), "missing"); !errors.Is(err, commoncrontask.ErrUpdate) {
		t.Fatalf("disable missing task = %v, want ErrUpdate", err)
	}
	if err := store.Enable(context.Background(), record.Id); err != nil {
		t.Fatal(err)
	}
	var enabled gormmodel.GormTaskConfig
	if err := db.Where("id = ?", record.Id).First(&enabled).Error; err != nil {
		t.Fatal(err)
	}
	firstNextRun := enabled.NextRun
	if enabled.Status != int(commoncrontask.StatusEnabled) || !firstNextRun.Valid || !firstNextRun.Time.After(now) {
		t.Fatalf("unexpected enabled task: %+v", enabled)
	}
	if err := store.Enable(context.Background(), record.Id); err != nil {
		t.Fatalf("repeated enable: %v", err)
	}
	enabled = gormmodel.GormTaskConfig{}
	if err := db.Where("id = ?", record.Id).First(&enabled).Error; err != nil {
		t.Fatal(err)
	}
	if !enabled.NextRun.Time.Equal(firstNextRun.Time) {
		t.Fatalf("repeated enable changed next_run: first=%v second=%v", firstNextRun.Time, enabled.NextRun.Time)
	}
}

func TestDBStoreDeleteIsIdempotent(t *testing.T) {
	db := newDBStoreTestDB(t)
	store := NewDBStore(&gormx.DB{DB: db})
	record := &gormmodel.GormTaskConfig{
		TaskCode: "DELETE",
		TaskName: "删除测试",
	}
	if err := db.Create(record).Error; err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(context.Background(), record.Id); err != nil {
		t.Fatalf("first delete: %v", err)
	}
	if err := store.Delete(context.Background(), record.Id); err != nil {
		t.Fatalf("repeated delete: %v", err)
	}
}

func newDBStoreTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&parseTime=true&_loc=auto"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&gormmodel.GormTaskConfig{}); err != nil {
		t.Fatalf("migrate task config: %v", err)
	}
	return db
}
