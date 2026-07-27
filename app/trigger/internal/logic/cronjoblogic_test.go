package logic

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"zero-service/app/trigger/internal/cronjob"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/crontask"
	"zero-service/common/gormx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCronJobLifecycle(t *testing.T) {
	store, db := newCronJobLogicTestStore(t)
	serviceContext := &svc.ServiceContext{DB: db, CronJobStore: store}
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

func TestCronJobRunGetAndList(t *testing.T) {
	store, db := newCronJobLogicTestStore(t)
	ctx := context.Background()
	nextRun := time.Now().Add(time.Hour).Truncate(time.Second)
	ruleJSON, err := json.Marshal(&trigger.PlanRulePb{Freq: 3, Hours: []int32{11}, Minutes: []int32{0}})
	if err != nil {
		t.Fatal(err)
	}
	extra, err := cronjob.MarshalExtra(&cronjob.CronJobExtra{
		DeptCode:     "D001",
		Type:         "inspection",
		GroupId:      "G001",
		Description:  "立即执行测试",
		Rule:         ruleJSON,
		ExcludeDates: []string{"2026-07-28"},
		BizExtra:     json.RawMessage(`{"source":"logic-test"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	config := &crontask.TaskConfig{
		TaskCode:    "management-logic",
		TaskName:    "管理接口测试",
		RRuleStr:    "DTSTART:20260701T000000Z\nRRULE:FREQ=DAILY",
		Priority:    5,
		LockTimeout: 90 * time.Second,
		Payload:     json.RawMessage(`{"id":1}`),
		Extra:       extra,
		Status:      crontask.StatusDisabled,
		NextRun:     nextRun,
	}
	if err := store.Insert(ctx, config); err != nil {
		t.Fatal(err)
	}
	baseTime := time.Date(2026, 7, 27, 10, 0, 0, 0, time.Local)
	if err := db.Model(&gormmodel.CronJob{}).
		Where("id = ?", config.ID).
		Updates(map[string]interface{}{"create_time": baseTime, "update_time": baseTime}).Error; err != nil {
		t.Fatal(err)
	}
	laterConfig := *config
	laterConfig.ID = ""
	laterConfig.TaskCode = "management-logic-later"
	laterConfig.TaskName = "管理接口测试二"
	if err := store.Insert(ctx, &laterConfig); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&gormmodel.CronJob{}).
		Where("id = ?", laterConfig.ID).
		Updates(map[string]interface{}{"create_time": baseTime.Add(time.Hour), "update_time": baseTime.Add(time.Hour)}).Error; err != nil {
		t.Fatal(err)
	}
	deletedConfig := laterConfig
	deletedConfig.ID = ""
	deletedConfig.TaskCode = "management-logic-deleted"
	if err := store.Insert(ctx, &deletedConfig); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, deletedConfig.ID); err != nil {
		t.Fatal(err)
	}

	executed := make(chan *crontask.TaskConfig, 1)
	lastRunUpdated := make(chan error, 1)
	schedulerStore := &notifyingTaskStore{TaskStore: store, lastRunUpdated: lastRunUpdated}
	scheduler := crontask.NewScheduler(schedulerStore, func(_ context.Context, task *crontask.TaskConfig) error {
		executed <- task
		return nil
	})
	serviceContext := &svc.ServiceContext{
		DB:               db,
		CronJobStore:     store,
		CronJobScheduler: scheduler,
	}

	if _, err := NewRunCronJobLogic(ctx, serviceContext).RunCronJob(&trigger.RunCronJobReq{JobId: config.ID}); err != nil {
		t.Fatal(err)
	}
	select {
	case task := <-executed:
		if task.NextRun.IsZero() {
			t.Fatal("manual execution time must not be zero")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for manual execution")
	}
	select {
	case err := <-lastRunUpdated:
		if err != nil {
			t.Fatalf("update manual last run: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for last run update")
	}

	loaded, err := store.GetByID(ctx, config.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.LastRun.IsZero() {
		t.Fatal("manual execution did not update last run")
	}
	if loaded.Status != crontask.StatusDisabled || !loaded.NextRun.Equal(nextRun) {
		t.Fatalf("manual execution changed periodic plan: %+v", loaded)
	}

	detail, err := NewGetCronJobLogic(ctx, serviceContext).GetCronJob(&trigger.GetCronJobReq{JobId: config.ID})
	if err != nil {
		t.Fatal(err)
	}
	if detail.CronJob == nil || detail.CronJob.TaskCode != config.TaskCode || detail.CronJob.Extra != `{"source":"logic-test"}` {
		t.Fatalf("unexpected Cron Job detail: %+v", detail.CronJob)
	}
	wantAuditTime := baseTime.Format("2006-01-02 15:04:05")
	if detail.CronJob.CreateTime != wantAuditTime || detail.CronJob.UpdateTime == "" {
		t.Fatalf("unexpected detail audit times: %+v", detail.CronJob)
	}
	list, err := NewListCronJobsLogic(ctx, serviceContext).ListCronJobs(&trigger.ListCronJobsReq{
		PageNum:  1,
		PageSize: 1,
		TaskCode: "management",
		TaskName: "管理",
		Status:   []int32{int32(crontask.StatusDisabled)},
		DeptCode: "D001",
		Type:     "inspection",
		GroupId:  "G001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if list.Total != 2 || len(list.CronJobs) != 1 || list.CronJobs[0].JobId != laterConfig.ID {
		t.Fatalf("unexpected Cron Job list: %+v", list)
	}
	if list.CronJobs[0].CreateTime != baseTime.Add(time.Hour).Format("2006-01-02 15:04:05") {
		t.Fatalf("unexpected list audit time: %+v", list.CronJobs[0])
	}

	if _, err := NewGetCronJobLogic(ctx, serviceContext).GetCronJob(&trigger.GetCronJobReq{JobId: "missing"}); !tool.IsErrorByPbCode(err, extproto.Code__1_02_RECORD_NOT_EXIST) {
		t.Fatalf("get missing error = %v, want RECORD_NOT_EXIST", err)
	}
	if _, err := NewRunCronJobLogic(ctx, serviceContext).RunCronJob(&trigger.RunCronJobReq{JobId: "missing"}); !tool.IsErrorByPbCode(err, extproto.Code__1_02_RECORD_NOT_EXIST) {
		t.Fatalf("run missing error = %v, want RECORD_NOT_EXIST", err)
	}
}

type notifyingTaskStore struct {
	crontask.TaskStore
	lastRunUpdated chan<- error
}

func (s *notifyingTaskStore) UpdateLastRun(ctx context.Context, id string, lastRun time.Time) error {
	err := s.TaskStore.UpdateLastRun(ctx, id, lastRun)
	s.lastRunUpdated <- err
	return err
}

func newCronJobLogicTestStore(t *testing.T) (*cronjob.DBStore, *gormx.DB) {
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
	wrappedDB := &gormx.DB{DB: db}
	return cronjob.NewDBStore(wrappedDB), wrappedDB
}
