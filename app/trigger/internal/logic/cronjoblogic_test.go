package logic

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"zero-service/app/trigger/internal/cronjob"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/carbonx"
	"zero-service/common/crontask"
	"zero-service/common/gormx"
	"zero-service/common/rrulex"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCronJobLifecycle(t *testing.T) {
	store, db := newCronJobLogicTestStore(t)
	serviceContext := &svc.ServiceContext{DB: db, CronJobStore: store}
	ctx := context.Background()
	nextHour := time.Now().Add(time.Hour).Hour()

	created, err := NewCreateCronJobLogic(ctx, serviceContext).CreateCronJob(&trigger.CreateCronJobReq{
		TaskCode:    "logic-lifecycle",
		TaskName:    "生命周期测试",
		Type:        "test",
		DeptCode:    "D001",
		Payload:     `{"id":1}`,
		LockTimeout: 90_000,
		Rule: &trigger.PlanRulePb{
			Freq:    3,
			Hours:   []int32{int32(nextHour)},
			Minutes: []int32{0},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.JobId == "" {
		t.Fatal("create must return jobId")
	}
	if created.GroupId == "" {
		t.Fatal("create must return generated groupId")
	}
	loaded, err := store.GetByID(ctx, created.JobId)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.LockTimeout != 90*time.Second {
		t.Fatalf("lock timeout = %v, want %v", loaded.LockTimeout, 90*time.Second)
	}

	disable := NewDisableCronJobLogic(ctx, serviceContext)
	for i := range 2 {
		if _, err := disable.DisableCronJob(&trigger.DisableCronJobReq{JobId: created.JobId}); err != nil {
			t.Fatalf("disable attempt %d: %v", i+1, err)
		}
	}
	enable := NewEnableCronJobLogic(ctx, serviceContext)
	for i := range 2 {
		if _, err := enable.EnableCronJob(&trigger.EnableCronJobReq{JobId: created.JobId}); err != nil {
			t.Fatalf("enable attempt %d: %v", i+1, err)
		}
	}

	deleteLogic := NewDeleteCronJobLogic(ctx, serviceContext)
	for i := range 2 {
		if _, err := deleteLogic.DeleteCronJob(&trigger.DeleteCronJobReq{JobId: created.JobId}); err != nil {
			t.Fatalf("delete attempt %d: %v", i+1, err)
		}
	}
	if _, err := store.GetByID(ctx, created.JobId); !errors.Is(err, crontask.ErrNotFound) {
		t.Fatalf("deleted job must not be queryable: %v", err)
	}
}

func TestUpdateCronJobRejectsStableIdentityChange(t *testing.T) {
	t.Skip("UpdateCronJobReq no longer carries Type, GroupId or DeptCode")
}

func TestCreateCronJobRemainsStrict(t *testing.T) {
	store, db := newCronJobLogicTestStore(t)
	serviceContext := &svc.ServiceContext{DB: db, CronJobStore: store}
	logic := NewCreateCronJobLogic(context.Background(), serviceContext)
	req := validCronJobRequest("strict-create")
	if _, err := logic.CreateCronJob(req); err != nil {
		t.Fatal(err)
	}
	if _, err := logic.CreateCronJob(req); !tool.IsErrorByPbCode(err, extproto.Code__1_02_RECORD_ALREADY_EXIST) {
		t.Fatalf("duplicate create error = %v, want RECORD_ALREADY_EXIST", err)
	}
}

func TestBuildCronJobTaskCompilesValidRule(t *testing.T) {
	task, _, err := buildCronJobTask(cronJobTaskData{
		taskCode: "valid-rule", taskName: "合法规则", taskType: "test",
		groupID: "G001", description: "rule test", deptCode: "D001",
		rule: &trigger.PlanRulePb{
			Freq: 3, Hours: []int32{11}, Minutes: []int32{0},
		},
		startTime: "2026-08-01 00:00:00", endTime: "2026-08-31 23:59:59",
		excludeDates: []string{"2026-08-12"}, payload: `{"id":1}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := rrulex.Validate(task.RRuleStr); err != nil {
		t.Fatalf("compiled rule is invalid: %v", err)
	}
	extra, err := cronjob.ParseExtra(task.Extra)
	if err != nil {
		t.Fatal(err)
	}
	if extra.DeptCode != "D001" || extra.Type != "test" || extra.GroupId != "G001" || len(extra.Rule) == 0 {
		t.Fatalf("unexpected cron job extra: %+v", extra)
	}
}

func TestBuildCronJobTaskCompilesExactTimesIntoSetAndExtra(t *testing.T) {
	now := time.Date(2099, 7, 1, 0, 0, 0, 0, time.Local)
	specified := "2099-07-01 08:00:00"
	task, _, err := buildCronJobTask(cronJobTaskData{
		taskCode: "exact-rule", taskName: "精确时间", taskType: "test", deptCode: "D001",
		rule:      &trigger.PlanRulePb{Freq: 3, Hours: []int32{9}, Minutes: []int32{0}},
		startTime: "2099-07-01 00:00:00", endTime: "2099-07-31 23:59:59",
		specifiedTimes: []string{specified},
		excludedTimes:  []string{"2099-07-01 09:00:00"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(task.RRuleStr, "RDATE") || !strings.Contains(task.RRuleStr, "EXDATE") {
		t.Fatalf("compiled set lacks exact-time components: %s", task.RRuleStr)
	}
	extra, err := cronjob.ParseExtra(task.Extra)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(extra.SpecifiedTimes, []string{specified}) || !reflect.DeepEqual(extra.ExcludedTimes, []string{"2099-07-01 09:00:00"}) {
		t.Fatalf("exact times missing from extra: %+v", extra)
	}
	wantNext, err := time.ParseInLocation("2006-01-02 15:04:05", specified, task.NextRun.Location())
	if err != nil {
		t.Fatal(err)
	}
	if task.NextRun.Before(now) || !task.NextRun.Equal(wantNext) {
		t.Fatalf("initial next run = %v, want specified time %v", task.NextRun, wantNext)
	}
}

func TestCronJobWritePathsPersistAndClearExactTimes(t *testing.T) {
	store, db := newCronJobLogicTestStore(t)
	ctx := context.Background()
	serviceContext := &svc.ServiceContext{DB: db, CronJobStore: store}

	createReq := validCronJobRequest("exact-create-update")
	createReq.StartTime = "2099-07-01 00:00:00"
	createReq.EndTime = "2099-07-31 23:59:59"
	createReq.SpecifiedTimes = []string{"2099-07-01 08:00:00"}
	createReq.ExcludedTimes = []string{"2099-07-01 09:00:00"}
	created, err := NewCreateCronJobLogic(ctx, serviceContext).CreateCronJob(createReq)
	if err != nil {
		t.Fatal(err)
	}
	createdTask, err := store.GetByID(ctx, created.JobId)
	if err != nil {
		t.Fatal(err)
	}
	assertCronJobExactTimes(t, createdTask, createReq.SpecifiedTimes, createReq.ExcludedTimes, true, true)

	updateReq := updateCronJobRequestFromCreate(created.JobId, createReq)
	updateReq.SpecifiedTimes = nil
	updateReq.ExcludedTimes = nil
	if _, err := NewUpdateCronJobLogic(ctx, serviceContext).UpdateCronJob(updateReq); err != nil {
		t.Fatal(err)
	}
	updatedTask, err := store.GetByID(ctx, created.JobId)
	if err != nil {
		t.Fatal(err)
	}
	assertCronJobExactTimes(t, updatedTask, nil, nil, false, false)

	submitReq := validSubmitCronJobRequest("exact-submit")
	submitReq.StartTime = "2099-08-01 00:00:00"
	submitReq.EndTime = "2099-08-31 23:59:59"
	submitReq.SpecifiedTimes = []string{"2099-08-01 08:00:00"}
	submitReq.ExcludedTimes = []string{"2099-08-01 09:00:00"}
	submitted, err := NewSubmitCronJobLogic(ctx, serviceContext).SubmitCronJob(submitReq)
	if err != nil {
		t.Fatal(err)
	}
	submittedTask, err := store.GetByID(ctx, submitted.JobId)
	if err != nil {
		t.Fatal(err)
	}
	assertCronJobExactTimes(t, submittedTask, submitReq.SpecifiedTimes, submitReq.ExcludedTimes, true, true)

	submitReq.SpecifiedTimes = []string{"2099-08-02 07:30:00"}
	submitReq.ExcludedTimes = nil
	resubmitted, err := NewSubmitCronJobLogic(ctx, serviceContext).SubmitCronJob(submitReq)
	if err != nil {
		t.Fatal(err)
	}
	if resubmitted.JobId != submitted.JobId {
		t.Fatalf("submit update changed job id: got %q want %q", resubmitted.JobId, submitted.JobId)
	}
	resubmittedTask, err := store.GetByID(ctx, submitted.JobId)
	if err != nil {
		t.Fatal(err)
	}
	assertCronJobExactTimes(t, resubmittedTask, submitReq.SpecifiedTimes, nil, true, false)
}

func assertCronJobExactTimes(t *testing.T, task *crontask.TaskConfig, specified, excluded []string, wantRDATE, wantEXDATE bool) {
	t.Helper()
	extra, err := cronjob.ParseExtra(task.Extra)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(extra.SpecifiedTimes, specified) || !reflect.DeepEqual(extra.ExcludedTimes, excluded) {
		t.Fatalf("exact-time lists = specified %v excluded %v, want %v and %v", extra.SpecifiedTimes, extra.ExcludedTimes, specified, excluded)
	}
	if strings.Contains(task.RRuleStr, "RDATE") != wantRDATE || strings.Contains(task.RRuleStr, "EXDATE") != wantEXDATE {
		t.Fatalf("compiled set exact-time components mismatch: %q", task.RRuleStr)
	}
}

func TestUpdateCronJobRejectsInFlightTask(t *testing.T) {
	for _, status := range []crontask.TaskStatus{crontask.StatusEnabled, crontask.StatusDisabled} {
		t.Run(status.String(), func(t *testing.T) {
			store, db := newCronJobLogicTestStore(t)
			ctx := context.Background()
			serviceContext := &svc.ServiceContext{DB: db, CronJobStore: store}
			created, err := NewCreateCronJobLogic(ctx, serviceContext).CreateCronJob(validCronJobRequest("update-" + status.String()))
			if err != nil {
				t.Fatal(err)
			}
			scheduledTime := time.Now().Add(-time.Minute).Truncate(time.Second)
			leaseTime := time.Now().Add(time.Minute).Truncate(time.Second)
			lastRun := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
			lastScheduledRun := lastRun.Add(-time.Minute)
			if err := db.Model(&gormmodel.CronJob{}).Where("id = ?", created.JobId).Updates(map[string]any{
				"status":             int(status),
				"next_run":           leaseTime,
				"scheduled_time":     scheduledTime,
				"last_run":           lastRun,
				"last_scheduled_run": lastScheduledRun,
			}).Error; err != nil {
				t.Fatal(err)
			}

			config := validUpdateCronJobRequest(created.JobId)
			config.TaskName = "更新后的任务"
			if _, err := NewUpdateCronJobLogic(ctx, serviceContext).UpdateCronJob(config); !tool.IsErrorByPbCode(err, extproto.Code__1_02_DB) {
				t.Fatalf("in-flight update error = %v, want DB", err)
			}
			loaded, err := store.GetByID(ctx, created.JobId)
			if err != nil {
				t.Fatal(err)
			}
			if loaded.Status != status || loaded.TaskName == config.TaskName {
				t.Fatalf("in-flight config/state changed = %+v", loaded)
			}
			if !loaded.NextRun.Equal(leaseTime) || !loaded.ScheduledTime.Equal(scheduledTime) ||
				!loaded.LastRun.Equal(lastRun) || !loaded.LastScheduledRun.Equal(lastScheduledRun) {
				t.Fatalf("runtime fields changed: %+v", loaded)
			}
		})
	}
}

func TestUpdateCronJobRejectsMissing(t *testing.T) {
	store, db := newCronJobLogicTestStore(t)
	ctx := context.Background()
	serviceContext := &svc.ServiceContext{DB: db, CronJobStore: store}
	if _, err := NewUpdateCronJobLogic(ctx, serviceContext).UpdateCronJob(validUpdateCronJobRequest("missing")); !tool.IsErrorByPbCode(err, extproto.Code__1_02_RECORD_NOT_EXIST) {
		t.Fatalf("missing update error = %v, want RECORD_NOT_EXIST", err)
	}
}

func TestUpdateCronJobPreservesGroupAndReturnsIt(t *testing.T) {
	store, db := newCronJobLogicTestStore(t)
	ctx := context.Background()
	serviceContext := &svc.ServiceContext{DB: db, CronJobStore: store}
	createReq := validCronJobRequest("update-group")
	createReq.GroupId = "stable-group"
	created, err := NewCreateCronJobLogic(ctx, serviceContext).CreateCronJob(createReq)
	if err != nil {
		t.Fatal(err)
	}
	updateReq := updateCronJobRequestFromCreate(created.JobId, createReq)
	updateReq.TaskName = "更新后的任务"
	updated, err := NewUpdateCronJobLogic(ctx, serviceContext).UpdateCronJob(updateReq)
	if err != nil {
		t.Fatal(err)
	}
	if updated.GroupId != "stable-group" || updated.TaskCode != createReq.TaskCode {
		t.Fatalf("unexpected update response: %+v", updated)
	}
}

func TestUpdateCronJobAcceptsOmittedStableIdentity(t *testing.T) {
	store, db := newCronJobLogicTestStore(t)
	ctx := context.Background()
	serviceContext := &svc.ServiceContext{DB: db, CronJobStore: store}
	createReq := validCronJobRequest("update-omitted-identity")
	createReq.GroupId = "stable-group"
	created, err := NewCreateCronJobLogic(ctx, serviceContext).CreateCronJob(createReq)
	if err != nil {
		t.Fatal(err)
	}
	updateReq := updateCronJobRequestFromCreate(created.JobId, createReq)
	updated, err := NewUpdateCronJobLogic(ctx, serviceContext).UpdateCronJob(updateReq)
	if err != nil {
		t.Fatal(err)
	}
	if updated.GroupId != created.GroupId {
		t.Fatalf("update group id = %q, want %q", updated.GroupId, created.GroupId)
	}
	loaded, err := store.GetByID(ctx, created.JobId)
	if err != nil {
		t.Fatal(err)
	}
	extra, err := cronjob.ParseExtra(loaded.Extra)
	if err != nil {
		t.Fatal(err)
	}
	if extra.GroupId != created.GroupId || extra.DeptCode != createReq.DeptCode || extra.Type != createReq.Type {
		t.Fatalf("stable identity changed: %+v", extra)
	}
}

func TestSubmitCronJobCreatesUpdatesAndRejectsDeletedCode(t *testing.T) {
	store, db := newCronJobLogicTestStore(t)
	ctx := context.Background()
	serviceContext := &svc.ServiceContext{DB: db, CronJobStore: store}
	submit := NewSubmitCronJobLogic(ctx, serviceContext)
	created, err := submit.SubmitCronJob(validSubmitCronJobRequest("submit-active"))
	if err != nil {
		t.Fatal(err)
	}
	if created.GroupId == "" {
		t.Fatal("submit create must return generated groupId")
	}
	if repeated, err := submit.SubmitCronJob(validSubmitCronJobRequest("submit-active")); err != nil {
		t.Fatalf("identical submit must be idempotent: %v", err)
	} else if repeated.JobId != created.JobId {
		t.Fatalf("identical submit changed job id: got %q want %q", repeated.JobId, created.JobId)
	}
	if err := store.Disable(ctx, created.JobId); err != nil {
		t.Fatal(err)
	}
	updatedReq := validSubmitCronJobRequest("submit-active")
	updatedReq.TaskName = "提交更新后的任务"
	updated, err := submit.SubmitCronJob(updatedReq)
	if err != nil {
		t.Fatal(err)
	}
	if updated.JobId != created.JobId {
		t.Fatalf("submit changed job id: got %q want %q", updated.JobId, created.JobId)
	}
	if updated.TaskCode != updatedReq.TaskCode {
		t.Fatalf("submit task code = %q, want %q", updated.TaskCode, updatedReq.TaskCode)
	}
	loaded, err := store.GetByID(ctx, created.JobId)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != crontask.StatusDisabled || loaded.TaskName != updatedReq.TaskName {
		t.Fatalf("submit did not preserve state/update config: %+v", loaded)
	}

	deleted, err := submit.SubmitCronJob(validSubmitCronJobRequest("submit-deleted"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, deleted.JobId); err != nil {
		t.Fatal(err)
	}
	if _, err := submit.SubmitCronJob(validSubmitCronJobRequest("submit-deleted")); !tool.IsErrorByPbCode(err, extproto.Code__1_02_RECORD_ALREADY_EXIST) {
		t.Fatalf("deleted code submit error = %v, want RECORD_ALREADY_EXIST", err)
	}
}

func TestSubmitCronJobAcceptsOmittedIdentityOnlyForUpdate(t *testing.T) {
	store, db := newCronJobLogicTestStore(t)
	ctx := context.Background()
	serviceContext := &svc.ServiceContext{DB: db, CronJobStore: store}
	submit := NewSubmitCronJobLogic(ctx, serviceContext)
	createReq := validSubmitCronJobRequest("submit-omitted-identity")
	createReq.GroupId = "stable-group"
	created, err := submit.SubmitCronJob(createReq)
	if err != nil {
		t.Fatal(err)
	}
	updateReq := validSubmitCronJobRequest(createReq.TaskCode)
	updateReq.GroupId = ""
	updateReq.DeptCode = ""
	updateReq.Type = ""
	updated, err := submit.SubmitCronJob(updateReq)
	if err != nil {
		t.Fatal(err)
	}
	if updated.GroupId != created.GroupId || updated.JobId != created.JobId {
		t.Fatalf("unexpected submit update response: %+v", updated)
	}
	newReq := validSubmitCronJobRequest("submit-missing-create-identity")
	newReq.DeptCode = ""
	newReq.Type = ""
	if _, err := submit.SubmitCronJob(newReq); !tool.IsErrorByPbCode(err, extproto.Code__1_01_PARAM_INVALID) {
		t.Fatalf("submit create identity error = %v, want PARAM_INVALID", err)
	}
}

func TestSubmitCronJobRejectsInFlightTask(t *testing.T) {
	store, db := newCronJobLogicTestStore(t)
	ctx := context.Background()
	serviceContext := &svc.ServiceContext{DB: db, CronJobStore: store}
	submit := NewSubmitCronJobLogic(ctx, serviceContext)
	created, err := submit.SubmitCronJob(validSubmitCronJobRequest("submit-in-flight"))
	if err != nil {
		t.Fatal(err)
	}
	scheduledTime := time.Now().Add(-time.Minute).Truncate(time.Second)
	leaseTime := time.Now().Add(time.Minute).Truncate(time.Second)
	if err := db.Model(&gormmodel.CronJob{}).Where("id = ?", created.JobId).Updates(map[string]any{
		"next_run":       leaseTime,
		"scheduled_time": scheduledTime,
	}).Error; err != nil {
		t.Fatal(err)
	}
	req := validSubmitCronJobRequest("submit-in-flight")
	req.TaskName = "must-not-change"
	if _, err := submit.SubmitCronJob(req); !tool.IsErrorByPbCode(err, extproto.Code__1_02_DB) {
		t.Fatalf("in-flight submit error = %v, want DB", err)
	}
	loaded, err := store.GetByID(ctx, created.JobId)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TaskName == req.TaskName || !loaded.NextRun.Equal(leaseTime) || !loaded.ScheduledTime.Equal(scheduledTime) {
		t.Fatalf("in-flight submit changed configuration or lease: %+v", loaded)
	}
}

func validCronJobRequest(taskCode string) *trigger.CreateCronJobReq {
	return &trigger.CreateCronJobReq{
		TaskCode: taskCode,
		TaskName: "测试周期任务",
		Type:     "test",
		DeptCode: "D001",
		Rule: &trigger.PlanRulePb{
			Freq:    3,
			Hours:   []int32{int32(time.Now().Add(time.Hour).Hour())},
			Minutes: []int32{0},
		},
	}
}

func validUpdateCronJobRequest(jobID string) *trigger.UpdateCronJobReq {
	return updateCronJobRequestFromCreate(jobID, validCronJobRequest("unused"))
}

func validSubmitCronJobRequest(taskCode string) *trigger.SubmitCronJobReq {
	return submitCronJobRequestFromCreate(validCronJobRequest(taskCode))
}

func updateCronJobRequestFromCreate(jobID string, req *trigger.CreateCronJobReq) *trigger.UpdateCronJobReq {
	return &trigger.UpdateCronJobReq{
		JobId: jobID, TaskName: req.TaskName, Description: req.Description,
		StartTime: req.StartTime, EndTime: req.EndTime, Rule: req.Rule,
		ExcludeDates: append([]string(nil), req.ExcludeDates...), Priority: req.Priority,
		SpecifiedTimes: append([]string(nil), req.SpecifiedTimes...), ExcludedTimes: append([]string(nil), req.ExcludedTimes...),
		Payload: req.Payload, LockTimeout: req.LockTimeout, MaxDelay: req.MaxDelay,
		SkipTimeFilter: req.SkipTimeFilter, Ext1: req.Ext1, Ext2: req.Ext2, Ext3: req.Ext3,
		Ext4: req.Ext4, Ext5: req.Ext5,
	}
}

func submitCronJobRequestFromCreate(req *trigger.CreateCronJobReq) *trigger.SubmitCronJobReq {
	return &trigger.SubmitCronJobReq{
		TaskCode: req.TaskCode, TaskName: req.TaskName, Type: req.Type, GroupId: req.GroupId,
		Description: req.Description, StartTime: req.StartTime, EndTime: req.EndTime, Rule: req.Rule,
		ExcludeDates: append([]string(nil), req.ExcludeDates...), Priority: req.Priority, Payload: req.Payload,
		SpecifiedTimes: append([]string(nil), req.SpecifiedTimes...), ExcludedTimes: append([]string(nil), req.ExcludedTimes...),
		LockTimeout: req.LockTimeout, MaxDelay: req.MaxDelay, SkipTimeFilter: req.SkipTimeFilter,
		Ext1: req.Ext1, Ext2: req.Ext2, Ext3: req.Ext3, Ext4: req.Ext4, Ext5: req.Ext5, DeptCode: req.DeptCode,
	}
}

func TestCronJobRunGetAndList(t *testing.T) {
	store, db := newCronJobLogicTestStore(t)
	ctx := context.Background()
	nextRun := time.Now().Add(time.Hour).Truncate(time.Second)
	ruleJSON, err := json.Marshal(&trigger.PlanRulePb{Freq: 3, Hours: []int32{11}, Minutes: []int32{0}})
	if err != nil {
		t.Fatal(err)
	}
	extra, err := cronjob.MarshalExtra(&cronjob.CronJobExtra{
		DeptCode:       "D001",
		Type:           "inspection",
		GroupId:        "G001",
		Description:    "立即执行测试",
		Rule:           ruleJSON,
		ExcludeDates:   []string{"2026-07-28"},
		SpecifiedTimes: []string{"2026-07-29 12:34:56"},
		ExcludedTimes:  []string{"2026-07-30 12:34:56"},
	})
	if err != nil {
		t.Fatal(err)
	}
	config := &crontask.TaskConfig{
		TaskCode:    "management-logic",
		TaskName:    "管理接口测试",
		RRuleStr:    "DTSTART:20260701T000000Z\nRRULE:FREQ=DAILY",
		Priority:    5,
		LockTimeout: 90 * time.Second,
		Payload:     json.RawMessage(`{"id":1}`),
		Extra:       extra,
		Status:      crontask.StatusDisabled,
		NextRun:     nextRun,
	}
	if err := store.Insert(ctx, config); err != nil {
		t.Fatal(err)
	}
	baseTime := time.Date(2026, 7, 27, 10, 0, 0, 0, time.Local)
	if err := db.Model(&gormmodel.CronJob{}).
		Where("id = ?", config.ID).
		Updates(map[string]any{"create_time": baseTime, "update_time": baseTime}).Error; err != nil {
		t.Fatal(err)
	}
	laterConfig := *config
	laterConfig.ID = ""
	laterConfig.TaskCode = "management-logic-later"
	laterConfig.TaskName = "管理接口测试二"
	if err := store.Insert(ctx, &laterConfig); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&gormmodel.CronJob{}).
		Where("id = ?", laterConfig.ID).
		Updates(map[string]any{"create_time": baseTime.Add(time.Hour), "update_time": baseTime.Add(time.Hour)}).Error; err != nil {
		t.Fatal(err)
	}
	deletedConfig := laterConfig
	deletedConfig.ID = ""
	deletedConfig.TaskCode = "management-logic-deleted"
	if err := store.Insert(ctx, &deletedConfig); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, deletedConfig.ID); err != nil {
		t.Fatal(err)
	}

	executed := make(chan *crontask.TaskConfig, 1)
	lastRunUpdated := make(chan error, 1)
	schedulerStore := &notifyingTaskStore{TaskStore: store, lastRunUpdated: lastRunUpdated}
	scheduler := crontask.NewScheduler(schedulerStore, func(_ context.Context, task *crontask.TaskConfig) error {
		executed <- task
		return nil
	})
	serviceContext := &svc.ServiceContext{
		DB:               db,
		CronJobStore:     store,
		CronJobScheduler: scheduler,
	}

	if _, err := NewRunCronJobLogic(ctx, serviceContext).RunCronJob(&trigger.RunCronJobReq{JobId: config.ID}); err != nil {
		t.Fatal(err)
	}
	select {
	case task := <-executed:
		if task.NextRun.IsZero() {
			t.Fatal("manual execution time must not be zero")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for manual execution")
	}
	select {
	case err := <-lastRunUpdated:
		if err != nil {
			t.Fatalf("update manual last run: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for last run update")
	}

	loaded, err := store.GetByID(ctx, config.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.LastRun.IsZero() {
		t.Fatal("manual execution did not update last run")
	}
	if loaded.Status != crontask.StatusDisabled || !loaded.NextRun.Equal(nextRun) {
		t.Fatalf("manual execution changed periodic plan: %+v", loaded)
	}

	detail, err := NewGetCronJobLogic(ctx, serviceContext).GetCronJob(&trigger.GetCronJobReq{JobId: config.ID})
	if err != nil {
		t.Fatal(err)
	}
	if detail.CronJob == nil || detail.CronJob.TaskCode != config.TaskCode {
		t.Fatalf("unexpected Cron Job detail: %+v", detail.CronJob)
	}
	if !reflect.DeepEqual(detail.CronJob.SpecifiedTimes, []string{"2026-07-29 12:34:56"}) ||
		!reflect.DeepEqual(detail.CronJob.ExcludedTimes, []string{"2026-07-30 12:34:56"}) {
		t.Fatalf("detail exact times did not round trip: %+v", detail.CronJob)
	}
	wantAuditTime := carbonx.FormatDateTime(baseTime)
	if detail.CronJob.CreateTime != wantAuditTime || detail.CronJob.UpdateTime == "" {
		t.Fatalf("unexpected detail audit times: %+v", detail.CronJob)
	}
	list, err := NewListCronJobsLogic(ctx, serviceContext).ListCronJobs(&trigger.ListCronJobsReq{
		PageNum:  1,
		PageSize: 1,
		TaskCode: "management",
		TaskName: "管理",
		Status:   []int32{int32(crontask.StatusDisabled)},
		DeptCode: "D001",
		Type:     "inspection",
		GroupId:  "G001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if list.Total != 2 || len(list.CronJobs) != 1 || list.CronJobs[0].JobId != laterConfig.ID {
		t.Fatalf("unexpected Cron Job list: %+v", list)
	}
	if list.CronJobs[0].CreateTime != carbonx.FormatDateTime(baseTime.Add(time.Hour)) {
		t.Fatalf("unexpected list audit time: %+v", list.CronJobs[0])
	}
	if !reflect.DeepEqual(list.CronJobs[0].SpecifiedTimes, []string{"2026-07-29 12:34:56"}) ||
		!reflect.DeepEqual(list.CronJobs[0].ExcludedTimes, []string{"2026-07-30 12:34:56"}) {
		t.Fatalf("list exact times did not round trip: %+v", list.CronJobs[0])
	}

	if _, err := NewGetCronJobLogic(ctx, serviceContext).GetCronJob(&trigger.GetCronJobReq{JobId: "missing"}); !tool.IsErrorByPbCode(err, extproto.Code__1_02_RECORD_NOT_EXIST) {
		t.Fatalf("get missing error = %v, want RECORD_NOT_EXIST", err)
	}
	if _, err := NewRunCronJobLogic(ctx, serviceContext).RunCronJob(&trigger.RunCronJobReq{JobId: "missing"}); !tool.IsErrorByPbCode(err, extproto.Code__1_02_RECORD_NOT_EXIST) {
		t.Fatalf("run missing error = %v, want RECORD_NOT_EXIST", err)
	}
}

func TestPreviewCronJobSchedule(t *testing.T) {
	store, db := newCronJobLogicTestStore(t)
	ctx := context.Background()
	after := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Hour)
	excluded := after.Add(24 * time.Hour)
	specified := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	exactExcluded := specified.Add(time.Hour)
	ruleJSON, err := json.Marshal(&trigger.PlanRulePb{})
	if err != nil {
		t.Fatal(err)
	}
	extra, err := cronjob.MarshalExtra(&cronjob.CronJobExtra{
		Rule:           ruleJSON,
		SpecifiedTimes: []string{carbonx.FormatDateTime(specified)},
		ExcludedTimes:  []string{carbonx.FormatDateTime(exactExcluded)},
	})
	if err != nil {
		t.Fatal(err)
	}
	task := &crontask.TaskConfig{
		TaskCode: "preview-disabled",
		TaskName: "预览禁用任务",
		Status:   crontask.StatusDisabled,
		Extra:    extra,
		RRuleStr: "DTSTART:" + after.Format("20060102T150405Z") + "\n" +
			"RRULE:FREQ=DAILY;COUNT=12\n" +
			"RDATE:" + specified.Format("20060102T150405Z") + "," + exactExcluded.Format("20060102T150405Z") + "\n" +
			"EXDATE:" + excluded.Format("20060102T150405Z") + "," + exactExcluded.Format("20060102T150405Z"),
	}
	if err := store.Insert(ctx, task); err != nil {
		t.Fatal(err)
	}
	serviceContext := &svc.ServiceContext{
		DB:               db,
		CronJobStore:     store,
		CronJobScheduler: crontask.NewScheduler(store, nil),
	}
	logic := NewPreviewCronJobScheduleLogic(ctx, serviceContext)

	preview, err := logic.PreviewCronJobSchedule(&trigger.PreviewCronJobScheduleReq{JobId: task.ID})
	if err != nil {
		t.Fatal(err)
	}
	if preview.JobId != task.ID || preview.TaskCode != task.TaskCode || preview.RruleStr != task.RRuleStr {
		t.Fatalf("unexpected preview identity/rule: %+v", preview)
	}
	if len(preview.ExecutionTimes) != 10 {
		t.Fatalf("default execution time count = %d, want 10", len(preview.ExecutionTimes))
	}
	if preview.ExecutionTimes[0] != carbonx.FormatDateTime(specified) {
		t.Fatalf("first preview time = %q, want persisted RDATE %v", preview.ExecutionTimes[0], specified)
	}
	if preview.ScheduleDescription == "" {
		t.Fatal("schedule description must not be empty")
	}
	excludedText := carbonx.FormatDateTime(excluded)
	exactExcludedText := carbonx.FormatDateTime(exactExcluded)
	for _, executionTime := range preview.ExecutionTimes {
		if executionTime == excludedText || executionTime == exactExcludedText {
			t.Fatalf("EXDATE time %q was included", executionTime)
		}
	}
	loaded, err := store.GetByID(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != crontask.StatusDisabled || !loaded.NextRun.IsZero() || !loaded.LastRun.IsZero() {
		t.Fatalf("preview changed disabled task state: %+v", loaded)
	}

	limited, err := logic.PreviewCronJobSchedule(&trigger.PreviewCronJobScheduleReq{JobId: task.ID, Count: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(limited.ExecutionTimes) != 2 {
		t.Fatalf("explicit execution time count = %d, want 2", len(limited.ExecutionTimes))
	}
	if _, err := logic.PreviewCronJobSchedule(&trigger.PreviewCronJobScheduleReq{JobId: task.ID, Count: 101}); err == nil {
		t.Fatal("count above 100 must fail validation")
	}
}

func TestPreviewCronJobScheduleErrorsAndExhaustion(t *testing.T) {
	store, db := newCronJobLogicTestStore(t)
	ctx := context.Background()
	serviceContext := &svc.ServiceContext{
		DB:               db,
		CronJobStore:     store,
		CronJobScheduler: crontask.NewScheduler(store, nil),
	}
	logic := NewPreviewCronJobScheduleLogic(ctx, serviceContext)

	if _, err := logic.PreviewCronJobSchedule(&trigger.PreviewCronJobScheduleReq{JobId: "missing"}); !tool.IsErrorByPbCode(err, extproto.Code__1_02_RECORD_NOT_EXIST) {
		t.Fatalf("missing preview error = %v, want RECORD_NOT_EXIST", err)
	}

	exhausted := &crontask.TaskConfig{
		TaskCode: "preview-exhausted",
		TaskName: "已耗尽预览任务",
		Status:   crontask.StatusEnabled,
		Extra:    json.RawMessage(`{"rule":{}}`),
		RRuleStr: "DTSTART:20200101T000000Z\nRRULE:FREQ=DAILY;COUNT=1",
	}
	if err := store.Insert(ctx, exhausted); err != nil {
		t.Fatal(err)
	}
	preview, err := logic.PreviewCronJobSchedule(&trigger.PreviewCronJobScheduleReq{JobId: exhausted.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.ExecutionTimes) != 0 {
		t.Fatalf("exhausted execution times = %v, want empty", preview.ExecutionTimes)
	}

	malformed := &crontask.TaskConfig{
		TaskCode: "preview-malformed",
		TaskName: "非法规则预览任务",
		Status:   crontask.StatusEnabled,
		Extra:    json.RawMessage(`{"rule":{}}`),
		RRuleStr: "DTSTART:20300101T000000Z\nRRULE:FREQ=DAILY;COUNT=1",
	}
	if err := store.Insert(ctx, malformed); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&gormmodel.CronJob{}).Where("id = ?", malformed.ID).Update("rrule_str", "FREQ=DAILY").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := logic.PreviewCronJobSchedule(&trigger.PreviewCronJobScheduleReq{JobId: malformed.ID}); !tool.IsErrorByPbCode(err, extproto.Code__1_01_PARAM_INVALID) {
		t.Fatalf("malformed preview error = %v, want PARAM_INVALID", err)
	}

	emptyRule := &crontask.TaskConfig{
		TaskCode: "preview-empty-rule",
		TaskName: "空规则预览任务",
		Status:   crontask.StatusEnabled,
		Extra:    json.RawMessage(`{"rule":{}}`),
		RRuleStr: "DTSTART:20300101T000000Z\nRRULE:FREQ=DAILY;COUNT=1",
	}
	if err := store.Insert(ctx, emptyRule); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&gormmodel.CronJob{}).Where("id = ?", emptyRule.ID).Update("rrule_str", "").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := logic.PreviewCronJobSchedule(&trigger.PreviewCronJobScheduleReq{JobId: emptyRule.ID}); !tool.IsErrorByPbCode(err, extproto.Code__1_01_PARAM_INVALID) {
		t.Fatalf("empty rule preview error = %v, want PARAM_INVALID", err)
	}

	if _, err := logic.PreviewCronJobSchedule(nil); err == nil {
		t.Fatal("nil request must return an error")
	}
	missingScheduler := NewPreviewCronJobScheduleLogic(ctx, &svc.ServiceContext{CronJobStore: store})
	if _, err := missingScheduler.PreviewCronJobSchedule(&trigger.PreviewCronJobScheduleReq{JobId: exhausted.ID}); !tool.IsErrorByPbCode(err, extproto.Code__1_02_DB) {
		t.Fatalf("missing scheduler error = %v, want DB", err)
	}
}

type notifyingTaskStore struct {
	crontask.TaskStore
	lastRunUpdated chan<- error
}

func (s *notifyingTaskStore) UpdateLastRun(ctx context.Context, id string, lastRun time.Time) error {
	err := s.TaskStore.UpdateLastRun(ctx, id, lastRun)
	s.lastRunUpdated <- err
	return err
}

func newCronJobLogicTestStore(t *testing.T) (*cronjob.DBStore, *gormx.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&parseTime=true&_loc=auto"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&gormmodel.CronJob{}); err != nil {
		t.Fatal(err)
	}
	wrappedDB := &gormx.DB{DB: db}
	return cronjob.NewDBStore(wrappedDB), wrappedDB
}
