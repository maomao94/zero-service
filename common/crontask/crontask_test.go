package crontask

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dromara/carbon/v2"
	"github.com/teambition/rrule-go"
)

func testRRuleSet(rule string) string {
	return "DTSTART:20260727T000000Z\nRRULE:" + rule
}

func TestNextAfterRejectsBareRRule(t *testing.T) {
	after := time.Date(2026, 7, 27, 9, 0, 0, 0, time.UTC)
	if _, err := NextAfter("FREQ=DAILY;BYHOUR=10;BYMINUTE=30;BYSECOND=0", after); err == nil {
		t.Fatal("NextAfter must reject bare RRULE")
	}
	if err := ValidateRRule("FREQ=DAILY;BYHOUR=10;BYMINUTE=30;BYSECOND=0"); err == nil {
		t.Fatal("ValidateRRule must reject bare RRULE")
	}
}

func TestNextAfterSupportsCRLFRRuleSet(t *testing.T) {
	value := "DTSTART;TZID=Asia/Shanghai:20260727T090000\r\n" +
		"RRULE:FREQ=DAILY;COUNT=2\r\n" +
		"EXDATE;TZID=Asia/Shanghai:20260728T090000"
	after := time.Date(2026, 7, 27, 0, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	next, err := NextAfter(value, after)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 7, 27, 9, 0, 0, 0, next.Location())
	if !next.Equal(want) {
		t.Fatalf("next = %v, want %v", next, want)
	}
	if err := ValidateRRule(value); err != nil {
		t.Fatalf("ValidateRRule(CRLF set): %v", err)
	}
}

