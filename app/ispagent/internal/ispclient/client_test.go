package ispclient

import (
	"context"
	"strings"
	"testing"
)

func TestTaskRun(t *testing.T) {
	client := &IspClient{}
	if err := client.runTask(context.Background(), "TASK001"); err == nil || !strings.Contains(err.Error(), "任务调度器未初始化") {
		t.Fatalf("expected uninitialized task runner error, got %v", err)
	}

	var got string
	client.SetTaskRun(func(_ context.Context, taskCode string) error {
		got = taskCode
		return nil
	})
	if err := client.runTask(context.Background(), "TASK001"); err != nil {
		t.Fatalf("run task: %v", err)
	}
	if got != "TASK001" {
		t.Fatalf("task code = %q, want TASK001", got)
	}

	client.SetTaskRun(nil)
	if err := client.runTask(context.Background(), "TASK001"); err == nil {
		t.Fatal("expected task runner error after clearing closure")
	}
}
