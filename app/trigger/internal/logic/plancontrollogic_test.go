package logic

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/authctx"
	"zero-service/common/errx"
	"zero-service/model"
	"zero-service/third_party/extproto"

	"gorm.io/gorm"
)

func newPlanControlTestRig(t *testing.T, planStatus, batchStatus int) (*gorm.DB, *svc.ServiceContext, gormmodel.Plan, gormmodel.PlanBatch) {
	t.Helper()
	db := openConfirmTestDB(t)
	svcCtx := newConfirmTestServiceContext(t, db)

	plan := gormmodel.Plan{PlanId: "plan", Status: planStatus}
	plan.Id = "plan-pk"
	if planStatus == model.PlanStatusPaused {
		plan.PausedTime = sql.NullTime{Time: time.Now().Add(-time.Minute), Valid: true}
		plan.PausedReason = sql.NullString{String: "历史暂停", Valid: true}
	}
	batch := gormmodel.PlanBatch{PlanPk: plan.Id, PlanId: plan.PlanId, BatchId: "batch", Status: batchStatus}
	batch.Id = "batch-pk"
	if batchStatus == model.PlanStatusPaused {
		batch.PausedTime = sql.NullTime{Time: time.Now().Add(-time.Minute), Valid: true}
		batch.PausedReason = sql.NullString{String: "历史暂停", Valid: true}
	}
	for _, value := range []any{&plan, &batch} {
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db, svcCtx, plan, batch
}

func TestPausePlanUpdatesPlanAndEnabledBatches(t *testing.T) {
	db, svcCtx, plan, batch := newPlanControlTestRig(t, model.PlanStatusEnabled, model.PlanStatusEnabled)
	ctx := authctx.WithUserID(context.Background(), "tester")

	if _, err := NewPausePlanLogic(ctx, svcCtx).PausePlan(&trigger.PausePlanReq{
		Id:     plan.Id,
		Reason: "人工暂停",
	}); err != nil {
		t.Fatal(err)
	}

	var gotPlan gormmodel.Plan
	if err := db.Where("id = ?", plan.Id).First(&gotPlan).Error; err != nil {
		t.Fatal(err)
	}
	if gotPlan.Status != model.PlanStatusPaused || !gotPlan.PausedTime.Valid || gotPlan.PausedReason.String != "人工暂停" {
		t.Fatalf("plan status/paused = %d/%v/%q, want paused/true/人工暂停",
			gotPlan.Status, gotPlan.PausedTime.Valid, gotPlan.PausedReason.String)
	}
	var gotBatch gormmodel.PlanBatch
	if err := db.Where("id = ?", batch.Id).First(&gotBatch).Error; err != nil {
		t.Fatal(err)
	}
	if gotBatch.Status != model.PlanStatusPaused || gotBatch.PausedReason.String != "人工暂停" {
		t.Fatalf("batch status/reason = %d/%q, want paused/人工暂停", gotBatch.Status, gotBatch.PausedReason.String)
	}
}

func TestResumePlanClearsPausedFieldsAndReenablesBatches(t *testing.T) {
	db, svcCtx, plan, batch := newPlanControlTestRig(t, model.PlanStatusPaused, model.PlanStatusPaused)
	ctx := authctx.WithUserID(context.Background(), "tester")

	if _, err := NewResumePlanLogic(ctx, svcCtx).ResumePlan(&trigger.ResumePlanReq{Id: plan.Id}); err != nil {
		t.Fatal(err)
	}

	var gotPlan gormmodel.Plan
	if err := db.Where("id = ?", plan.Id).First(&gotPlan).Error; err != nil {
		t.Fatal(err)
	}
	if gotPlan.Status != model.PlanStatusEnabled || gotPlan.PausedTime.Valid || gotPlan.PausedReason.Valid {
		t.Fatalf("plan status/paused = %d/%v/%q, want enabled/cleared",
			gotPlan.Status, gotPlan.PausedTime.Valid, gotPlan.PausedReason.String)
	}
	var gotBatch gormmodel.PlanBatch
	if err := db.Where("id = ?", batch.Id).First(&gotBatch).Error; err != nil {
		t.Fatal(err)
	}
	if gotBatch.Status != model.PlanStatusEnabled || gotBatch.PausedTime.Valid || gotBatch.PausedReason.Valid {
		t.Fatalf("batch status/paused = %d/%v/%q, want enabled/cleared",
			gotBatch.Status, gotBatch.PausedTime.Valid, gotBatch.PausedReason.String)
	}
}

func TestPausePlanBatchUpdatesBatchOnly(t *testing.T) {
	db, svcCtx, plan, batch := newPlanControlTestRig(t, model.PlanStatusEnabled, model.PlanStatusEnabled)
	ctx := authctx.WithUserID(context.Background(), "tester")

	if _, err := NewPausePlanBatchLogic(ctx, svcCtx).PausePlanBatch(&trigger.PausePlanBatchReq{
		Id:     batch.Id,
		Reason: "人工暂停",
	}); err != nil {
		t.Fatal(err)
	}

	var gotBatch gormmodel.PlanBatch
	if err := db.Where("id = ?", batch.Id).First(&gotBatch).Error; err != nil {
		t.Fatal(err)
	}
	if gotBatch.Status != model.PlanStatusPaused || gotBatch.PausedReason.String != "人工暂停" {
		t.Fatalf("batch status/reason = %d/%q, want paused/人工暂停", gotBatch.Status, gotBatch.PausedReason.String)
	}
	// 批次级暂停不改动计划本身。
	var gotPlan gormmodel.Plan
	if err := db.Where("id = ?", plan.Id).First(&gotPlan).Error; err != nil {
		t.Fatal(err)
	}
	if gotPlan.Status != model.PlanStatusEnabled {
		t.Fatalf("plan status = %d, want enabled", gotPlan.Status)
	}
}

func TestResumePlanBatchUpdatesBatchAndPausedPlan(t *testing.T) {
	db, svcCtx, plan, batch := newPlanControlTestRig(t, model.PlanStatusPaused, model.PlanStatusPaused)
	ctx := authctx.WithUserID(context.Background(), "tester")

	if _, err := NewResumePlanBatchLogic(ctx, svcCtx).ResumePlanBatch(&trigger.ResumePlanBatchReq{
		Id: batch.Id,
	}); err != nil {
		t.Fatal(err)
	}

	var gotBatch gormmodel.PlanBatch
	if err := db.Where("id = ?", batch.Id).First(&gotBatch).Error; err != nil {
		t.Fatal(err)
	}
	if gotBatch.Status != model.PlanStatusEnabled {
		t.Fatalf("batch status = %d, want enabled", gotBatch.Status)
	}
	var gotPlan gormmodel.Plan
	if err := db.Where("id = ?", plan.Id).First(&gotPlan).Error; err != nil {
		t.Fatal(err)
	}
	if gotPlan.Status != model.PlanStatusEnabled || gotPlan.PausedTime.Valid || gotPlan.PausedReason.Valid {
		t.Fatalf("plan status/paused = %d/%v/%q, want enabled/cleared",
			gotPlan.Status, gotPlan.PausedTime.Valid, gotPlan.PausedReason.String)
	}
}

func TestPausePlanRejectsConcurrentTermination(t *testing.T) {
	db, svcCtx, plan, _ := newPlanControlTestRig(t, model.PlanStatusEnabled, model.PlanStatusEnabled)
	injectStatusChangeAfterLoad(t, db, "plan", plan.Id, map[string]any{
		"status":            model.PlanStatusTerminated,
		"finished_time":     time.Now(),
		"terminated_reason": "并发终止",
	}, &gormmodel.Plan{})
	ctx := authctx.WithUserID(context.Background(), "tester")

	_, err := NewPausePlanLogic(ctx, svcCtx).PausePlan(&trigger.PausePlanReq{Id: plan.Id, Reason: "人工暂停"})
	if !errx.IsErrorByPbCode(err, extproto.Code__1_00_BIZ_STATE) {
		t.Fatalf("PausePlan() error = %v, want BIZ_STATE", err)
	}

	var got gormmodel.Plan
	if err := db.Where("id = ?", plan.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != model.PlanStatusTerminated || got.TerminatedReason.String != "并发终止" {
		t.Fatalf("plan status/reason = %d/%q, want terminated/并发终止", got.Status, got.TerminatedReason.String)
	}
	if got.PausedReason.Valid {
		t.Fatalf("paused_reason = %q, want empty", got.PausedReason.String)
	}
}

func TestResumePlanRejectsConcurrentTermination(t *testing.T) {
	db, svcCtx, plan, _ := newPlanControlTestRig(t, model.PlanStatusPaused, model.PlanStatusPaused)
	injectStatusChangeAfterLoad(t, db, "plan", plan.Id, map[string]any{
		"status":            model.PlanStatusTerminated,
		"finished_time":     time.Now(),
		"terminated_reason": "并发终止",
	}, &gormmodel.Plan{})
	ctx := authctx.WithUserID(context.Background(), "tester")

	_, err := NewResumePlanLogic(ctx, svcCtx).ResumePlan(&trigger.ResumePlanReq{Id: plan.Id})
	if !errx.IsErrorByPbCode(err, extproto.Code__1_00_BIZ_STATE) {
		t.Fatalf("ResumePlan() error = %v, want BIZ_STATE", err)
	}

	var got gormmodel.Plan
	if err := db.Where("id = ?", plan.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != model.PlanStatusTerminated || got.TerminatedReason.String != "并发终止" {
		t.Fatalf("plan status/reason = %d/%q, want terminated/并发终止", got.Status, got.TerminatedReason.String)
	}
}

func TestPausePlanBatchRejectsConcurrentTermination(t *testing.T) {
	db, svcCtx, _, batch := newPlanControlTestRig(t, model.PlanStatusEnabled, model.PlanStatusEnabled)
	injectStatusChangeAfterLoad(t, db, "plan_batch", batch.Id, map[string]any{
		"status":            model.PlanStatusTerminated,
		"finished_time":     time.Now(),
		"terminated_reason": "并发终止",
	}, &gormmodel.PlanBatch{})
	ctx := authctx.WithUserID(context.Background(), "tester")

	_, err := NewPausePlanBatchLogic(ctx, svcCtx).PausePlanBatch(&trigger.PausePlanBatchReq{Id: batch.Id, Reason: "人工暂停"})
	if !errx.IsErrorByPbCode(err, extproto.Code__1_00_BIZ_STATE) {
		t.Fatalf("PausePlanBatch() error = %v, want BIZ_STATE", err)
	}

	var got gormmodel.PlanBatch
	if err := db.Where("id = ?", batch.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != model.PlanStatusTerminated || got.TerminatedReason.String != "并发终止" {
		t.Fatalf("batch status/reason = %d/%q, want terminated/并发终止", got.Status, got.TerminatedReason.String)
	}
	if got.PausedReason.Valid {
		t.Fatalf("paused_reason = %q, want empty", got.PausedReason.String)
	}
}

func TestResumePlanBatchRejectsConcurrentTermination(t *testing.T) {
	db, svcCtx, _, batch := newPlanControlTestRig(t, model.PlanStatusEnabled, model.PlanStatusPaused)
	injectStatusChangeAfterLoad(t, db, "plan_batch", batch.Id, map[string]any{
		"status":            model.PlanStatusTerminated,
		"finished_time":     time.Now(),
		"terminated_reason": "并发终止",
	}, &gormmodel.PlanBatch{})
	ctx := authctx.WithUserID(context.Background(), "tester")

	_, err := NewResumePlanBatchLogic(ctx, svcCtx).ResumePlanBatch(&trigger.ResumePlanBatchReq{Id: batch.Id})
	if !errx.IsErrorByPbCode(err, extproto.Code__1_00_BIZ_STATE) {
		t.Fatalf("ResumePlanBatch() error = %v, want BIZ_STATE", err)
	}

	var got gormmodel.PlanBatch
	if err := db.Where("id = ?", batch.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != model.PlanStatusTerminated || got.TerminatedReason.String != "并发终止" {
		t.Fatalf("batch status/reason = %d/%q, want terminated/并发终止", got.Status, got.TerminatedReason.String)
	}
}
