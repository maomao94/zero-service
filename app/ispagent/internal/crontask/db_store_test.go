package crontask

import (
	"context"
	"database/sql"
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
		TaskCode: "TASK001",
		TaskName: "测试任务",
		Priority: 1,
		Status:   int(commoncrontask.StatusEnabled),
		NextRun:  sql.NullTime{Time: now.Add(-time.Minute), Valid: true},
	}).Error; err != nil {
		t.Fatalf("insert task config: %v", err)
	}

	store := NewDBStore(&gormx.DB{DB: db})
	got, err := store.LockAndFetch(context.Background(), now, 30*time.Second)
	if err != nil {
		t.Fatalf("LockAndFetch: %v", err)
	}
	if got.TaskCode != "TASK001" {
		t.Fatalf("task code = %q, want TASK001", got.TaskCode)
	}
	if !got.NextRun.Equal(now.Add(-time.Minute)) {
		t.Fatalf("next run = %v, want original due time", got.NextRun)
	}

	if err := store.UpdateNextRun(context.Background(), got.ID, time.Time{}, now); err != nil {
		t.Fatalf("UpdateNextRun(zero): %v", err)
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

	future := now.Add(time.Hour)
	if err := store.UpdateNextRun(context.Background(), got.ID, future, now); err != nil {
		t.Fatalf("restore next run: %v", err)
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
	got, err := store.LockAndFetch(context.Background(), now, 30*time.Second)
	if err != nil {
		t.Fatalf("LockAndFetch: %v", err)
	}
	if got.TaskCode != "HIGH" {
		t.Fatalf("task code = %q, want HIGH", got.TaskCode)
	}

	var low gormmodel.GormTaskConfig
	if err := db.Where("task_code = ?", "LOW").First(&low).Error; err != nil {
		t.Fatalf("reload low priority task: %v", err)
	}
	if !low.NextRun.Valid || !low.NextRun.Time.Equal(now.Add(-2*time.Minute)) {
		t.Fatalf("unselected task next_run changed: %v", low.NextRun)
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
