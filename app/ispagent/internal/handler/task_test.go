package handler

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	ctask "zero-service/app/ispagent/internal/crontask"
	"zero-service/app/ispagent/model/gormmodel"
	"zero-service/common/crontask"
	"zero-service/common/gormx"
	"zero-service/common/isp"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHandleTaskControlRejectsItems(t *testing.T) {
	ctx := context.Background()
	store := crontask.NewMemoryStore()
	fields := &ctask.IspTaskFields{SubstationCode: "SUB001", TaskCode: "SIP25082613151430"}
	if err := store.Insert(ctx, &crontask.TaskConfig{
		TaskCode: "SIP25082613151430",
		TaskName: "35kVSVG周期任务模板",
		Extra:    []byte(ctask.SerializeExtra(fields)),
		NextRun:  time.Date(2025, 12, 16, 10, 0, 0, 0, time.Local),
	}); err != nil {
		t.Fatalf("insert task: %v", err)
	}

	_, err := HandleTaskControl(ctx, &isp.Message{
		Command: isp.CommandTaskPause,
		Code:    "SUB001_SIP25082613151430_20251216100000",
		Items: []isp.Item{{
			"task_patrolled_id": "SIP25082613151430_20251216100000",
			"task_code":         "SIP25082613151430",
			"plan_start_time":   "2025-12-16 10:00:00",
			"start_time":        "2025-12-16 10:00:00",
		}},
	}, store, nil, nil, nil)
	if err == nil {
		t.Fatal("expected item error")
	}
	if !strings.Contains(err.Error(), "任务控制指令不应包含 Item") {
		t.Fatalf("expected item error, got %v", err)
	}
}

func TestHandleTaskControlReturnsTaskNotFound(t *testing.T) {
	_, err := HandleTaskControl(context.Background(), &isp.Message{
		Command: isp.CommandTaskStart,
		Code:    "missing-task",
	}, crontask.NewMemoryStore(), nil, nil, nil)
	if err == nil {
		t.Fatal("expected missing task error")
	}
	if !strings.Contains(err.Error(), "任务不存在: missing-task") {
		t.Fatalf("expected user-facing task not found error, got %v", err)
	}
}

func TestHandleTaskControlRejectsMultipleItems(t *testing.T) {
	_, err := HandleTaskControl(context.Background(), &isp.Message{
		Command: isp.CommandTaskPause,
		Code:    "SUB001_TASK001_20251216100000",
		Items: []isp.Item{
			{"plan_start_time": "2025-12-16 10:00:00"},
			{"plan_start_time": "2025-12-16 10:00:00"},
		},
	}, crontask.NewMemoryStore(), nil, nil, nil)
	if err == nil {
		t.Fatal("expected multiple items error")
	}
	if !strings.Contains(err.Error(), "任务控制指令不应包含 Item") {
		t.Fatalf("expected multiple items error, got %v", err)
	}
}

func TestHandleTaskDispatchDeleteMissingTaskIsNoop(t *testing.T) {
	err := HandleTaskDispatch(context.Background(), &isp.Message{
		Code: "SUB001",
		Items: []isp.Item{{
			"task_code":        "missing-delete",
			"task_name":        "待删除任务",
			"fixed_start_time": "2025-12-16 10:00:00",
			"isenable":         "2",
		}},
	}, crontask.NewMemoryStore())
	if err != nil {
		t.Fatalf("expected missing delete to be noop, got %v", err)
	}
}

func TestHandleTaskDispatchDisableMissingTaskInsertsDisabledTask(t *testing.T) {
	ctx := context.Background()
	store := crontask.NewMemoryStore()
	err := HandleTaskDispatch(ctx, &isp.Message{
		Code: "SUB001",
		Items: []isp.Item{{
			"task_code":        "missing-disable",
			"task_name":        "待停用任务",
			"fixed_start_time": "2025-12-16 10:00:00",
			"isenable":         "1",
		}},
	}, store)
	if err != nil {
		t.Fatalf("HandleTaskDispatch: %v", err)
	}
	task, err := store.GetByCode(ctx, "missing-disable")
	if err != nil {
		t.Fatalf("expected disabled task inserted, got %v", err)
	}
	if task.Status != crontask.StatusDisabled {
		t.Fatalf("expected disabled status, got %v", task.Status)
	}
}

