package logic

import (
	"context"
	"strings"
	"testing"
	"time"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/authctx"
	"zero-service/common/gormx"
	"zero-service/common/tool"
	"zero-service/model"

	"github.com/alicebob/miniredis/v2"
	"github.com/zeromicro/go-zero/core/mathx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openConfirmTestDB(t *testing.T) *gorm.DB {
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
	if err := db.AutoMigrate(&gormmodel.Plan{}, &gormmodel.PlanBatch{}, &gormmodel.PlanExecItem{}, &gormmodel.PlanConfirm{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func newConfirmTestServiceContext(t *testing.T, db *gorm.DB) *svc.ServiceContext {
	t.Helper()
	redisServer := miniredis.RunT(t)
	redisClient := redis.New(redisServer.Addr())
	return &svc.ServiceContext{
		DB:             &gormx.DB{DB: db},
		Redis:          redisClient,
		IdUtil:         tool.NewIdUtil(redisClient),
		UnstableExpiry: mathx.NewUnstable(0),
	}
}

func createConfirmPlanRequest(confirmType int32, leadMinutes int32) *trigger.CreatePlanTaskReq {
	return &trigger.CreatePlanTaskReq{
		DeptCode:           "dept",
		PlanId:             "confirm-plan",
		PlanName:           "confirm plan",
		Type:               "test",
		StartTime:          "2099-07-01 00:00:00",
		EndTime:            "2099-07-01 23:59:59",
		Rule:               &trigger.PlanRulePb{Freq: 3, Hours: []int32{9}, Minutes: []int32{0}},
		ExecItems:          []*trigger.CreatePlanExecItemPb{{ItemId: "item"}},
		SkipTimeFilter:     true,
		ConfirmType:        confirmType,
		ConfirmLeadMinutes: leadMinutes,
	}
}

func TestCreatePlanTaskWritesBatchConfirm(t *testing.T) {
	db := openConfirmTestDB(t)
	svcCtx := newConfirmTestServiceContext(t, db)
	request := createConfirmPlanRequest(int32(0), 0)
	request.ConfirmType = int32(1) // model.ConfirmTypeBatch
	request.ConfirmLeadMinutes = 5
	request.ConfirmTimeoutMinutes = 540 // 09:00 触发，18:00 截止

	if _, err := NewCreatePlanTaskLogic(context.Background(), svcCtx).CreatePlanTask(request); err != nil {
		t.Fatal(err)
	}

	var batch gormmodel.PlanBatch
	if err := db.Where("plan_id = ?", request.PlanId).First(&batch).Error; err != nil {
		t.Fatal(err)
	}
	var confirm gormmodel.PlanConfirm
	if err := db.Where("target_pk = ?", batch.Id).First(&confirm).Error; err != nil {
		t.Fatal(err)
	}
	if confirm.ConfirmType != 1 || confirm.LeadMinutes != 5 {
		t.Fatalf("confirm type/lead = %d/%d, want 1/5", confirm.ConfirmType, confirm.LeadMinutes)
	}
	if !confirm.NotifyAt.Before(batch.PlanTriggerTime.Time) {
		t.Fatalf("notify_at = %s, want before trigger %s", confirm.NotifyAt, batch.PlanTriggerTime.Time)
	}
	if confirm.Deadline.Day() != batch.PlanTriggerTime.Time.Day() {
		t.Fatalf("deadline = %s, want same day as trigger %s", confirm.Deadline, batch.PlanTriggerTime.Time)
	}
	if confirm.Deadline.Hour() != 18 || confirm.Deadline.Minute() != 0 {
		t.Fatalf("deadline = %s, want 18:00:00", confirm.Deadline)
	}
	if confirm.ConfirmStatus != 0 {
		t.Fatalf("confirm status = %d, want waiting", confirm.ConfirmStatus)
	}
}

func TestCreatePlanTaskRejectsExecItemConfirm(t *testing.T) {
	db := openConfirmTestDB(t)
	svcCtx := newConfirmTestServiceContext(t, db)
	request := createConfirmPlanRequest(int32(2), 0) // model.ConfirmTypeExecItem
	if _, err := NewCreatePlanTaskLogic(context.Background(), svcCtx).CreatePlanTask(request); err == nil {
		t.Fatal("CreatePlanTask() error = nil, want exec item confirm rejected")
	}
}

func TestConfirmPlanBatch(t *testing.T) {
	db := openConfirmTestDB(t)
	svcCtx := newConfirmTestServiceContext(t, db)
	request := createConfirmPlanRequest(int32(1), 1)
	if _, err := NewCreatePlanTaskLogic(context.Background(), svcCtx).CreatePlanTask(request); err != nil {
		t.Fatal(err)
	}
	var batch gormmodel.PlanBatch
	if err := db.Where("plan_id = ?", request.PlanId).First(&batch).Error; err != nil {
		t.Fatal(err)
	}

	logic := NewConfirmPlanBatchLogic(context.Background(), svcCtx)
	if _, err := logic.ConfirmPlanBatch(&trigger.ConfirmPlanBatchReq{Id: batch.Id}); err != nil {
		t.Fatal(err)
	}
	var confirm gormmodel.PlanConfirm
	if err := db.Where("target_pk = ?", batch.Id).First(&confirm).Error; err != nil {
		t.Fatal(err)
	}
	if !confirm.ConfirmedTime.Valid {
		t.Fatal("confirmed_time is not set")
	}
	if confirm.ConfirmStatus != 1 {
		t.Fatalf("confirm status = %d, want confirmed", confirm.ConfirmStatus)
	}

	// 重复确认按幂等成功处理。
	if _, err := logic.ConfirmPlanBatch(&trigger.ConfirmPlanBatchReq{Id: batch.Id}); err != nil {
		t.Fatalf("second confirm error = %v, want nil", err)
	}
}

func TestConfirmPlanBatchRejectsExpired(t *testing.T) {
	db := openConfirmTestDB(t)
	svcCtx := newConfirmTestServiceContext(t, db)
	request := createConfirmPlanRequest(int32(1), 1)
	if _, err := NewCreatePlanTaskLogic(context.Background(), svcCtx).CreatePlanTask(request); err != nil {
		t.Fatal(err)
	}
	var batch gormmodel.PlanBatch
	if err := db.Where("plan_id = ?", request.PlanId).First(&batch).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&gormmodel.PlanConfirm{}).Where("target_pk = ?", batch.Id).
		Update("deadline", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := NewConfirmPlanBatchLogic(context.Background(), svcCtx).
		ConfirmPlanBatch(&trigger.ConfirmPlanBatchReq{Id: batch.Id}); err == nil {
		t.Fatal("ConfirmPlanBatch() error = nil, want expired rejected")
	}
}

func TestRejectPlanBatch(t *testing.T) {
	db := openConfirmTestDB(t)
	svcCtx := newConfirmTestServiceContext(t, db)
	request := createConfirmPlanRequest(int32(1), 1)
	if _, err := NewCreatePlanTaskLogic(context.Background(), svcCtx).CreatePlanTask(request); err != nil {
		t.Fatal(err)
	}
	var batch gormmodel.PlanBatch
	if err := db.Where("plan_id = ?", request.PlanId).First(&batch).Error; err != nil {
		t.Fatal(err)
	}

	logic := NewRejectPlanBatchLogic(authctx.WithUserID(context.Background(), "tester"), svcCtx)
	if _, err := logic.RejectPlanBatch(&trigger.RejectPlanBatchReq{Id: batch.Id, Reason: "条件不满足"}); err != nil {
		t.Fatal(err)
	}

	var confirm gormmodel.PlanConfirm
	if err := db.Where("target_pk = ?", batch.Id).First(&confirm).Error; err != nil {
		t.Fatal(err)
	}
	if confirm.ConfirmStatus != model.PlanConfirmStatusConfirmed {
		t.Fatalf("confirm status = %d, want confirmed", confirm.ConfirmStatus)
	}
	if !confirm.ConfirmedTime.Valid || confirm.ConfirmedUser.String != "tester" {
		t.Fatalf("confirmed_time/user = %v/%q, want set/tester", confirm.ConfirmedTime, confirm.ConfirmedUser.String)
	}
	if !strings.Contains(confirm.Remark.String, "条件不满足") {
		t.Fatalf("remark = %q, want contains 条件不满足", confirm.Remark.String)
	}

	var gotBatch gormmodel.PlanBatch
	if err := db.Where("id = ?", batch.Id).First(&gotBatch).Error; err != nil {
		t.Fatal(err)
	}
	if gotBatch.Status != model.PlanStatusTerminated {
		t.Fatalf("batch status = %d, want terminated", gotBatch.Status)
	}
	var item gormmodel.PlanExecItem
	if err := db.Where("batch_pk = ?", batch.Id).First(&item).Error; err != nil {
		t.Fatal(err)
	}
	if item.Status != model.StatusWaiting {
		t.Fatalf("item status = %d, want waiting (批次终止后不再下发)", item.Status)
	}

	// 重复拒绝按幂等成功处理。
	if _, err := logic.RejectPlanBatch(&trigger.RejectPlanBatchReq{Id: batch.Id}); err != nil {
		t.Fatalf("second reject error = %v, want nil", err)
	}
	// 拒绝后不可再确认。
	if _, err := NewConfirmPlanBatchLogic(context.Background(), svcCtx).
		ConfirmPlanBatch(&trigger.ConfirmPlanBatchReq{Id: batch.Id}); err == nil {
		t.Fatal("ConfirmPlanBatch() after reject error = nil, want rejected")
	}
}

func TestCreatePlanTaskRejectsZeroLeadMinutes(t *testing.T) {
	db := openConfirmTestDB(t)
	svcCtx := newConfirmTestServiceContext(t, db)
	request := createConfirmPlanRequest(int32(1), 0)
	if _, err := NewCreatePlanTaskLogic(context.Background(), svcCtx).CreatePlanTask(request); err == nil {
		t.Fatal("CreatePlanTask() error = nil, want zero lead rejected")
	}
}
