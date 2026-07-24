package crontask

import (
	"context"
	"testing"
	"time"
)

func TestManualExecutionContext(t *testing.T) {
	runAt := time.Date(2026, 7, 24, 16, 30, 0, 0, time.Local)
	ctx := WithManualExecution(context.Background(), "SUB001_TASK001_20260724163000", runAt)

	gotID, gotRunAt, ok := ManualExecutionFromContext(ctx)
	if !ok {
		t.Fatal("expected manual execution metadata")
	}
	if gotID != "SUB001_TASK001_20260724163000" {
		t.Fatalf("task patrolled id = %q", gotID)
	}
	if !gotRunAt.Equal(runAt) {
		t.Fatalf("run at = %v, want %v", gotRunAt, runAt)
	}
}

func TestManualExecutionContextRejectsIncompleteMetadata(t *testing.T) {
	if _, _, ok := ManualExecutionFromContext(context.Background()); ok {
		t.Fatal("unexpected metadata from empty context")
	}
	ctx := WithManualExecution(context.Background(), "", time.Now())
	if _, _, ok := ManualExecutionFromContext(ctx); ok {
		t.Fatal("unexpected metadata with empty task patrolled id")
	}
}