func TestHandleTaskDispatchPreservesExistingLockTimeout(t *testing.T) {
	ctx := context.Background()
	store := crontask.NewMemoryStore()
	if err := store.Insert(ctx, &crontask.TaskConfig{
		ID:          "existing-task",
		TaskCode:    "task-lock-timeout",
		TaskName:    "旧任务名称",
		LockTimeout: 90 * time.Second,
		Status:      crontask.StatusEnabled,
		NextRun:     time.Date(2026, 12, 15, 10, 0, 0, 0, time.Local),
	}); err != nil {
		t.Fatalf("insert existing task: %v", err)
	}
	err := HandleTaskDispatch(ctx, &isp.Message{
		Code: "SUB001",
		Items: []isp.Item{{
			"task_code":        "task-lock-timeout",
			"task_name":        "锁超时测试",
			"fixed_start_time": "2026-12-16 10:00:00",
			"isenable":         "0",
		}},
	}, store)
	if err != nil {
		t.Fatalf("HandleTaskDispatch: %v", err)
	}
	task, err := store.GetByCode(ctx, "task-lock-timeout")
	if err != nil {
		t.Fatal(err)
	}
	if task.LockTimeout != 90*time.Second {
		t.Fatalf("lock timeout = %v, want preserved %v", task.LockTimeout, 90*time.Second)
	}
}

