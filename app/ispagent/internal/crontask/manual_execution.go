package crontask

import (
	"context"
	"time"
)

type manualExecution struct {
	taskPatrolledID string
	runAt           time.Time
}

type manualExecutionContextKey struct{}

// WithManualExecution 记录 ISP 立即执行的巡视任务标识和执行时间。
func WithManualExecution(ctx context.Context, taskPatrolledID string, runAt time.Time) context.Context {
	return context.WithValue(ctx, manualExecutionContextKey{}, manualExecution{
		taskPatrolledID: taskPatrolledID,
		runAt:           runAt,
	})
}

// ManualExecutionFromContext 读取 ISP 立即执行元数据。
func ManualExecutionFromContext(ctx context.Context) (taskPatrolledID string, runAt time.Time, ok bool) {
	execution, ok := ctx.Value(manualExecutionContextKey{}).(manualExecution)
	if !ok || execution.taskPatrolledID == "" || execution.runAt.IsZero() {
		return "", time.Time{}, false
	}
	return execution.taskPatrolledID, execution.runAt, true
}
