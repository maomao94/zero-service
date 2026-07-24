package svc

import (
	"context"
	"testing"
	"time"

	ctask "zero-service/app/ispagent/internal/crontask"
	"zero-service/common/crontask"
)

func TestTaskExecutionUsesManualIdentity(t *testing.T) {
	scheduledAt := time.Date(2026, 7, 25, 10, 0, 0, 0, time.Local)
	manualAt := time.Date(2026, 7, 24, 16, 30, 0, 0, time.Local)
	task := &crontask.TaskConfig{TaskCode: "TASK001", NextRun: scheduledAt}
	fields := &ctask.IspTaskFields{SubstationCode: "SUB001"}
	ctx := ctask.WithManualExecution(context.Background(), "SUB001_TASK001_20260724163000", manualAt)

	runAt, taskPatrolledID := taskExecution(ctx, task, fields)
	if taskPatrolledID != "SUB001_TASK001_20260724163000" {
		t.Fatalf("task patrolled id = %q", taskPatrolledID)
	}
	if !runAt.Equal(manualAt) {
		t.Fatalf("run at = %v, want %v", runAt, manualAt)
	}
}

func TestTaskExecutionUsesScheduledIdentity(t *testing.T) {
	scheduledAt := time.Date(2026, 7, 25, 10, 0, 0, 0, time.Local)
	task := &crontask.TaskConfig{TaskCode: "TASK001", NextRun: scheduledAt}
	fields := &ctask.IspTaskFields{SubstationCode: "SUB001"}

	runAt, taskPatrolledID := taskExecution(context.Background(), task, fields)
	if taskPatrolledID != "SUB001_TASK001_20260725100000" {
		t.Fatalf("task patrolled id = %q", taskPatrolledID)
	}
	if !runAt.Equal(scheduledAt) {
		t.Fatalf("run at = %v, want %v", runAt, scheduledAt)
	}
}
