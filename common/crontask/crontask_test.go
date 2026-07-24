package crontask

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dromara/carbon/v2"
)

func TestMemoryStoreInsertAndGet(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	cfg := &TaskConfig{
		TaskCode: "test-task",
		TaskName: "test",
		Status:   StatusEnabled,
		NextRun:  carbon.Now().StdTime().Add(-time.Hour),
	}
	if err := store.Insert(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.ID == "" {
		t.Fatal("expected auto-increment ID")
	}

	got, err := store.GetByCode(ctx, "test-task")
	if err != nil {
		t.Fatal(err)
	}
	if got.TaskCode != "test-task" {
		t.Fatalf("expected test-task, got %s", got.TaskCode)
	}
}

func TestMemoryStoreGetByCodeNotFound(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.GetByCode(context.Background(), "nonexistent")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStoreList(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	for _, task := range []*TaskConfig{
		{TaskCode: "a", Status: StatusEnabled},
		{TaskCode: "b", Status: StatusDisabled},
		{TaskCode: "c", Status: StatusEnabled},
	} {
		if err := store.Insert(ctx, task); err != nil {
			t.Fatal(err)
		}
	}

	list, err := store.List(ctx, ListCondition{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("expected all 3 tasks, got %d", len(list))
	}
	list, err = store.List(ctx, ListCondition{Statuses: []TaskStatus{StatusEnabled}})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 enabled tasks, got %d", len(list))
	}
}

func TestMemoryStoreEnableDisable(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	cfg := &TaskConfig{TaskCode: "t", Status: StatusEnabled}
	if err := store.Insert(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	if err := store.Disable(ctx, cfg.ID); err != nil {
		t.Fatal(err)
	}

	got, _ := store.GetByCode(ctx, "t")
	if got.Status != StatusDisabled {
		t.Fatalf("expected disabled, got %v", got.Status)
	}
	if err := store.Enable(ctx, cfg.ID); err != nil {
		t.Fatal(err)
	}
	got, _ = store.GetByCode(ctx, "t")
	if got.Status != StatusEnabled {
		t.Fatalf("expected enabled, got %v", got.Status)
	}
	if err := store.Disable(ctx, "missing"); !errors.Is(err, ErrUpdate) {
		t.Fatalf("disable missing task = %v, want ErrUpdate", err)
	}
}

func TestMemoryStoreLockAndFetch(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := carbon.Now().StdTime()

	t1 := &TaskConfig{TaskCode: "t1", Status: StatusEnabled, NextRun: now.Add(-time.Hour), Priority: 1}
	t2 := &TaskConfig{TaskCode: "t2", Status: StatusEnabled, NextRun: now.Add(-time.Minute), Priority: 2}
	store.Insert(ctx, t1)
	store.Insert(ctx, t2)

	claim, err := store.LockAndFetch(ctx, now, 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	got := claim.Task

	// higher priority (t2, priority=2) should be fetched
	if got.TaskCode != "t2" {
		t.Fatalf("expected t2 (higher priority), got %s", got.TaskCode)
	}

	// LockAndFetch returns the original next_run (for computeNextRun),
	// the lock extension is stored in the store.
	if !got.NextRun.Before(now) {
		t.Fatalf("expected original nextRun in past, got %v", got.NextRun)
	}

	// stored task should have next_run extended (locked)
	stored, _ := store.GetByCode(ctx, "t2")
	if !stored.NextRun.After(now) {
		t.Fatalf("expected nextRun extended in store, got %v", stored.NextRun)
	}
	if !stored.NextRun.Equal(claim.LockedUntil) {
		t.Fatalf("stored next run = %v, want locked until %v", stored.NextRun, claim.LockedUntil)
	}
	if claim.LockedUntil.Nanosecond() != 0 {
		t.Fatalf("locked until must use database-safe second precision: %v", claim.LockedUntil)
	}
}

func TestMemoryStoreLockAndFetchUsesTaskLockTimeout(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := time.Date(2026, 7, 24, 10, 0, 0, 0, time.Local)
	configuredLockTimeout := 2 * time.Minute

	if err := store.Insert(ctx, &TaskConfig{
		TaskCode:    "task-lock-timeout",
		Status:      StatusEnabled,
		NextRun:     now.Add(-time.Minute),
		LockTimeout: configuredLockTimeout,
	}); err != nil {
		t.Fatal(err)
	}

	claim, err := store.LockAndFetch(ctx, now, 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	wantLockedUntil := now.Add(configuredLockTimeout)
	if !claim.LockedUntil.Equal(wantLockedUntil) {
		t.Fatalf("locked until = %v, want %v", claim.LockedUntil, wantLockedUntil)
	}
	if claim.Task.LockTimeout != configuredLockTimeout {
		t.Fatalf("task lock timeout = %v, want %v", claim.Task.LockTimeout, configuredLockTimeout)
	}
}

func TestResolveLockTimeout(t *testing.T) {
	defaultLockTimeout := 5 * time.Minute
	if got := ResolveLockTimeout(0, defaultLockTimeout); got != defaultLockTimeout {
		t.Fatalf("zero task lock timeout = %v, want default %v", got, defaultLockTimeout)
	}
	if got := ResolveLockTimeout(-time.Second, defaultLockTimeout); got != defaultLockTimeout {
		t.Fatalf("negative task lock timeout = %v, want default %v", got, defaultLockTimeout)
	}
	if got := ResolveLockTimeout(time.Minute, defaultLockTimeout); got != time.Minute {
		t.Fatalf("configured task lock timeout = %v, want %v", got, time.Minute)
	}
	if got := ResolveLockTimeout(time.Second, defaultLockTimeout); got != MinLockTimeout {
		t.Fatalf("short task lock timeout = %v, want minimum %v", got, MinLockTimeout)
	}
	if got := ResolveLockTimeout(0, time.Second); got != MinLockTimeout {
		t.Fatalf("short default lock timeout = %v, want minimum %v", got, MinLockTimeout)
	}
}

func TestMemoryStoreLockAndFetchClampsShortLockTimeout(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := time.Date(2026, 7, 24, 10, 0, 0, int(900*time.Millisecond), time.Local)
	if err := store.Insert(ctx, &TaskConfig{
		TaskCode:    "short-lock-timeout",
		Status:      StatusEnabled,
		NextRun:     now.Add(-time.Minute),
		LockTimeout: time.Second,
	}); err != nil {
		t.Fatal(err)
	}

	claim, err := store.LockAndFetch(ctx, now, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	wantLockedUntil := now.Add(MinLockTimeout).Truncate(time.Second)
	if !claim.LockedUntil.Equal(wantLockedUntil) {
		t.Fatalf("locked until = %v, want %v", claim.LockedUntil, wantLockedUntil)
	}
	if _, err := store.LockAndFetch(ctx, now, time.Second); !errors.Is(err, ErrNotFound) {
		t.Fatalf("short lock timeout allowed immediate reclaim: %v", err)
	}
}

func TestMemoryStoreLockAndFetchPriorityRandom(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := carbon.Now().StdTime()

	for i := 0; i < 10; i++ {
		store.Insert(ctx, &TaskConfig{
			TaskCode: "t" + string(rune('0'+i)),
			Status:   StatusEnabled,
			NextRun:  now.Add(-time.Hour),
			Priority: 1,
		})
	}

	seen := make(map[string]bool)
	for i := 0; i < 20; i++ {
		// reset next_run so they are all eligible
		for _, task := range store.tasks {
			task.NextRun = now.Add(-time.Hour)
		}
		claim, _ := store.LockAndFetch(ctx, now, 30*time.Second)
		seen[claim.Task.TaskCode] = true
	}

	// with enough iterations, should see multiple different tasks
	if len(seen) < 3 {
		t.Fatalf("expected randomness, only saw %d tasks", len(seen))
	}
}

func TestMemoryStoreLockAndFetchNotFound(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.LockAndFetch(context.Background(), carbon.Now().StdTime(), 30*time.Second)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStoreComplete(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := carbon.Now().StdTime()

	store.Insert(ctx, &TaskConfig{TaskCode: "t", Status: StatusEnabled, NextRun: now.Add(-time.Hour)})

	claim, _ := store.LockAndFetch(ctx, now, 30*time.Second)
	newNext := now.Add(time.Hour)

	err := store.Complete(ctx, claim.Task.ID, claim.LockedUntil, newNext, now)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	got, _ := store.GetByCode(ctx, "t")
	if !got.NextRun.Equal(newNext) {
		t.Fatalf("expected next run updated, got %v", got.NextRun)
	}
	if !got.LastRun.Equal(now) {
		t.Fatalf("expected last run %v, got %v", now, got.LastRun)
	}
}

func TestMemoryStoreCompleteRejectsLostClaim(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := carbon.Now().StdTime()
	cfg := &TaskConfig{TaskCode: "lost", Status: StatusEnabled, NextRun: now.Add(-time.Minute)}
	if err := store.Insert(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	claim, err := store.LockAndFetch(ctx, now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	store.tasks[cfg.ID].NextRun = claim.LockedUntil.Add(time.Second)
	store.mu.Unlock()
	if err := store.Complete(ctx, cfg.ID, claim.LockedUntil, now.Add(time.Hour), now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected lost claim, got %v", err)
	}
}

func TestMemoryStoreCompleteAllowsConcurrentDisable(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := carbon.Now().StdTime()
	cfg := &TaskConfig{TaskCode: "disabled", Status: StatusEnabled, NextRun: now.Add(-time.Minute)}
	if err := store.Insert(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	claim, err := store.LockAndFetch(ctx, now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Disable(ctx, cfg.ID); err != nil {
		t.Fatal(err)
	}
	nextRun := now.Add(time.Hour)
	if err := store.Complete(ctx, cfg.ID, claim.LockedUntil, nextRun, now); err != nil {
		t.Fatalf("disabled in-flight task should complete: %v", err)
	}
	got, err := store.GetByCode(ctx, cfg.TaskCode)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusDisabled || !got.NextRun.Equal(nextRun) || !got.LastRun.Equal(now) {
		t.Fatalf("unexpected completed disabled task: %+v", got)
	}
}

func TestMemoryStoreIgnoresZeroNextRun(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	if err := store.Insert(ctx, &TaskConfig{
		TaskCode: "exhausted",
		Status:   StatusEnabled,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := store.LockAndFetch(ctx, carbon.Now().StdTime(), time.Minute); err != ErrNotFound {
		t.Fatalf("expected zero next run to be ignored, got %v", err)
	}
}

func TestMemoryStoreKeepsNextRunByValue(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	original := carbon.Now().StdTime().Add(time.Hour)
	cfg := &TaskConfig{
		TaskCode: "clone",
		Status:   StatusEnabled,
		NextRun:  original,
	}
	if err := store.Insert(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	cfg.NextRun = original.Add(time.Hour)
	got, err := store.GetByCode(ctx, cfg.TaskCode)
	if err != nil {
		t.Fatal(err)
	}
	if !got.NextRun.Equal(original) {
		t.Fatalf("stored next run changed through caller value: %v", got.NextRun)
	}

	got.NextRun = original.Add(2 * time.Hour)
	again, err := store.GetByCode(ctx, cfg.TaskCode)
	if err != nil {
		t.Fatal(err)
	}
	if !again.NextRun.Equal(original) {
		t.Fatalf("stored next run changed through returned value: %v", again.NextRun)
	}
}

func TestSchedulerTriggersHandler(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := carbon.Now().StdTime()

	store.Insert(ctx, &TaskConfig{
		TaskCode: "t",
		TaskName: "test",
		Status:   StatusEnabled,
		NextRun:  now.Add(-time.Hour),
		RRuleStr: "DTSTART:20200101T000000\nRRULE:FREQ=DAILY;COUNT=1",
	})

	var mu sync.Mutex
	var executed []string
	handler := func(ctx context.Context, task *TaskConfig) error {
		mu.Lock()
		executed = append(executed, task.TaskCode)
		mu.Unlock()
		return nil
	}

	s := NewScheduler(store, handler, WithInterval(100*time.Millisecond), WithLockExpire(30*time.Second))
	s.Start()
	defer s.Stop()

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	count := len(executed)
	mu.Unlock()
	if count == 0 {
		t.Fatal("expected handler to be called at least once")
	}

	// one-time task remains enabled but has no next schedule after its only execution.
	got, _ := store.GetByCode(ctx, "t")
	if !got.NextRun.IsZero() {
		t.Fatalf("expected no next run, got %v", got.NextRun)
	}
	if got.Status != StatusEnabled {
		t.Fatalf("expected task status to remain enabled, got %v", got.Status)
	}
	if _, err := store.LockAndFetch(ctx, carbon.Now().StdTime(), time.Second); err != ErrNotFound {
		t.Fatalf("expected exhausted task not to be fetched, got %v", err)
	}
}

func TestRunNow(t *testing.T) {
	store := NewMemoryStore()
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "manual-run")

	store.Insert(ctx, &TaskConfig{
		TaskCode: "t",
		Status:   StatusEnabled,
		NextRun:  carbon.Now().StdTime().Add(time.Hour),
	})

	var mu sync.Mutex
	executed := false
	contextValue := ""
	handler := func(ctx context.Context, task *TaskConfig) error {
		mu.Lock()
		executed = true
		contextValue, _ = ctx.Value(contextKey{}).(string)
		mu.Unlock()
		return nil
	}

	s := NewScheduler(store, handler, WithInterval(time.Hour), WithLockExpire(time.Hour))
	s.Start()
	defer s.Stop()

	if err := s.RunNow(ctx, "t"); err != nil {
		t.Fatal(err)
	}

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	if !executed {
		t.Fatal("expected RunNow to trigger handler")
	}
	if contextValue != "manual-run" {
		t.Fatalf("expected RunNow context value, got %q", contextValue)
	}
	mu.Unlock()
	got, err := store.GetByCode(ctx, "t")
	if err != nil {
		t.Fatal(err)
	}
	if !got.NextRun.After(time.Now()) {
		t.Fatalf("RunNow changed periodic next run: %v", got.NextRun)
	}
	if got.LastRun.IsZero() {
		t.Fatal("expected RunNow to update last run")
	}
}

func TestRunNowProvidesExecutionTimeForZeroNextRun(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	if err := store.Insert(ctx, &TaskConfig{
		TaskCode: "manual-exhausted",
		Status:   StatusEnabled,
		RRuleStr: "FREQ=DAILY;COUNT=1",
	}); err != nil {
		t.Fatal(err)
	}

	executed := make(chan time.Time, 1)
	s := NewScheduler(store, func(ctx context.Context, task *TaskConfig) error {
		executed <- task.NextRun
		return nil
	})
	if err := s.RunNow(ctx, "manual-exhausted"); err != nil {
		t.Fatal(err)
	}

	select {
	case runAt := <-executed:
		if runAt.IsZero() {
			t.Fatal("expected RunNow to provide an execution time")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for RunNow")
	}
}

func TestRecurringTaskComputesNextRun(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := carbon.Now().StdTime().Truncate(time.Hour)

	// daily recurrence, DTSTART should be part of the rrule string
	rruleStr := "FREQ=DAILY;INTERVAL=1"

	cfg := &TaskConfig{
		TaskCode: "recurring",
		TaskName: "test",
		Status:   StatusEnabled,
		RRuleStr: rruleStr,
		NextRun:  now.Add(-time.Hour * 24),
	}

	store.Insert(ctx, cfg)

	var mu sync.Mutex
	executed := false
	handler := func(ctx context.Context, task *TaskConfig) error {
		mu.Lock()
		executed = true
		mu.Unlock()
		return nil
	}

	s := NewScheduler(store, handler, WithInterval(100*time.Millisecond), WithLockExpire(5*time.Second))
	s.Start()
	defer s.Stop()

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	if !executed {
		t.Fatal("expected recurring task to be executed")
	}
	mu.Unlock()

	// task should still be enabled
	got, _ := store.GetByCode(ctx, "recurring")
	if got.Status != StatusEnabled {
		t.Fatalf("expected enabled after recurring execution, got %v", got.Status)
	}
	if got.NextRun.IsZero() || got.NextRun.Equal(cfg.NextRun) {
		t.Fatal("expected nextRun to be updated to next occurrence")
	}
}

func TestEmptyStoreNoPanic(t *testing.T) {
	store := NewMemoryStore()
	handler := func(ctx context.Context, task *TaskConfig) error { return nil }
	s := NewScheduler(store, handler, WithInterval(100*time.Millisecond), WithLockExpire(time.Second))
	s.Start()

	time.Sleep(300 * time.Millisecond)
	// should not panic
	s.Stop()
}

func TestMemoryStoreUpdate(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	cfg := &TaskConfig{TaskCode: "t", TaskName: "test", Status: StatusEnabled}
	if err := store.Insert(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	cfg.TaskName = "updated"
	if err := store.Update(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	got, _ := store.GetByCode(ctx, "t")
	if got.TaskName != "updated" {
		t.Fatalf("expected updated, got %s", got.TaskName)
	}
}

func TestMemoryStoreInsertDuplicate(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	store.Insert(ctx, &TaskConfig{TaskCode: "dup"})
	err := store.Insert(ctx, &TaskConfig{TaskCode: "dup"})
	if err != ErrDuplicate {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

func TestMemoryStoreUpdateDuplicateCode(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	store.Insert(ctx, &TaskConfig{TaskCode: "a"})
	store.Insert(ctx, &TaskConfig{TaskCode: "b"})

	a, _ := store.GetByCode(ctx, "a")
	a.TaskCode = "b"

	err := store.Update(ctx, a)
	if err != ErrDuplicate {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

func TestComputeNextRunInvalidRRule(t *testing.T) {
	_, err := computeNextRun(&TaskConfig{
		TaskCode: "t",
		RRuleStr: "INVALID_RRULE",
		NextRun:  carbon.Now().StdTime(),
	})
	if err == nil {
		t.Fatal("expected error for invalid rrule")
	}
}

func TestComputeNextRunExpiredTaskReturnsZero(t *testing.T) {
	next, err := computeNextRun(&TaskConfig{
		TaskCode: "t",
		RRuleStr: "FREQ=DAILY;COUNT=1",
		NextRun:  carbon.Now().StdTime(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !next.IsZero() {
		t.Fatalf("expected zero next run, got %v", next)
	}
}

func TestComputeNextRunAllowsZeroCurrentSchedule(t *testing.T) {
	next, err := computeNextRun(&TaskConfig{
		TaskCode: "manual",
		RRuleStr: "FREQ=DAILY;COUNT=1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !next.IsZero() {
		t.Fatalf("expected exhausted rule to stay without next run, got %v", next)
	}
}

func TestExecuteTaskErrorKeepsNextRun(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := carbon.Now().StdTime()

	cfg := &TaskConfig{
		TaskCode: "fail-task",
		Status:   StatusEnabled,
		RRuleStr: "FREQ=DAILY;INTERVAL=1",
		NextRun:  now.Add(-time.Hour),
	}
	if err := store.Insert(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	handler := func(ctx context.Context, task *TaskConfig) error {
		return context.DeadlineExceeded
	}

	s := NewScheduler(store, handler, WithInterval(100*time.Millisecond), WithLockExpire(30*time.Second))
	s.Start()
	defer s.Stop()

	time.Sleep(300 * time.Millisecond)

	got, _ := store.GetByCode(ctx, "fail-task")
	// LockAndFetch extended NextRun, so it should be in the future
	if got.NextRun.IsZero() || !got.NextRun.After(cfg.NextRun) {
		t.Fatal("expected LockAndFetch to have extended NextRun")
	}
}

func TestExecuteTaskDeleteSignalDeletesTask(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := carbon.Now().StdTime()
	cfg := &TaskConfig{
		TaskCode: "deleted-by-handler",
		Status:   StatusEnabled,
		RRuleStr: "FREQ=DAILY",
		NextRun:  now.Add(-time.Minute),
	}
	if err := store.Insert(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	claim, err := store.LockAndFetch(ctx, now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	s := NewScheduler(store, func(context.Context, *TaskConfig) error {
		return errors.Join(errors.New("business task missing"), ErrDeleteTask)
	})
	s.executeTask(claim)
	if _, err := store.GetByCode(ctx, cfg.TaskCode); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected task deleted, got %v", err)
	}
}

func TestExecuteTaskDirectDeleteSignalDeletesTask(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := carbon.Now().StdTime()
	cfg := &TaskConfig{
		TaskCode: "deleted-directly",
		Status:   StatusEnabled,
		RRuleStr: "FREQ=DAILY",
		NextRun:  now.Add(-time.Minute),
	}
	if err := store.Insert(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	claim, err := store.LockAndFetch(ctx, now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	NewScheduler(store, func(context.Context, *TaskConfig) error {
		return ErrDeleteTask
	}).executeTask(claim)
	if _, err := store.GetByCode(ctx, cfg.TaskCode); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected task deleted, got %v", err)
	}
}

type failOnceDeleteStore struct {
	*MemoryStore
	deleteCalls int
}

func (s *failOnceDeleteStore) Delete(ctx context.Context, id string) error {
	s.deleteCalls++
	if s.deleteCalls == 1 {
		return errors.New("delete unavailable")
	}
	return s.MemoryStore.Delete(ctx, id)
}

func TestExecuteTaskDeleteFailureRetriesAfterLease(t *testing.T) {
	store := &failOnceDeleteStore{MemoryStore: NewMemoryStore()}
	ctx := context.Background()
	now := carbon.Now().StdTime()
	cfg := &TaskConfig{
		TaskCode: "delete-retry",
		Status:   StatusEnabled,
		RRuleStr: "FREQ=DAILY",
		NextRun:  now.Add(-time.Minute),
	}
	if err := store.Insert(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	scheduler := NewScheduler(store, func(context.Context, *TaskConfig) error {
		return ErrDeleteTask
	})
	firstClaim, err := store.LockAndFetch(ctx, now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	scheduler.executeTask(firstClaim)
	if _, err := store.GetByCode(ctx, cfg.TaskCode); err != nil {
		t.Fatalf("delete failure must keep task for retry: %v", err)
	}
	secondClaim, err := store.LockAndFetch(ctx, firstClaim.LockedUntil, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	scheduler.executeTask(secondClaim)
	if _, err := store.GetByCode(ctx, cfg.TaskCode); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected retry to delete task, got %v", err)
	}
}

func TestSchedulerStopWithPendingTasks(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := carbon.Now().StdTime()

	store.Insert(ctx, &TaskConfig{
		TaskCode: "t",
		Status:   StatusEnabled,
		NextRun:  now.Add(-time.Hour),
		RRuleStr: "FREQ=DAILY;INTERVAL=1",
	})

	handler := func(ctx context.Context, task *TaskConfig) error {
		time.Sleep(500 * time.Millisecond)
		return nil
	}

	s := NewScheduler(store, handler, WithInterval(50*time.Millisecond), WithLockExpire(time.Second))
	s.Start()
	time.Sleep(60 * time.Millisecond)
	s.Stop()
}

func TestSchedulerStopWaitsForInFlightHandler(t *testing.T) {
	store := NewMemoryStore()
	now := carbon.Now().StdTime()
	if err := store.Insert(context.Background(), &TaskConfig{
		TaskCode: "graceful-stop",
		Status:   StatusEnabled,
		NextRun:  now.Add(-time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	scheduler := NewScheduler(store, func(context.Context, *TaskConfig) error {
		close(started)
		<-release
		return nil
	}, WithInterval(time.Second), WithLockExpire(time.Minute))
	scheduler.Start()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("handler did not start")
	}
	stopped := make(chan struct{})
	go func() {
		scheduler.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
		t.Fatal("Stop returned before in-flight handler completed")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("Stop did not return after handler completed")
	}
}

func TestConcurrentLockAndFetch(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := carbon.Now().StdTime()

	store.Insert(ctx, &TaskConfig{
		TaskCode: "shared",
		Status:   StatusEnabled,
		NextRun:  now.Add(-time.Hour),
	})

	var winners atomic.Int64
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.LockAndFetch(ctx, now, 30*time.Second)
			if err == nil {
				winners.Add(1)
			}
		}()
	}
	wg.Wait()

	// only one instance should have won the lock
	if winners.Load() != 1 {
		t.Fatalf("expected only 1 winner, got %d", winners.Load())
	}
}