func TestHandleTaskControlParsesSubstationFromMessageCode(t *testing.T) {
	ctx := context.Background()
	store := crontask.NewMemoryStore()
	db := newTaskControlTestDB(t)
	patrolTask := gormmodel.GormIspPatrolTask{
		SendCode:        "send",
		ReceiveCode:     "receive",
		Code:            "SUB001",
		TaskPatrolledID: "SUB001_TASK001_20251216100000",
		TaskName:        "测试任务",
		TaskCode:        "TASK001",
		TaskState:       "2",
		PlanStartTime:   time.Date(2025, 12, 16, 10, 0, 0, 0, time.Local),
		StartTime:       time.Date(2025, 12, 16, 10, 0, 0, 0, time.Local),
		TaskProgress:    "0",
	}
	if err := db.Create(&patrolTask).Error; err != nil {
		t.Fatalf("insert patrol task: %v", err)
	}

	type taskControlNotification struct {
		code  string
		items []isp.Item
	}
	notified := make(chan taskControlNotification, 1)
	_, err := handleTaskControl(ctx, &isp.Message{
		Command: isp.CommandTaskPause,
		Code:    "SUB001_TASK001_20251216100000",
	}, store, nil, db, func(ctx context.Context, code string, items []isp.Item) {
		notified <- taskControlNotification{code: code, items: items}
	}, 0)
	if err != nil {
		t.Fatalf("HandleTaskControl: %v", err)
	}

	select {
	case notification := <-notified:
		if notification.code != "SUB001" {
			t.Fatalf("expected substation code from msg.Code, got %q", notification.code)
		}
		if len(notification.items) != 1 {
			t.Fatalf("expected one notification item, got %d", len(notification.items))
		}
		item := notification.items[0]
		if item["task_patrolled_id"] != patrolTask.TaskPatrolledID || item["task_code"] != patrolTask.TaskCode || item["task_state"] != "3" {
			t.Fatalf("unexpected notification item: %v", item)
		}
		if item["plan_start_time"] != "2025-12-16 10:00:00" || item["start_time"] != "2025-12-16 10:00:00" {
			t.Fatalf("unexpected notification times: %v", item)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for notify")
	}
	var updated gormmodel.GormIspPatrolTask
	if err := db.Select("task_state", "plan_start_time", "start_time").
		Where("task_patrolled_id = ?", "SUB001_TASK001_20251216100000").
		First(&updated).Error; err != nil {
		t.Fatalf("query updated patrol task: %v", err)
	}
	if updated.TaskState != "3" {
		t.Fatalf("expected paused state, got %q", updated.TaskState)
	}
	if !updated.PlanStartTime.Equal(patrolTask.PlanStartTime) || !updated.StartTime.Equal(patrolTask.StartTime) {
		t.Fatalf("expected non-start control to preserve times, got plan=%v start=%v", updated.PlanStartTime, updated.StartTime)
	}
}

func TestHandleTaskControlStartRunsTaskThroughScheduler(t *testing.T) {
	ctx := context.Background()
	store := crontask.NewMemoryStore()
	fields := &ctask.IspTaskFields{SubstationCode: "SUB001", TaskCode: "TASK001"}
	nextRun := time.Date(2025, 12, 16, 10, 0, 0, 0, time.Local)
	if err := store.Insert(ctx, &crontask.TaskConfig{
		ID:       "task-id",
		TaskCode: "TASK001",
		TaskName: "测试任务",
		Extra:    []byte(ctask.SerializeExtra(fields)),
		NextRun:  nextRun,
	}); err != nil {
		t.Fatalf("insert task: %v", err)
	}
	type manualExecution struct {
		task            *crontask.TaskConfig
		taskPatrolledID string
		runAt           time.Time
		ok              bool
	}
	executed := make(chan manualExecution, 1)
	scheduler := crontask.NewScheduler(store, func(ctx context.Context, task *crontask.TaskConfig) error {
		taskPatrolledID, runAt, ok := ctask.ManualExecutionFromContext(ctx)
		executed <- manualExecution{task: task, taskPatrolledID: taskPatrolledID, runAt: runAt, ok: ok}
		return nil
	})

	taskPatrolledID, err := HandleTaskControl(ctx, &isp.Message{
		Command: isp.CommandTaskStart,
		Code:    "TASK001",
	}, store, scheduler.RunNow, nil, nil)
	if err != nil {
		t.Fatalf("HandleTaskControl: %v", err)
	}
	if !strings.HasPrefix(taskPatrolledID, "SUB001_TASK001_") {
		t.Fatalf("unexpected task patrolled id: %q", taskPatrolledID)
	}

	select {
	case execution := <-executed:
		if execution.task.TaskCode != "TASK001" {
			t.Fatalf("expected TASK001 to execute, got %q", execution.task.TaskCode)
		}
		if execution.task.NextRun.IsZero() || execution.task.NextRun.Nanosecond() != 0 {
			t.Fatalf("expected current second execution time, got %v", execution.task.NextRun)
		}
		if !execution.ok {
			t.Fatal("expected manual execution metadata")
		}
		if execution.taskPatrolledID != taskPatrolledID {
			t.Fatalf("handler task patrolled id = %q, response id = %q", execution.taskPatrolledID, taskPatrolledID)
		}
		if execution.runAt.IsZero() || execution.runAt.Nanosecond() != 0 {
			t.Fatalf("unexpected manual execution time: %v", execution.runAt)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for RunNow handler")
	}

	deadline := time.Now().Add(time.Second)
	for {
		stored, err := store.GetByCode(ctx, "TASK001")
		if err != nil {
			t.Fatalf("get task after RunNow: %v", err)
		}
		if !stored.NextRun.Equal(nextRun) {
			t.Fatalf("RunNow changed periodic next run: %v", stored.NextRun)
		}
		if !stored.LastRun.IsZero() {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timeout waiting for RunNow to update last run")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

var taskControlTestDBSequence atomic.Uint64

func newTaskControlTestDB(t *testing.T) *gormx.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:task-control-%d?mode=memory&cache=shared&parseTime=true&_loc=auto", taskControlTestDBSequence.Add(1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&gormmodel.GormIspPatrolTask{}); err != nil {
		t.Fatalf("migrate patrol task: %v", err)
	}
	return &gormx.DB{DB: db}
}

func TestHandleTaskControlTreatsNilTaskAsNotFound(t *testing.T) {
	_, err := HandleTaskControl(context.Background(), &isp.Message{
		Command: isp.CommandTaskStart,
		Code:    "nil-task",
	}, nilTaskStore{}, nil, nil, nil)
	if err == nil {
		t.Fatal("expected nil task error")
	}
	if !strings.Contains(err.Error(), "任务不存在: nil-task") {
		t.Fatalf("expected user-facing task not found error, got %v", err)
	}
}

type nilTaskStore struct{}

func (nilTaskStore) LockAndFetch(context.Context, time.Time, time.Duration) (*crontask.TaskClaim, error) {
	return nil, crontask.ErrNotFound
}

func (nilTaskStore) Complete(context.Context, string, time.Time, time.Time, time.Time) error {
	return crontask.ErrNotFound
}

func (nilTaskStore) UpdateLastRun(context.Context, string, time.Time) error {
	return crontask.ErrNotFound
}

func (nilTaskStore) GetByCode(context.Context, string) (*crontask.TaskConfig, error) {
	return nil, nil
}

func (nilTaskStore) Insert(context.Context, *crontask.TaskConfig) error {
	return nil
}

func (nilTaskStore) Update(context.Context, *crontask.TaskConfig) error {
	return nil
}

func (nilTaskStore) Enable(context.Context, string) error {
	return nil
}

func (nilTaskStore) Disable(context.Context, string) error {
	return nil
}

func (nilTaskStore) Delete(context.Context, string) error {
	return nil
}

func (nilTaskStore) List(context.Context, crontask.ListCondition) ([]*crontask.TaskConfig, error) {
	return nil, nil
}
