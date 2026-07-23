package crontask

import (
	"context"
	"sync"
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

func TestMemoryStoreListEnabled(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	store.Insert(ctx, &TaskConfig{TaskCode: "a", Status: StatusEnabled})
	store.Insert(ctx, &TaskConfig{TaskCode: "b", Status: StatusDisabled})
	store.Insert(ctx, &TaskConfig{TaskCode: "c", Status: StatusEnabled})

	list, err := store.ListEnabled(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 enabled, got %d", len(list))
	}
}

func TestMemoryStoreUpdateStatus(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	cfg := &TaskConfig{TaskCode: "t", Status: StatusEnabled}
	if err := store.Insert(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	if err := store.UpdateStatus(ctx, cfg.ID, StatusDisabled); err != nil {
		t.Fatal(err)
	}

	got, _ := store.GetByCode(ctx, "t")
	if got.Status != StatusDisabled {
		t.Fatalf("expected disabled, got %v", got.Status)
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

	got, err := store.LockAndFetch(ctx, now, 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}

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
	if stored.Version != 1 {
		t.Fatalf("expected version 1, got %d", stored.Version)
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
		got, _ := store.LockAndFetch(ctx, now, 30*time.Second)
		seen[got.TaskCode] = true
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

func TestMemoryStoreUpdateNextRun(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := carbon.Now().StdTime()

	store.Insert(ctx, &TaskConfig{TaskCode: "t", Status: StatusEnabled, NextRun: now.Add(-time.Hour)})

	cfg, _ := store.LockAndFetch(ctx, now, 30*time.Second)
	newNext := now.Add(time.Hour)

	err := store.UpdateNextRun(ctx, cfg.ID, newNext, now)
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
	ctx := context.Background()

	store.Insert(ctx, &TaskConfig{
		TaskCode: "t",
		Status:   StatusEnabled,
		NextRun:  carbon.Now().StdTime().Add(time.Hour),
	})

	var mu sync.Mutex
	executed := false
	handler := func(ctx context.Context, task *TaskConfig) error {
		mu.Lock()
		executed = true
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
	mu.Unlock()
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
	// should not panic
	time.Sleep(100 * time.Millisecond)
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

	var mu sync.Mutex
	winners := make(map[int64]bool)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := store.LockAndFetch(ctx, now, 30*time.Second)
			if err == nil {
				mu.Lock()
				winners[got.Version] = true
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	// only one instance should have won the lock
	if len(winners) > 1 {
		t.Fatalf("expected only 1 winner, got %d", len(winners))
	}
}
