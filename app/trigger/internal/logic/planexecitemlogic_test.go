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
	"zero-service/model"

	"gorm.io/gorm"
)

// newExecItemTestRig 准备「启用的计划 + 启用的批次 + 一条指定状态的执行项」。
func newExecItemTestRig(t *testing.T, status int) (*gorm.DB, *svc.ServiceContext, gormmodel.PlanExecItem) {
	t.Helper()
	db := openConfirmTestDB(t)
	svcCtx := newConfirmTestServiceContext(t, db)

	plan := gormmodel.Plan{PlanId: "plan", Status: model.PlanStatusEnabled}
	plan.Id = "plan-pk"
	batch := gormmodel.PlanBatch{PlanPk: plan.Id, PlanId: plan.PlanId, BatchId: "batch", Status: model.PlanStatusEnabled}
	batch.Id = "batch-pk"
	item := gormmodel.PlanExecItem{
		PlanPk:          plan.Id,
		PlanId:          plan.PlanId,
		BatchPk:         batch.Id,
		BatchId:         batch.BatchId,
		ExecId:          "exec",
		ItemId:          "item",
		Status:          status,
		NextTriggerTime: time.Now().Add(time.Hour),
	}
	item.Id = "item-pk"
	if status == model.StatusPaused {
		item.PausedTime = sql.NullTime{Time: time.Now().Add(-time.Minute), Valid: true}
		item.PausedReason = sql.NullString{String: "历史暂停", Valid: true}
	}
	for _, value := range []any{&plan, &batch, &item} {
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db, svcCtx, item
}

func TestRunPlanExecItemUpdatesNextTriggerTimeWithoutChangingStatus(t *testing.T) {
	db, svcCtx, item := newExecItemTestRig(t, model.StatusWaiting)
	ctx := authctx.WithUserID(context.Background(), "tester")

	if _, err := NewRunPlanExecItemLogic(ctx, svcCtx).RunPlanExecItem(&trigger.RunPlanExecItemReq{Id: item.Id}); err != nil {
		t.Fatal(err)
	}

	var got gormmodel.PlanExecItem
	if err := db.Where("id = ?", item.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusWaiting {
		t.Fatalf("status = %d, want waiting", got.Status)
	}
	// 原 next_trigger_time 是 +1h，更新成功后应回到当前时间附近。
	if got.NextTriggerTime.After(time.Now().Add(time.Minute)) {
		t.Fatalf("next_trigger_time = %s, want ~now", got.NextTriggerTime)
	}
	if got.UpdateUser.String != "tester" {
		t.Fatalf("update_user = %q, want tester", got.UpdateUser.String)
	}
}

func TestPausePlanExecItemWritesPausedFields(t *testing.T) {
	db, svcCtx, item := newExecItemTestRig(t, model.StatusWaiting)
	ctx := authctx.WithUserID(context.Background(), "tester")

	if _, err := NewPausePlanExecItemLogic(ctx, svcCtx).PausePlanExecItem(&trigger.PausePlanExecItemReq{
		Id:     item.Id,
		Reason: "人工暂停",
	}); err != nil {
		t.Fatal(err)
	}

	var got gormmodel.PlanExecItem
	if err := db.Where("id = ?", item.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusPaused {
		t.Fatalf("status = %d, want paused", got.Status)
	}
	if !got.PausedTime.Valid || got.PausedReason.String != "人工暂停" {
		t.Fatalf("paused_time/reason = %v/%q, want valid/人工暂停", got.PausedTime, got.PausedReason.String)
	}
	if got.UpdateUser.String != "tester" {
		t.Fatalf("update_user = %q, want tester", got.UpdateUser.String)
	}
}

func TestResumePlanExecItemClearsPausedFields(t *testing.T) {
	db, svcCtx, item := newExecItemTestRig(t, model.StatusPaused)
	ctx := authctx.WithUserID(context.Background(), "tester")

	if _, err := NewResumePlanExecItemLogic(ctx, svcCtx).ResumePlanExecItem(&trigger.ResumePlanExecItemReq{
		Id: item.Id,
	}); err != nil {
		t.Fatal(err)
	}

	var got gormmodel.PlanExecItem
	if err := db.Where("id = ?", item.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusWaiting {
		t.Fatalf("status = %d, want waiting", got.Status)
	}
	if got.PausedTime.Valid || got.PausedReason.Valid {
		t.Fatalf("paused_time/reason = %v/%q, want cleared", got.PausedTime, got.PausedReason.String)
	}
	if got.UpdateUser.String != "tester" {
		t.Fatalf("update_user = %q, want tester", got.UpdateUser.String)
	}
}

func TestTerminatePlanExecItemWritesTerminatedReasonAndClearsPaused(t *testing.T) {
	db, svcCtx, item := newExecItemTestRig(t, model.StatusPaused)
	ctx := authctx.WithUserID(context.Background(), "tester")

	if _, err := NewTerminatePlanExecItemLogic(ctx, svcCtx).TerminatePlanExecItem(&trigger.TerminatePlanExecItemReq{
		Id:     item.Id,
		Reason: "人工终止",
	}); err != nil {
		t.Fatal(err)
	}

	var got gormmodel.PlanExecItem
	if err := db.Where("id = ?", item.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusTerminated {
		t.Fatalf("status = %d, want terminated", got.Status)
	}
	if got.TerminatedReason.String != "人工终止" {
		t.Fatalf("terminated_reason = %q, want 人工终止", got.TerminatedReason.String)
	}
	if got.PausedTime.Valid || got.PausedReason.Valid {
		t.Fatalf("paused_time/reason = %v/%q, want cleared", got.PausedTime, got.PausedReason.String)
	}
	if got.UpdateUser.String != "tester" {
		t.Fatalf("update_user = %q, want tester", got.UpdateUser.String)
	}
}

func TestTerminatePlanExecItemDefaultsTerminatedReason(t *testing.T) {
	db, svcCtx, item := newExecItemTestRig(t, model.StatusWaiting)
	ctx := authctx.WithUserID(context.Background(), "tester")

	if _, err := NewTerminatePlanExecItemLogic(ctx, svcCtx).TerminatePlanExecItem(&trigger.TerminatePlanExecItemReq{
		Id: item.Id,
	}); err != nil {
		t.Fatal(err)
	}

	var got gormmodel.PlanExecItem
	if err := db.Where("id = ?", item.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusTerminated {
		t.Fatalf("status = %d, want terminated", got.Status)
	}
	if got.TerminatedReason.String != "执行项被终止" {
		t.Fatalf("terminated_reason = %q, want 执行项被终止", got.TerminatedReason.String)
	}
}

// injectStatusChangeAfterLoad 在 RPC 读取目标行之后、写回之前，模拟其它链路改掉状态。
// 用于回归“更新必须带上读取时的状态条件”，避免并发下把终态写回非终态。
func injectStatusChangeAfterLoad(t *testing.T, db *gorm.DB, table, id string, updates map[string]any, model any) {
	t.Helper()
	injected := false
	err := db.Callback().Query().After("gorm:query").Register("test:change_status_after_load", func(tx *gorm.DB) {
		if injected || tx.Statement.Table != table {
			return
		}
		injected = true
		if err := db.Model(model).Where("id = ?", id).Updates(updates).Error; err != nil {
			t.Errorf("注入状态变更失败: %v", err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunPlanExecItemRejectsConcurrentStatusChange(t *testing.T) {
	db, svcCtx, item := newExecItemTestRig(t, model.StatusWaiting)
	injectStatusChangeAfterLoad(t, db, "plan_exec_item", item.Id, map[string]any{
		"status":            model.StatusTerminated,
		"terminated_reason": "并发终止",
	}, &gormmodel.PlanExecItem{})
	ctx := authctx.WithUserID(context.Background(), "tester")

	if _, err := NewRunPlanExecItemLogic(ctx, svcCtx).RunPlanExecItem(&trigger.RunPlanExecItemReq{Id: item.Id}); err == nil {
		t.Fatal("RunPlanExecItem() error = nil, want 状态已变化")
	}

	var got gormmodel.PlanExecItem
	if err := db.Where("id = ?", item.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusTerminated || got.TerminatedReason.String != "并发终止" {
		t.Fatalf("status/reason = %d/%q, want terminated/并发终止", got.Status, got.TerminatedReason.String)
	}
	// 竞争失败时不能改写 next_trigger_time（原始值在 1 小时后）。
	if !got.NextTriggerTime.After(time.Now().Add(30 * time.Minute)) {
		t.Fatalf("next_trigger_time = %s, want unchanged", got.NextTriggerTime)
	}
}

func TestPausePlanExecItemRejectsConcurrentStatusChange(t *testing.T) {
	db, svcCtx, item := newExecItemTestRig(t, model.StatusWaiting)
	injectStatusChangeAfterLoad(t, db, "plan_exec_item", item.Id, map[string]any{
		"status":            model.StatusTerminated,
		"terminated_reason": "并发终止",
	}, &gormmodel.PlanExecItem{})
	ctx := authctx.WithUserID(context.Background(), "tester")

	if _, err := NewPausePlanExecItemLogic(ctx, svcCtx).PausePlanExecItem(&trigger.PausePlanExecItemReq{
		Id:     item.Id,
		Reason: "人工暂停",
	}); err == nil {
		t.Fatal("PausePlanExecItem() error = nil, want 状态已变化")
	}

	var got gormmodel.PlanExecItem
	if err := db.Where("id = ?", item.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusTerminated || got.TerminatedReason.String != "并发终止" {
		t.Fatalf("status/reason = %d/%q, want terminated/并发终止", got.Status, got.TerminatedReason.String)
	}
	if got.PausedTime.Valid || got.PausedReason.Valid {
		t.Fatalf("paused_time/reason = %v/%q, want empty", got.PausedTime, got.PausedReason.String)
	}
}

func TestResumePlanExecItemRejectsConcurrentStatusChange(t *testing.T) {
	db, svcCtx, item := newExecItemTestRig(t, model.StatusPaused)
	injectStatusChangeAfterLoad(t, db, "plan_exec_item", item.Id, map[string]any{
		"status":            model.StatusTerminated,
		"terminated_reason": "并发终止",
	}, &gormmodel.PlanExecItem{})
	ctx := authctx.WithUserID(context.Background(), "tester")

	if _, err := NewResumePlanExecItemLogic(ctx, svcCtx).ResumePlanExecItem(&trigger.ResumePlanExecItemReq{
		Id: item.Id,
	}); err == nil {
		t.Fatal("ResumePlanExecItem() error = nil, want 状态已变化")
	}

	var got gormmodel.PlanExecItem
	if err := db.Where("id = ?", item.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusTerminated || got.TerminatedReason.String != "并发终止" {
		t.Fatalf("status/reason = %d/%q, want terminated/并发终止", got.Status, got.TerminatedReason.String)
	}
}

func TestTerminatePlanExecItemRejectsConcurrentCompletion(t *testing.T) {
	db, svcCtx, item := newExecItemTestRig(t, model.StatusRunning)
	injectStatusChangeAfterLoad(t, db, "plan_exec_item", item.Id, map[string]any{
		"status": model.StatusCompleted,
	}, &gormmodel.PlanExecItem{})
	ctx := authctx.WithUserID(context.Background(), "tester")

	if _, err := NewTerminatePlanExecItemLogic(ctx, svcCtx).TerminatePlanExecItem(&trigger.TerminatePlanExecItemReq{
		Id:     item.Id,
		Reason: "人工终止",
	}); err == nil {
		t.Fatal("TerminatePlanExecItem() error = nil, want 状态已变化")
	}

	var got gormmodel.PlanExecItem
	if err := db.Where("id = ?", item.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusCompleted {
		t.Fatalf("status = %d, want completed", got.Status)
	}
	if got.TerminatedReason.Valid {
		t.Fatalf("terminated_reason = %q, want empty", got.TerminatedReason.String)
	}
}
