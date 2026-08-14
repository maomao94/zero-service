package gormmodel

import (
	"reflect"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openCronJobTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&parseTime=true&_loc=auto"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&CronJob{}); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func TestCronJobTaskCodeColumnSize128(t *testing.T) {
	field, ok := reflect.TypeFor[CronJob]().FieldByName("TaskCode")
	if !ok {
		t.Fatal("CronJob.TaskCode not found")
	}
	tag := field.Tag.Get("gorm")
	if !strings.Contains(tag, "size:128") {
		t.Fatalf("TaskCode gorm tag should declare size:128, got %q", tag)
	}
	if !strings.Contains(tag, "uniqueIndex:uq_cron_job_task_code") {
		t.Fatalf("TaskCode gorm tag should keep uniqueIndex uq_cron_job_task_code, got %q", tag)
	}
}

func TestCronJobStores128RuneTaskCode(t *testing.T) {
	db := openCronJobTestDB(t)
	code := strings.Repeat("a", 128)
	if err := db.Create(&CronJob{TaskCode: code}).Error; err != nil {
		t.Fatalf("create with 128-rune task code: %v", err)
	}
	var job CronJob
	if err := db.Where("task_code = ?", code).First(&job).Error; err != nil {
		t.Fatalf("load 128-rune task code: %v", err)
	}
	if job.TaskCode != code {
		t.Fatalf("task code roundtrip mismatch: got %d runes", len([]rune(job.TaskCode)))
	}
	if err := db.Create(&CronJob{TaskCode: code}).Error; err == nil {
		t.Fatal("duplicate task code should be rejected by unique index")
	}
}
