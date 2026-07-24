package cronjob

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"zero-service/common/crontask"
	"zero-service/facade/streamevent/streamevent"

	"google.golang.org/grpc"
)

type fakeEventClient struct {
	response *streamevent.HandleCronJobEventRes
	err      error
	request  *streamevent.HandleCronJobEventReq
}

func (f *fakeEventClient) HandleCronJobEvent(_ context.Context, in *streamevent.HandleCronJobEventReq, _ ...grpc.CallOption) (*streamevent.HandleCronJobEventRes, error) {
	f.request = in
	return f.response, f.err
}

func TestEventHandlerSuccessKeepsScheduledTime(t *testing.T) {
	client := &fakeEventClient{response: &streamevent.HandleCronJobEventRes{
		Receipt: streamevent.CronJobReceiptPb_CRON_JOB_RECEIPT_SUCCESS,
	}}
	handler := NewEventHandler(client)
	task := eventHandlerTask(t)
	if err := handler(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	if client.request.ScheduledTime != "2026-07-24 11:00:00" {
		t.Fatalf("scheduled time = %q", client.request.ScheduledTime)
	}
	if client.request.JobId != task.ID || client.request.DeptCode != "D001" || client.request.Type != "inspection" {
		t.Fatalf("unexpected callback request: %+v", client.request)
	}
}

func TestEventHandlerTaskNotFoundRequestsDelete(t *testing.T) {
	client := &fakeEventClient{response: &streamevent.HandleCronJobEventRes{
		Receipt: streamevent.CronJobReceiptPb_CRON_JOB_RECEIPT_TASK_NOT_FOUND,
		Message: "业务任务不存在",
	}}
	err := NewEventHandler(client)(context.Background(), eventHandlerTask(t))
	if !errors.Is(err, crontask.ErrDeleteTask) {
		t.Fatalf("expected ErrDeleteTask, got %v", err)
	}
}

func TestEventHandlerUnknownAndRPCErrorRetry(t *testing.T) {
	tests := []struct {
		name   string
		client *fakeEventClient
	}{
		{name: "unknown", client: &fakeEventClient{response: &streamevent.HandleCronJobEventRes{}}},
		{name: "rpc error", client: &fakeEventClient{err: context.DeadlineExceeded}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := NewEventHandler(test.client)(context.Background(), eventHandlerTask(t)); err == nil || errors.Is(err, crontask.ErrDeleteTask) {
				t.Fatalf("expected ordinary retry error, got %v", err)
			}
		})
	}
}

func eventHandlerTask(t *testing.T) *crontask.TaskConfig {
	t.Helper()
	rule, _ := json.Marshal(map[string]any{"freq": 3})
	extra, err := MarshalExtra(&CronJobExtra{
		DeptCode:  "D001",
		Type:      "inspection",
		StartTime: "2026-07-01 00:00:00",
		EndTime:   "2026-07-31 23:59:59",
		Rule:      rule,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &crontask.TaskConfig{
		ID:       "job-1",
		TaskCode: "task-1",
		TaskName: "test",
		Payload:  json.RawMessage(`{"id":1}`),
		Extra:    extra,
		NextRun:  time.Date(2026, 7, 24, 11, 0, 0, 0, time.Local),
	}
}
