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
	if got.TaskCode != "TASK001" {
		t.Fatalf("task code = %q, want TASK001", got.TaskCode)
	}
	if !got.NextRun.Equal(now.Add(-time.Minute)) {
		t.Fatalf("next run = %v, want original due time", got.NextRun)
	}
	if got.LockTimeout != 2*time.Minute {
		t.Fatalf("lock timeout = %v, want %v", got.LockTimeout, 2*time.Minute)
	}
	if want := now.Add(2 * time.Minute); !claim.LockedUntil.Equal(want) {
		t.Fatalf("locked until = %v, want task-specific %v", claim.LockedUntil, want)
	}

	if err := store.Complete(context.Background(), got.ID, claim.LockedUntil, time.Time{}, now); err != nil {
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
	if _, err := store.LockAndFetch(context.Background(), now, 30*time.Second); err != commoncrontask.ErrNotFound {
		t.Fatalf("expected NULL next_run tasks not to be fetched, got %v", err)
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
	if err := store.Complete(context.Background(), record.Id, claim.LockedUntil, nextRun, now); err != nil {
		t.Fatalf("disabled in-flight task should complete: %v", err)
	}

	if err := db.Model(&gormmodel.GormTaskConfig{}).
		Where("id = ?", record.Id).
		Update("next_run", nextRun.Add(time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	if err := store.Complete(context.Background(), record.Id, nextRun, now.Add(2*time.Hour), now); !errors.Is(err, commoncrontask.ErrNotFound) {
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
		RRuleStr: "FREQ=DAILY",
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
