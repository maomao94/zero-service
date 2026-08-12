package cronjob

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"zero-service/app/trigger/model/gormmodel"
	"zero-service/common/crontask"
	"zero-service/common/gormx"
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
	req := client.request
	if req.ScheduledTime != "2026-07-24 11:00:00" {
		t.Fatalf("scheduled time = %q", req.ScheduledTime)
	}
	// 身份、业务扩展（Type/GroupId/Description/Ext1-5）、机构编码与本次原计划时间均来自扁平映射。
	if req.JobId != task.ID || req.TaskCode != task.TaskCode || req.TaskName != task.TaskName {
		t.Fatalf("identity fields mismatch: %+v", req)
	}
	if req.Type != "inspection" || req.GroupId != "G-1" || req.Description != "巡检任务" {
		t.Fatalf("business extension fields mismatch: %+v", req)
	}
	for i, ext := range []string{req.Ext1, req.Ext2, req.Ext3, req.Ext4, req.Ext5} {
		if want := "e" + string(rune('1'+i)); ext != want {
			t.Fatalf("Ext%d = %q, want %q", i+1, ext, want)
		}
	}
	if req.DeptCode != "D001" {
		t.Fatalf("dept code = %q", req.DeptCode)
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

func TestLoggingEventHandlerStoresGrpcMessage(t *testing.T) {
	db := newCronJobTestDB(t)
	client := &fakeEventClient{response: &streamevent.HandleCronJobEventRes{
		Receipt: streamevent.CronJobReceiptPb_CRON_JOB_RECEIPT_SUCCESS,
		Message: "业务处理成功",
	}}
	handler := NewLoggingEventHandler(&gormx.DB{DB: db}, client)
	if err := handler(context.Background(), eventHandlerTask(t)); err != nil {
		t.Fatal(err)
	}
	var log gormmodel.CronExecLog
	if err := db.Order("create_time DESC").First(&log).Error; err != nil {
		t.Fatal(err)
	}
	if log.Message != "业务处理成功" || log.ErrorMessage != "" || log.Status != 1 {
		t.Fatalf("unexpected cron execution log: %+v", log)
	}
}

func TestLoggingEventHandlerStoresBusinessErrorMessage(t *testing.T) {
	tests := []struct {
		name    string
		receipt streamevent.CronJobReceiptPb
		message string
	}{
		{name: "task not found", receipt: streamevent.CronJobReceiptPb_CRON_JOB_RECEIPT_TASK_NOT_FOUND, message: "业务任务不存在"},
		{name: "unknown", receipt: streamevent.CronJobReceiptPb_CRON_JOB_RECEIPT_UNKNOWN, message: "未知业务回执"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := newCronJobTestDB(t)
			client := &fakeEventClient{response: &streamevent.HandleCronJobEventRes{Receipt: test.receipt, Message: test.message}}
			if err := NewLoggingEventHandler(&gormx.DB{DB: db}, client)(context.Background(), eventHandlerTask(t)); err == nil {
				t.Fatal("expected business receipt error")
			}
			var log gormmodel.CronExecLog
			if err := db.Order("create_time DESC").First(&log).Error; err != nil {
				t.Fatal(err)
			}
			if log.Message != test.message || log.ErrorMessage == "" || log.Status != 0 {
				t.Fatalf("unexpected cron execution log: %+v", log)
			}
		})
	}
}

func TestLoggingEventHandlerLeavesMessageEmptyWithoutResponse(t *testing.T) {
	tests := []struct {
		name   string
		client *fakeEventClient
	}{
		{name: "rpc error", client: &fakeEventClient{err: context.DeadlineExceeded}},
		{name: "nil response", client: &fakeEventClient{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := newCronJobTestDB(t)
			if err := NewLoggingEventHandler(&gormx.DB{DB: db}, test.client)(context.Background(), eventHandlerTask(t)); err == nil {
				t.Fatal("expected callback error")
			}
			var log gormmodel.CronExecLog
			if err := db.Order("create_time DESC").First(&log).Error; err != nil {
				t.Fatal(err)
			}
			if log.Message != "" || log.ErrorMessage == "" || log.Status != 0 {
				t.Fatalf("unexpected cron execution log: %+v", log)
			}
		})
	}
}

func eventHandlerTask(t *testing.T) *crontask.TaskConfig {
	t.Helper()
	rule, _ := json.Marshal(map[string]any{"freq": 3})
	extra, err := MarshalExtra(&CronJobExtra{
		DeptCode:    "D001",
		Type:        "inspection",
		GroupId:     "G-1",
		Description: "巡检任务",
		Ext1:        "e1",
		Ext2:        "e2",
		Ext3:        "e3",
		Ext4:        "e4",
		Ext5:        "e5",
		Rule:        rule,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &crontask.TaskConfig{
		ID:            "job-1",
		TaskCode:      "task-1",
		TaskName:      "test",
		Payload:       json.RawMessage(`{"id":1}`),
		Extra:         extra,
		ScheduledTime: time.Date(2026, 7, 24, 11, 0, 0, 0, time.Local),
	}
}