func TestSchedulerPreviewNextRunsHonorsExdateAndCount(t *testing.T) {
	after := time.Date(2026, 7, 27, 8, 0, 0, 0, time.UTC)
	task := &TaskConfig{RRuleStr: "DTSTART:20260727T090000Z\n" +
		"RRULE:FREQ=DAILY;COUNT=5\n" +
		"EXDATE:20260728T090000Z"}
	scheduler := NewScheduler(nil, nil)

	runs, err := scheduler.PreviewNextRuns(task, after, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []time.Time{
		time.Date(2026, 7, 27, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 29, 9, 0, 0, 0, time.UTC),
	}
	if len(runs) != len(want) {
		t.Fatalf("runs = %v, want %v", runs, want)
	}
	for i := range want {
		if !runs[i].Equal(want[i]) {
			t.Fatalf("runs[%d] = %v, want %v", i, runs[i], want[i])
		}
	}
	strictRuns, err := scheduler.PreviewNextRuns(task, want[0], 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(strictRuns) != 1 || !strictRuns[0].Equal(want[1]) {
		t.Fatalf("strict-after runs = %v, want %v", strictRuns, want[1])
	}
}

func TestSchedulerPreviewNextRunsAppliesInvalidTimeFilter(t *testing.T) {
	after := time.Date(2026, 7, 27, 8, 0, 0, 0, time.UTC)
	task := &TaskConfig{RRuleStr: "DTSTART:20260727T080000Z\nRRULE:FREQ=HOURLY;COUNT=8"}
	filterCalls := 0
	scheduler := NewScheduler(nil, nil, WithInvalidTimeFilter(func(task *TaskConfig, next time.Time) time.Time {
		filterCalls++
		for next.Hour() >= 9 && next.Hour() <= 12 {
			var err error
			next, err = NextAfter(task.RRuleStr, next)
			if err != nil {
				t.Fatalf("advance filtered time: %v", err)
			}
		}
		return next
	}))

	runs, err := scheduler.PreviewNextRuns(task, after, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 || runs[0].Hour() != 13 || runs[1].Hour() != 14 {
		t.Fatalf("filtered runs = %v", runs)
	}
	if filterCalls != 2 {
		t.Fatalf("filter calls = %d, want 2 bounded result iterations", filterCalls)
	}
}

func TestSchedulerPreviewNextRunsRejectsNonAdvancingFilterResult(t *testing.T) {
	after := time.Date(2026, 7, 27, 8, 0, 0, 0, time.UTC)
	task := &TaskConfig{RRuleStr: "DTSTART:20260727T090000Z\nRRULE:FREQ=HOURLY;COUNT=3"}
	scheduler := NewScheduler(nil, nil, WithInvalidTimeFilter(func(_ *TaskConfig, _ time.Time) time.Time {
		return after
	}))
	if _, err := scheduler.PreviewNextRuns(task, after, 1); err == nil {
		t.Fatal("non-advancing filter result must return an error")
	}
}

func TestSchedulerPreviewNextRunsExhausted(t *testing.T) {
	task := &TaskConfig{RRuleStr: testRRuleSet("FREQ=DAILY;COUNT=1")}
	runs, err := NewScheduler(nil, nil).PreviewNextRuns(task, time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 0 {
		t.Fatalf("exhausted runs = %v, want empty", runs)
	}
}

func TestSchedulerPreviewNextRunsRejectsInvalidInputs(t *testing.T) {
	after := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	if _, err := (*Scheduler)(nil).PreviewNextRuns(&TaskConfig{RRuleStr: testRRuleSet("FREQ=DAILY")}, after, 1); err == nil {
		t.Fatal("nil scheduler must return an error")
	}
	if _, err := NewScheduler(nil, nil).PreviewNextRuns(nil, after, 1); err == nil {
		t.Fatal("nil task must return an error")
	}
	if _, err := NewScheduler(nil, nil).PreviewNextRuns(&TaskConfig{}, after, 1); err == nil {
		t.Fatal("empty recurring rule must return an error")
	}
}

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
	got, err = store.GetByID(ctx, cfg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != cfg.ID || got.TaskCode != cfg.TaskCode {
		t.Fatalf("unexpected task by ID: %+v", got)
	}
}

func TestMemoryStoreGetByCodeNotFound(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.GetByCode(context.Background(), "nonexistent")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	_, err = store.GetByID(context.Background(), "nonexistent")
	if err != ErrNotFound {
		t.Fatalf("GetByID expected ErrNotFound, got %v", err)
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

func TestMemoryStoreEnablePreservesOneShotSchedule(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	scheduled := time.Now().Add(time.Hour).Truncate(time.Second)
	cfg := &TaskConfig{TaskCode: "one-shot-enable", Status: StatusEnabled, NextRun: scheduled}
	if err := store.Insert(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	if err := store.Disable(ctx, cfg.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.Enable(ctx, cfg.ID); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetByID(ctx, cfg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.NextRun.Equal(scheduled) {
		t.Fatalf("one-shot next run = %v, want preserved %v", got.NextRun, scheduled)
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

	err := store.Complete(ctx, claim.Task.ID, claim.LockedUntil, Completion{NextRun: newNext, LastRun: now, LastScheduledRun: claim.Task.ScheduledTime})
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
	if !got.LastScheduledRun.Equal(claim.Task.ScheduledTime) {
		t.Fatalf("expected last scheduled run %v, got %v", claim.Task.ScheduledTime, got.LastScheduledRun)
	}
}

func TestMemoryStoreCompleteWithZeroHistoryPreservesHistory(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := time.Date(2026, 7, 27, 9, 0, 0, 0, time.Local)
	originalLastRun := now.Add(-2 * time.Hour)
	originalScheduledRun := now.Add(-3 * time.Hour)
	cfg := &TaskConfig{
		TaskCode: "skip-history", Status: StatusEnabled, NextRun: now.Add(-time.Hour),
		LastRun: originalLastRun, LastScheduledRun: originalScheduledRun,
	}
	if err := store.Insert(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	claim, err := store.LockAndFetch(ctx, now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Complete(ctx, cfg.ID, claim.LockedUntil, Completion{NextRun: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	got, _ := store.GetByID(ctx, cfg.ID)
	if !got.LastRun.Equal(originalLastRun) || !got.LastScheduledRun.Equal(originalScheduledRun) {
		t.Fatalf("zero completion history changed successful runs: %+v", got)
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
	if err := store.Complete(ctx, cfg.ID, claim.LockedUntil, Completion{NextRun: now.Add(time.Hour), LastRun: now}); !errors.Is(err, ErrNotFound) {
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
	if err := store.Complete(ctx, cfg.ID, claim.LockedUntil, Completion{NextRun: nextRun, LastRun: now, LastScheduledRun: claim.Task.ScheduledTime}); err != nil {
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

	originalScheduledRun := carbon.Now().StdTime().Add(-time.Hour)
	store.Insert(ctx, &TaskConfig{
		TaskCode:         "t",
		Status:           StatusEnabled,
		NextRun:          carbon.Now().StdTime().Add(time.Hour),
		LastScheduledRun: originalScheduledRun,
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

	if _, err := s.RunNow(ctx, "t"); err != nil {
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
	if !got.LastScheduledRun.Equal(originalScheduledRun) {
		t.Fatalf("RunNow changed last scheduled run: %v", got.LastScheduledRun)
	}
}

func TestRunNowProvidesExecutionTimeForZeroNextRun(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	if err := store.Insert(ctx, &TaskConfig{
		TaskCode: "manual-exhausted",
		Status:   StatusEnabled,
		RRuleStr: testRRuleSet("FREQ=DAILY;COUNT=1"),
	}); err != nil {
		t.Fatal(err)
	}

	executed := make(chan time.Time, 1)
	s := NewScheduler(store, func(ctx context.Context, task *TaskConfig) error {
		executed <- task.ScheduledTime
		return nil
	})
	if _, err := s.RunNow(ctx, "manual-exhausted"); err != nil {
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
	rruleStr := testRRuleSet("FREQ=DAILY;INTERVAL=1")

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

	lastRun := time.Now().Add(-time.Hour)
	lastScheduledRun := lastRun.Add(-time.Hour)
	cfg := &TaskConfig{
		TaskCode: "t", TaskName: "test", Status: StatusEnabled,
		LastRun: lastRun, LastScheduledRun: lastScheduledRun,
	}
	if err := store.Insert(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	cfg.TaskName = "updated"
	if err := store.Disable(ctx, cfg.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.Update(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	got, _ := store.GetByCode(ctx, "t")
	if got.TaskName != "updated" {
		t.Fatalf("expected updated, got %s", got.TaskName)
	}
	if !got.LastRun.Equal(lastRun) || !got.LastScheduledRun.Equal(lastScheduledRun) {
		t.Fatalf("Update changed execution history: %+v", got)
	}
	if got.Status != StatusDisabled {
		t.Fatalf("Update changed control status: %+v", got)
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

func TestMemoryStoreUpdateMissingReturnsErrUpdate(t *testing.T) {
	store := NewMemoryStore()
	err := store.Update(context.Background(), &TaskConfig{ID: "missing", TaskCode: "missing"})
	if !errors.Is(err, ErrUpdate) {
		t.Fatalf("missing update = %v, want ErrUpdate", err)
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
		RRuleStr: testRRuleSet("FREQ=DAILY;COUNT=1"),
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
		RRuleStr: testRRuleSet("FREQ=DAILY;COUNT=1"),
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
		RRuleStr: testRRuleSet("FREQ=DAILY;INTERVAL=1"),
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

func TestExecuteTaskSuccessRecordsActualAndScheduledTimes(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	scheduledRun := time.Now().Add(-time.Hour).Truncate(time.Second)
	cfg := &TaskConfig{
		TaskCode: "delayed-success", Status: StatusEnabled,
		RRuleStr: testRRuleSet("FREQ=DAILY"), NextRun: scheduledRun,
	}
	if err := store.Insert(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	claim, err := store.LockAndFetch(ctx, time.Now(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	handlerReturnedAfter := time.Now().Add(20 * time.Millisecond)
	scheduler := NewScheduler(store, func(context.Context, *TaskConfig) error {
		time.Sleep(time.Until(handlerReturnedAfter))
		return nil
	})
	scheduler.executeTask(claim)

	got, err := store.GetByID(ctx, cfg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.LastRun.Before(handlerReturnedAfter) {
		t.Fatalf("LastRun = %v, want actual completion at or after %v", got.LastRun, handlerReturnedAfter)
	}
	if !got.LastScheduledRun.Equal(scheduledRun) {
		t.Fatalf("LastScheduledRun = %v, want %v", got.LastScheduledRun, scheduledRun)
	}
}

func TestExecuteTaskFailureAndPanicDoNotRecordSuccess(t *testing.T) {
	tests := []struct {
		name    string
		handler Handler
	}{
		{name: "error", handler: func(context.Context, *TaskConfig) error { return errors.New("failed") }},
		{name: "panic", handler: func(context.Context, *TaskConfig) error { panic("failed") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryStore()
			ctx := context.Background()
			now := time.Now().Truncate(time.Second)
			cfg := &TaskConfig{TaskCode: "no-success-" + tt.name, Status: StatusEnabled, RRuleStr: testRRuleSet("FREQ=DAILY"), NextRun: now.Add(-time.Minute)}
			if err := store.Insert(ctx, cfg); err != nil {
				t.Fatal(err)
			}
			claim, err := store.LockAndFetch(ctx, now, time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			NewScheduler(store, tt.handler).executeTask(claim)
			got, _ := store.GetByID(ctx, cfg.ID)
			if !got.LastRun.IsZero() || !got.LastScheduledRun.IsZero() {
				t.Fatalf("failed handler recorded success: %+v", got)
			}
			if !got.NextRun.Equal(claim.LockedUntil) {
				t.Fatalf("failed handler completed claim: next=%v lease=%v", got.NextRun, claim.LockedUntil)
			}
		})
	}
}

func TestExecuteTaskStaleSkipDoesNotRecordSuccess(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	originalLastRun := now.Add(-2 * time.Hour)
	originalScheduledRun := now.Add(-3 * time.Hour)
	cfg := &TaskConfig{
		TaskCode: "stale", Status: StatusEnabled, RRuleStr: testRRuleSet("FREQ=DAILY"), NextRun: now.Add(-time.Hour),
		LastRun: originalLastRun, LastScheduledRun: originalScheduledRun,
	}
	if err := store.Insert(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	claim, err := store.LockAndFetch(ctx, now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	var called bool
	NewScheduler(store, func(context.Context, *TaskConfig) error {
		called = true
		return nil
	}, WithMaxDelay(time.Minute)).executeTask(claim)
	got, _ := store.GetByID(ctx, cfg.ID)
	if called {
		t.Fatal("stale task invoked handler")
	}
	if !got.LastRun.Equal(originalLastRun) || !got.LastScheduledRun.Equal(originalScheduledRun) {
		t.Fatalf("stale skip changed success history: %+v", got)
	}
}

func TestExecuteTaskLostLeaseDoesNotRecordSuccess(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	cfg := &TaskConfig{TaskCode: "lost-execute", Status: StatusEnabled, RRuleStr: testRRuleSet("FREQ=DAILY"), NextRun: now.Add(-time.Minute)}
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
	NewScheduler(store, func(context.Context, *TaskConfig) error { return nil }).executeTask(claim)
	got, _ := store.GetByID(ctx, cfg.ID)
	if !got.LastRun.IsZero() || !got.LastScheduledRun.IsZero() {
		t.Fatalf("lost lease recorded success: %+v", got)
	}
}

func TestExecuteTaskDeleteSignalDeletesTask(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := carbon.Now().StdTime()
	cfg := &TaskConfig{
		TaskCode: "deleted-by-handler",
		Status:   StatusEnabled,
		RRuleStr: testRRuleSet("FREQ=DAILY"),
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
		RRuleStr: testRRuleSet("FREQ=DAILY"),
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
		RRuleStr: testRRuleSet("FREQ=DAILY"),
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
		RRuleStr: testRRuleSet("FREQ=DAILY;INTERVAL=1"),
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

// refNextAfter 是未做起点平移的原始查询实现，作为平移后结果的差分参照。
func refNextAfter(t *testing.T, value string, after time.Time) (time.Time, error) {
	t.Helper()
	set, err := parseRRuleSet(value)
	if err != nil {
		return time.Time{}, err
	}
	return set.After(after, false), nil
}

// refPreviewRuns 是未做起点平移的原始预览实现，作为平移后结果的差分参照。
func refPreviewRuns(t *testing.T, value string, after time.Time, count int) []time.Time {
	t.Helper()
	set, err := parseRRuleSet(value)
	if err != nil {
		t.Fatal(err)
	}
	runs := make([]time.Time, 0, count)
	cursor := after
	for len(runs) < count {
		next := set.After(cursor, false)
		if next.IsZero() {
			break
		}
		runs = append(runs, next)
		cursor = next
	}
	return runs
}

func TestNextAfterMatchesOriginalAfterShift(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 8, 12, 15, 17, 54, 0, loc)
	rules := []struct {
		name string
		rule string
	}{
		{name: "daily-fixed-time", rule: "DTSTART;TZID=Asia/Shanghai:20260101T093000\nRRULE:FREQ=DAILY;BYHOUR=9;BYMINUTE=30;BYSECOND=0"},
		{name: "daily-bymonth", rule: "DTSTART;TZID=Asia/Shanghai:20260101T093000\nRRULE:FREQ=DAILY;BYMONTH=3;BYHOUR=9;BYMINUTE=30;BYSECOND=0"},
		{name: "weekly-byday", rule: "DTSTART;TZID=Asia/Shanghai:20260101T090000\nRRULE:FREQ=WEEKLY;BYDAY=MO,WE;BYHOUR=9;BYMINUTE=0;BYSECOND=0"},
		{name: "weekly-default-phase", rule: "DTSTART;TZID=Asia/Shanghai:20260101T090000\nRRULE:FREQ=WEEKLY;BYHOUR=9;BYMINUTE=0;BYSECOND=0"},
		{name: "monthly-day15", rule: "DTSTART;TZID=Asia/Shanghai:20260101T090000\nRRULE:FREQ=MONTHLY;BYMONTHDAY=15;BYHOUR=9;BYMINUTE=0;BYSECOND=0"},
		{name: "monthly-day31", rule: "DTSTART;TZID=Asia/Shanghai:20260101T090000\nRRULE:FREQ=MONTHLY;BYMONTHDAY=31;BYHOUR=9;BYMINUTE=0;BYSECOND=0"},
		{name: "yearly-bymonth", rule: "DTSTART;TZID=Asia/Shanghai:20260101T090000\nRRULE:FREQ=YEARLY;BYMONTH=1;BYMONTHDAY=1;BYHOUR=9;BYMINUTE=0;BYSECOND=0"},
		{name: "hourly-fixed-minute", rule: "DTSTART;TZID=Asia/Shanghai:20260101T093000\nRRULE:FREQ=HOURLY;BYMINUTE=30;BYSECOND=0"},
		{name: "minutely-fixed-second", rule: "DTSTART;TZID=Asia/Shanghai:20260101T093005\nRRULE:FREQ=MINUTELY;BYSECOND=0"},
		{name: "secondly-fixed-minute", rule: "DTSTART;TZID=Asia/Shanghai:20260101T093000\nRRULE:FREQ=SECONDLY;BYMINUTE=5;BYSECOND=0"},
		{name: "monthly-interval2", rule: "DTSTART;TZID=Asia/Shanghai:20260101T093000\nRRULE:FREQ=MONTHLY;INTERVAL=2;BYMONTHDAY=1;BYHOUR=9;BYMINUTE=30;BYSECOND=0"},
		{name: "daily-interval3", rule: "DTSTART;TZID=Asia/Shanghai:20260101T093000\nRRULE:FREQ=DAILY;INTERVAL=3;BYHOUR=9;BYMINUTE=30;BYSECOND=0"},
		{name: "weekly-interval2-byday", rule: "DTSTART;TZID=Asia/Shanghai:20260101T090000\nRRULE:FREQ=WEEKLY;INTERVAL=2;BYDAY=TH;BYHOUR=9;BYMINUTE=0;BYSECOND=0"},
		{name: "hourly-interval2", rule: "DTSTART;TZID=Asia/Shanghai:20260101T093000\nRRULE:FREQ=HOURLY;INTERVAL=2;BYMINUTE=30;BYSECOND=0"},
		{name: "minutely-interval5", rule: "DTSTART;TZID=Asia/Shanghai:20260101T093005\nRRULE:FREQ=MINUTELY;INTERVAL=5;BYSECOND=0"},
		{name: "yearly-interval2", rule: "DTSTART;TZID=Asia/Shanghai:20230101T090000\nRRULE:FREQ=YEARLY;INTERVAL=2;BYMONTH=1;BYMONTHDAY=1;BYHOUR=9;BYMINUTE=0;BYSECOND=0"},
		{name: "rdate-exdate", rule: "DTSTART;TZID=Asia/Shanghai:20260101T093000\nRRULE:FREQ=DAILY;BYHOUR=9;BYMINUTE=30;BYSECOND=0\nRDATE;TZID=Asia/Shanghai:20260115T100000\nEXDATE;TZID=Asia/Shanghai:20260201T093000"},
	}
	queries := []time.Time{now, now.Add(3 * time.Hour), now.Add(30 * 24 * time.Hour)}
	for _, tt := range rules {
		for _, q := range queries {
			want, err := refNextAfter(t, tt.rule, q)
			if err != nil {
				t.Fatalf("%s: reference query failed: %v", tt.name, err)
			}
			got, err := NextAfter(tt.rule, q)
			if err != nil {
				t.Fatalf("%s: NextAfter failed: %v", tt.name, err)
			}
			if !got.Equal(want) {
				t.Errorf("%s: NextAfter(%v) = %v, want %v", tt.name, q.Format("2006-01-02 15:04:05"), got, want)
			}
		}
	}
}

func TestPreviewNextRunsMatchesOriginalAfterShift(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 8, 12, 15, 17, 54, 0, loc)
	rule := "DTSTART;TZID=Asia/Shanghai:20260101T093000\nRRULE:FREQ=DAILY;BYHOUR=9;BYMINUTE=30;BYSECOND=0\nRDATE;TZID=Asia/Shanghai:20260115T100000\nEXDATE;TZID=Asia/Shanghai:20260201T093000"
	want := refPreviewRuns(t, rule, now, 5)
	got, err := NewScheduler(nil, nil).PreviewNextRuns(&TaskConfig{RRuleStr: rule}, now, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("runs = %v, want %v", got, want)
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Fatalf("runs[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestPreviewNextRunsUserRuleLongGapIsCorrect(t *testing.T) {
	hours := make([]string, 24)
	for i := range hours {
		hours[i] = fmtInt(i)
	}
	minutes := make([]string, 60)
	for i := range minutes {
		minutes[i] = fmtInt(i)
	}
	rule := "DTSTART;TZID=Asia/Shanghai:20200101T000000\nRRULE:FREQ=MINUTELY;UNTIL=20391231T155959Z;BYHOUR=" +
		strings.Join(hours, ",") + ";BYMINUTE=" + strings.Join(minutes, ",") + ";BYSECOND=0"
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 12, 15, 17, 54, 0, loc)

	// 全时段显式规则无相位依赖，用 DTSTART 贴近查询点的同构规则作为正确性参照。
	refRule := strings.Replace(rule, "20200101T000000", "20260701T000000", 1)
	want := refPreviewRuns(t, refRule, now, 10)

	started := time.Now()
	got, err := NewScheduler(nil, nil).PreviewNextRuns(&TaskConfig{RRuleStr: rule}, now, 10)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Errorf("preview of every-minute rule took %v, want < 2s (before fix: ~13s)", elapsed)
	}
	if len(got) != len(want) {
		t.Fatalf("runs = %v, want %v", got, want)
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Fatalf("runs[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func fmtInt(v int) string {
	return fmt.Sprintf("%d", v)
}

func TestShiftSetForQueryFallbackRules(t *testing.T) {
	after := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	stringRules := []string{
		"DTSTART:20260101T090000Z\nRRULE:FREQ=DAILY;COUNT=3;BYHOUR=9",
		"DTSTART:20260101T090000Z\nRRULE:FREQ=YEARLY;BYWEEKNO=1;BYDAY=MO",
		"DTSTART:20260101T090000Z\nRRULE:FREQ=YEARLY;BYYEARDAY=1",
	}
	for _, rule := range stringRules {
		set, err := parseRRuleSet(rule)
		if err != nil {
			t.Fatalf("parse %q: %v", rule, err)
		}
		if shifted := ShiftSetForQuery(set, after); shifted != nil {
			t.Errorf("rule %q must fall back to original set", rule)
		}
		want, err := refNextAfter(t, rule, after)
		if err != nil {
			t.Fatalf("reference query failed for %q: %v", rule, err)
		}
		got, err := NextAfter(rule, after)
		if err != nil {
			t.Fatalf("NextAfter failed for %q: %v", rule, err)
		}
		if !got.Equal(want) {
			t.Errorf("rule %q: NextAfter = %v, want %v", rule, got, want)
		}
	}

	dtstart := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	optionRules := []rrule.ROption{
		{Freq: rrule.MONTHLY, Dtstart: dtstart, Bymonthday: []int{1, 15}, Bysetpos: []int{1}},
		{Freq: rrule.YEARLY, Dtstart: dtstart, Byeaster: []int{1}},
	}
	for _, option := range optionRules {
		rule, err := rrule.NewRRule(option)
		if err != nil {
			t.Fatalf("build rule %v: %v", option, err)
		}
		set := &rrule.Set{}
		set.RRule(rule)
		if shifted := ShiftSetForQuery(set, after); shifted != nil {
			t.Errorf("option %v must fall back to original set", option)
		}
	}
}

func TestShiftSetForQueryNearDtStartIsCorrect(t *testing.T) {
	// DTSTART 与查询点同在一个周期内：无需平移，行为与原始一致。
	after := time.Date(2026, 8, 12, 10, 15, 0, 0, time.UTC)
	rule := "DTSTART:20260812T093000Z\nRRULE:FREQ=HOURLY;BYMINUTE=0"
	set, err := parseRRuleSet(rule)
	if err != nil {
		t.Fatal(err)
	}
	if shifted := ShiftSetForQuery(set, after); shifted != nil {
		t.Fatalf("DTSTART within one period must not be shifted, got %v", shifted.GetDTStart())
	}
	want, err := refNextAfter(t, rule, after)
	if err != nil {
		t.Fatal(err)
	}
	got, err := NextAfter(rule, after)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(want) {
		t.Fatalf("NextAfter = %v, want %v", got, want)
	}

	// anchor 恰好落在查询点上的平移也必须保持结果一致。
	after = time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	rule = "DTSTART:20260812T090000Z\nRRULE:FREQ=HOURLY;BYMINUTE=0"
	want, err = refNextAfter(t, rule, after)
	if err != nil {
		t.Fatal(err)
	}
	got, err = NextAfter(rule, after)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(want) {
		t.Fatalf("NextAfter = %v, want %v", got, want)
	}
}

func TestNextAfterAcrossFallBackDSTMatchesReference(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("location unavailable: %v", err)
	}
	// 2026-11-01 02:00 EDT 回拨到 01:00 EST：跨回拨边界的 duration 加法平移
	// 可能让锚点墙钟与查询点重合，结果必须与原始集一致。
	rules := []struct {
		name string
		rule string
	}{
		{name: "hourly", rule: "DTSTART;TZID=America/New_York:20261030T090000\nRRULE:FREQ=HOURLY;BYMINUTE=0;BYSECOND=0"},
		{name: "minutely", rule: "DTSTART;TZID=America/New_York:20261030T090000\nRRULE:FREQ=MINUTELY;BYSECOND=0"},
	}
	queries := []time.Time{
		time.Date(2026, 11, 2, 9, 0, 0, 0, loc),
		time.Date(2026, 11, 2, 9, 30, 0, 0, loc),
		time.Date(2026, 11, 1, 5, 30, 0, 0, loc),
	}
	for _, tt := range rules {
		for _, q := range queries {
			want, err := refNextAfter(t, tt.rule, q)
			if err != nil {
				t.Fatalf("%s: reference query failed: %v", tt.name, err)
			}
			got, err := NextAfter(tt.rule, q)
			if err != nil {
				t.Fatalf("%s: NextAfter failed: %v", tt.name, err)
			}
			if !got.Equal(want) {
				t.Errorf("%s: NextAfter(%v) = %v, want %v", tt.name, q.Format("2006-01-02 15:04:05 MST"), got, want)
			}
		}
	}
}

func TestNextRunsMatchesReference(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 8, 12, 15, 17, 54, 0, loc)
	rule := "DTSTART;TZID=Asia/Shanghai:20260101T093000\nRRULE:FREQ=DAILY;BYHOUR=9;BYMINUTE=30;BYSECOND=0\nRDATE;TZID=Asia/Shanghai:20260115T100000\nEXDATE;TZID=Asia/Shanghai:20260201T093000"

	want := refPreviewRuns(t, rule, now, 7)
	got, err := NextRuns(rule, now, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("runs = %v, want %v", got, want)
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Fatalf("runs[%d] = %v, want %v", i, got[i], want[i])
		}
	}

	if runs, err := NextRuns(rule, now, 0); err != nil || len(runs) != 0 {
		t.Fatalf("NextRuns count=0 = %v, %v; want empty, nil", runs, err)
	}
	if _, err := NextRuns("", now, 5); err == nil {
		t.Fatal("NextRuns must reject empty rule")
	}
}

func TestShiftDtStartByPeriod(t *testing.T) {
	loc := time.UTC
	date := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, loc) }
	datetime := func(y int, m time.Month, d, hh, mm, ss int) time.Time {
		return time.Date(y, m, d, hh, mm, ss, 0, loc)
	}
	tests := []struct {
		name     string
		dtstart  time.Time
		after    time.Time
		freq     rrule.Frequency
		interval int
		want     time.Time
		wantOK   bool
	}{
		// YEARLY：年差整对齐，直接在 after 之前。
		{name: "yearly-simple", dtstart: date(2020, 6, 15), after: date(2026, 8, 12), freq: rrule.YEARLY, interval: 1, want: date(2026, 6, 15), wantOK: true},
		// YEARLY：+6 年越过 after（9 月晚于 8 月），回退一个间隔到上一个相位 2025-09-15。
		{name: "yearly-retreat-previous-phase", dtstart: date(2020, 9, 15), after: date(2026, 8, 12), freq: rrule.YEARLY, interval: 1, want: date(2025, 9, 15), wantOK: true},
		// YEARLY INTERVAL=2：5 年对齐到 4 年。
		{name: "yearly-interval2-floor", dtstart: date(2020, 6, 15), after: date(2025, 8, 12), freq: rrule.YEARLY, interval: 2, want: date(2024, 6, 15), wantOK: true},
		// YEARLY 闰日：AddDate 进位到 2026-03-01，月/日相位被破坏 → 放弃平移。
		{name: "yearly-feb29-clamp", dtstart: date(2020, 2, 29), after: date(2026, 8, 12), freq: rrule.YEARLY, interval: 1, want: time.Time{}, wantOK: false},
		// YEARLY：未跨满一个间隔（years=0）→ 不平移。
		{name: "yearly-not-yet-one-period", dtstart: date(2020, 6, 15), after: date(2020, 8, 12), freq: rrule.YEARLY, interval: 1, want: time.Time{}, wantOK: false},
		// YEARLY：回退到 dtstart 自身（2026-06-15 越过、2025-06-15 即起点）→ Equal 兜底 false。
		{name: "yearly-retreat-to-dtstart", dtstart: date(2020, 6, 15), after: date(2021, 3, 1), freq: rrule.YEARLY, interval: 1, want: time.Time{}, wantOK: false},

		// MONTHLY：+31 月=2026-08-15 越过 → 回退到 2026-07-15。
		{name: "monthly-retreat-previous-phase", dtstart: date(2024, 1, 15), after: date(2026, 8, 12), freq: rrule.MONTHLY, interval: 1, want: date(2026, 7, 15), wantOK: true},
		// MONTHLY INTERVAL=3：31 月对齐到 30 月（07-15 未越过）。
		{name: "monthly-interval3-floor", dtstart: date(2024, 1, 15), after: date(2026, 8, 12), freq: rrule.MONTHLY, interval: 3, want: date(2026, 7, 15), wantOK: true},
		// MONTHLY 月末：1/31 +2 月=3/31 越过，+1 月=3/02 相位破坏 → 放弃。
		{name: "monthly-day31-clamp", dtstart: date(2024, 1, 31), after: date(2024, 3, 5), freq: rrule.MONTHLY, interval: 1, want: time.Time{}, wantOK: false},
		// MONTHLY：回退到 dtstart（2024-02-15 越过，2024-01-15 即起点）→ Equal 兜底 false。
		{name: "monthly-retreat-to-dtstart", dtstart: date(2024, 1, 15), after: date(2024, 2, 1), freq: rrule.MONTHLY, interval: 1, want: time.Time{}, wantOK: false},
		// MONTHLY：月相位对、日号 15 保持不变 → 平移成功。
		{name: "monthly-day15-preserved", dtstart: date(2024, 1, 15), after: date(2024, 4, 20), freq: rrule.MONTHLY, interval: 1, want: date(2024, 4, 15), wantOK: true},

		// WEEKLY：2024-01-01(周一) 起 954 天对齐到 952 天=136 周，保持周一相位。
		{name: "weekly-954d-floor", dtstart: date(2024, 1, 1), after: date(2026, 8, 12), freq: rrule.WEEKLY, interval: 1, want: date(2026, 8, 10), wantOK: true},
		// WEEKLY INTERVAL=2：954 对齐到 952=偶数周。
		{name: "weekly-interval2-floor", dtstart: date(2024, 1, 1), after: date(2026, 8, 12), freq: rrule.WEEKLY, interval: 2, want: date(2026, 8, 10), wantOK: true},
		// WEEKLY：不足一周即不平移。
		{name: "weekly-too-early", dtstart: date(2024, 1, 1), after: date(2024, 1, 3), freq: rrule.WEEKLY, interval: 1, want: time.Time{}, wantOK: false},
		// WEEKLY：对齐到最近周一（2024-01-08）。
		{name: "weekly-floor-monday", dtstart: date(2024, 1, 1), after: date(2024, 1, 10), freq: rrule.WEEKLY, interval: 1, want: date(2024, 1, 8), wantOK: true},

		// DAILY：954 天恰在 after，整对齐不动。
		{name: "daily-exact", dtstart: date(2024, 1, 1), after: date(2026, 8, 12), freq: rrule.DAILY, interval: 1, want: date(2026, 8, 12), wantOK: true},
		// DAILY INTERVAL=3：955 天对齐到 954 天（divisible-by-3 相位保留）。
		{name: "daily-interval3-floor", dtstart: date(2024, 1, 1), after: date(2026, 8, 13), freq: rrule.DAILY, interval: 3, want: date(2026, 8, 12), wantOK: true},
		// DAILY：不足一个间隔不平移。
		{name: "daily-too-early", dtstart: date(2024, 1, 1), after: date(2024, 1, 2), freq: rrule.DAILY, interval: 100, want: time.Time{}, wantOK: false},

		// HOURLY：652 小时对齐，锚点 14:00，分/秒相位保持。
		{name: "hourly-retreat-previous-phase", dtstart: datetime(2024, 1, 1, 10, 0, 0), after: datetime(2024, 1, 28, 14, 30, 0), freq: rrule.HOURLY, interval: 1, want: datetime(2024, 1, 28, 14, 0, 0), wantOK: true},
		// HOURLY INTERVAL=6：652 对齐到 648。
		{name: "hourly-interval6-floor", dtstart: datetime(2024, 1, 1, 10, 0, 0), after: datetime(2024, 1, 28, 14, 30, 0), freq: rrule.HOURLY, interval: 6, want: datetime(2024, 1, 28, 10, 0, 0), wantOK: true},
		// HOURLY：小时差不足则不平移。
		{name: "hourly-too-early", dtstart: datetime(2024, 1, 1, 10, 0, 0), after: datetime(2024, 1, 1, 10, 59, 0), freq: rrule.HOURLY, interval: 1, want: time.Time{}, wantOK: false},

		// MINUTELY：17.5 分钟截断到 17，对齐到 15。
		{name: "minutely-interval5-floor", dtstart: datetime(2024, 1, 1, 10, 0, 0), after: datetime(2024, 1, 1, 10, 17, 30), freq: rrule.MINUTELY, interval: 5, want: datetime(2024, 1, 1, 10, 15, 0), wantOK: true},
		// MINUTELY：秒相位保持（从 :15 起的 15 分钟对齐)。
		{name: "minutely-second-phase-preserved", dtstart: datetime(2024, 1, 1, 10, 0, 15), after: datetime(2024, 1, 1, 10, 17, 30), freq: rrule.MINUTELY, interval: 5, want: datetime(2024, 1, 1, 10, 15, 15), wantOK: true},

		// SECONDLY：50 秒对齐到 45。
		{name: "secondly-interval15-floor", dtstart: datetime(2024, 1, 1, 10, 0, 0), after: datetime(2024, 1, 1, 10, 0, 50), freq: rrule.SECONDLY, interval: 15, want: datetime(2024, 1, 1, 10, 0, 45), wantOK: true},

		// 前置守卫：after 不晚于 dtstart 或 dtstart 为零。
		{name: "after-not-after-dtstart", dtstart: date(2026, 8, 12), after: date(2026, 8, 12), freq: rrule.DAILY, interval: 1, want: time.Time{}, wantOK: false},
		{name: "zero-dtstart", dtstart: time.Time{}, after: date(2026, 8, 12), freq: rrule.DAILY, interval: 1, want: time.Time{}, wantOK: false},
		// interval < 1 归一化为 1。
		{name: "interval-zero-normalized", dtstart: date(2024, 1, 1), after: date(2024, 1, 10), freq: rrule.DAILY, interval: 0, want: date(2024, 1, 10), wantOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := shiftDtStartByPeriod(tt.dtstart, tt.after, tt.freq, tt.interval)
			if ok != tt.wantOK {
				t.Fatalf("shiftDtStartByPeriod(%v, %v, %d, %d) ok = %v, want %v", tt.dtstart.Format("2006-01-02 15:04:05"), tt.after.Format("2006-01-02 15:04:05"), tt.freq, tt.interval, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if !got.Equal(tt.want) {
				t.Errorf("shiftDtStartByPeriod(%v, %v, %d, %d) = %v, want %v", tt.dtstart.Format("2006-01-02 15:04:05"), tt.after.Format("2006-01-02 15:04:05"), tt.freq, tt.interval, got.Format("2006-01-02 15:04:05"), tt.want.Format("2006-01-02 15:04:05"))
			}
			if got.After(tt.after) {
				t.Errorf("shiftDtStartByPeriod shifted %v must not be after query point %v", got, tt.after)
			}
		})
	}
}
