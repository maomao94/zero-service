package crontask

import (
	"context"
	"testing"
	"time"

	"zero-service/app/ispagent/model/gormmodel"
	commoncrontask "zero-service/common/crontask"
	"zero-service/common/gormx"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDBStoreLockAndFetchUsesSQLiteRandomFunction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&parseTime=true&_loc=auto"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&gormmodel.GormTaskConfig{}); err != nil {
		t.Fatalf("migrate task config: %v", err)
	}

	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.Local)
	if err := db.Create(&gormmodel.GormTaskConfig{
		TaskCode: "TASK001",
		TaskName: "测试任务",
		Priority: 1,
		Status:   int(commoncrontask.StatusEnabled),
		NextRun:  now.Add(-time.Minute),
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
}
