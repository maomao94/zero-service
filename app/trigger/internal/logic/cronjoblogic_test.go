package logic

import (
	"context"
	"errors"
	"testing"
	"time"

	"zero-service/app/trigger/internal/cronjob"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/crontask"
	"zero-service/common/gormx"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCronJobLifecycle(t *testing.T) {
	store := newCronJobLogicTestStore(t)
	serviceContext := &svc.ServiceContext{CronJobStore: store}
	ctx := context.Background()
	nextHour := time.Now().Add(time.Hour).Hour()

	created, err := NewCreateCronJobLogic(ctx, serviceContext).CreateCronJob(&trigger.CreateCronJobReq{
		TaskCode:    "logic-lifecycle",
		TaskName:    "生命周期测试",
		Type:        "test",
		DeptCode:    "D001",
		Payload:     `{"id":1}`,
		Extra:       `{"source":"test"}`,
		LockTimeout: 90_000,
		Rule: &trigger.PlanRulePb{
			Freq:    3,
			Hours:   []int32{int32(nextHour)},
			Minutes: []int32{0},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.JobId == "" {
		t.Fatal("create must return jobId")
	}
	loaded, err := store.GetByID(ctx, created.JobId)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.LockTimeout != 90*time.Second {
		t.Fatalf("lock timeout = %v, want %v", loaded.LockTimeout, 90*time.Second)
	}

	disable := NewDisableCronJobLogic(ctx, serviceContext)
	for i := 0; i < 2; i++ {
		if _, err := disable.DisableCronJob(&trigger.DisableCronJobReq{JobId: created.JobId}); err != nil {
			t.Fatalf("disable attempt %d: %v", i+1, err)
		}
	}
	enable := NewEnableCronJobLogic(ctx, serviceContext)
	for i := 0; i < 2; i++ {
		if _, err := enable.EnableCronJob(&trigger.EnableCronJobReq{JobId: created.JobId}); err != nil {
			t.Fatalf("enable attempt %d: %v", i+1, err)
		}
	}

	deleteLogic := NewDeleteCronJobLogic(ctx, serviceContext)
	for i := 0; i < 2; i++ {
		if _, err := deleteLogic.DeleteCronJob(&trigger.DeleteCronJobReq{JobId: created.JobId}); err != nil {
			t.Fatalf("delete attempt %d: %v", i+1, err)
		}
	}
	if _, err := store.GetByID(ctx, created.JobId); !errors.Is(err, crontask.ErrNotFound) {
		t.Fatalf("deleted job must not be queryable: %v", err)
	}
}

func newCronJobLogicTestStore(t *testing.T) *cronjob.DBStore {
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
	return cronjob.NewDBStore(&gormx.DB{DB: db})
}
