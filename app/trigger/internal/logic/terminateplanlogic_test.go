package logic

import (
	"context"
	"fmt"
	"testing"
	"time"

	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/authctx"
	"zero-service/model"

	"gorm.io/gorm"
)

func TestTerminatePlanBatchTerminatesUnfinishedItems(t *testing.T) {
	db := openConfirmTestDB(t)
	svcCtx := newConfirmTestServiceContext(t, db)
	ctx := authctx.WithUserID(context.Background(), "tester")

	plan := gormmodel.Plan{PlanId: "plan", Status: model.PlanStatusEnabled}
	plan.Id = "plan-pk"
	batch := gormmodel.PlanBatch{PlanPk: plan.Id, PlanId: plan.PlanId, BatchId: "batch", Status: model.PlanStatusEnabled}
	batch.Id = "batch-pk"
	// 同计划的兄弟批次：批次级终止不能波及其它批次。
	siblingBatch := gormmodel.PlanBatch{PlanPk: plan.Id, PlanId: plan.PlanId, BatchId: "batch-sibling", Status: model.PlanStatusEnabled}
	siblingBatch.Id = "batch-pk-sibling"
	if err := db.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	for _, b := range []*gormmodel.PlanBatch{&batch, &siblingBatch} {
		if err := db.Create(b).Error; err != nil {
			t.Fatal(err)
		}
	}

	statuses := []int{
		model.StatusWaiting,
		model.StatusDelayed,
		model.StatusRunning,
		model.StatusPaused,
		model.StatusCompleted,
	}
	wantStatus := make(map[string]int, len(statuses)+1)
	for i, status := range statuses {
		execID := fmt.Sprintf("exec-%d", i)
		wantStatus[execID] = model.StatusTerminated
		if status == model.StatusCompleted {
			wantStatus[execID] = model.StatusCompleted
		}
		item := gormmodel.PlanExecItem{
			PlanPk:  plan.Id,
			PlanId:  plan.PlanId,
			BatchPk: batch.Id,
			BatchId: batch.BatchId,
			ExecId:  execID,
			ItemId:  fmt.Sprintf("item-%d", i),
			Status:  status,
		}
		item.Id = fmt.Sprintf("item-pk-%d", i)
		if err := db.Create(&item).Error; err != nil {
			t.Fatal(err)
		}
	}
	siblingItem := gormmodel.PlanExecItem{
		PlanPk:  plan.Id,
		PlanId:  plan.PlanId,
		BatchPk: siblingBatch.Id,
		BatchId: siblingBatch.BatchId,
		ExecId:  "exec-sibling",
		ItemId:  "item-sibling",
		Status:  model.StatusWaiting,
	}
	siblingItem.Id = "item-pk-sibling"
	if err := db.Create(&siblingItem).Error; err != nil {
		t.Fatal(err)
	}
	wantStatus[siblingItem.ExecId] = model.StatusWaiting

	if _, err := NewTerminatePlanBatchLogic(ctx, svcCtx).TerminatePlanBatch(&trigger.TerminatePlanBatchReq{
		Id:     batch.Id,
		Reason: "人工终止批次",
	}); err != nil {
		t.Fatal(err)
	}

	var gotBatch gormmodel.PlanBatch
	if err := db.Where("id = ?", batch.Id).First(&gotBatch).Error; err != nil {
		t.Fatal(err)
	}
	if gotBatch.Status != model.PlanStatusTerminated || !gotBatch.FinishedTime.Valid {
		t.Fatalf("batch status/finished = %d/%v, want terminated/true", gotBatch.Status, gotBatch.FinishedTime.Valid)
	}

	var gotSiblingBatch gormmodel.PlanBatch
	if err := db.Where("id = ?", siblingBatch.Id).First(&gotSiblingBatch).Error; err != nil {
		t.Fatal(err)
	}
	if gotSiblingBatch.Status != model.PlanStatusEnabled || gotSiblingBatch.FinishedTime.Valid {
		t.Fatalf("sibling batch status/finished = %d/%v, want enabled/false",
			gotSiblingBatch.Status, gotSiblingBatch.FinishedTime.Valid)
	}

	assertTerminatedExecItems(t, db, wantStatus, "人工终止批次")

	var gotSiblingItem gormmodel.PlanExecItem
	if err := db.Where("id = ?", siblingItem.Id).First(&gotSiblingItem).Error; err != nil {
		t.Fatal(err)
	}
	if gotSiblingItem.TerminatedReason.Valid {
		t.Fatalf("sibling item terminated reason = %q, want empty", gotSiblingItem.TerminatedReason.String)
	}
}

func TestTerminatePlanTerminatesUnfinishedItems(t *testing.T) {
	db := openConfirmTestDB(t)
	svcCtx := newConfirmTestServiceContext(t, db)
	ctx := authctx.WithUserID(context.Background(), "tester")

	plan := gormmodel.Plan{PlanId: "plan", Status: model.PlanStatusEnabled}
	plan.Id = "plan-pk"
	if err := db.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}

	batches := []*gormmodel.PlanBatch{
		{PlanPk: plan.Id, PlanId: plan.PlanId, BatchId: "batch-1", Status: model.PlanStatusEnabled},
		{PlanPk: plan.Id, PlanId: plan.PlanId, BatchId: "batch-2", Status: model.PlanStatusEnabled},
	}
	for i, batch := range batches {
		batch.Id = fmt.Sprintf("batch-pk-%d", i)
		if err := db.Create(batch).Error; err != nil {
			t.Fatal(err)
		}
	}

	statuses := []int{model.StatusRunning, model.StatusWaiting, model.StatusDelayed, model.StatusCompleted}
	wantStatus := make(map[string]int, len(statuses))
	for i, status := range statuses {
		execID := fmt.Sprintf("exec-%d", i)
		wantStatus[execID] = model.StatusTerminated
		if status == model.StatusCompleted {
			wantStatus[execID] = model.StatusCompleted
		}
		batch := batches[i%len(batches)]
		item := gormmodel.PlanExecItem{
			PlanPk:  plan.Id,
			PlanId:  plan.PlanId,
			BatchPk: batch.Id,
			BatchId: batch.BatchId,
			ExecId:  execID,
			ItemId:  fmt.Sprintf("item-%d", i),
			Status:  status,
		}
		item.Id = fmt.Sprintf("item-pk-%d", i)
		if err := db.Create(&item).Error; err != nil {
			t.Fatal(err)
		}
	}

	if _, err := NewTerminatePlanLogic(ctx, svcCtx).TerminatePlan(&trigger.TerminatePlanReq{
		Id:     plan.Id,
		Reason: "人工终止计划",
	}); err != nil {
		t.Fatal(err)
	}

	var gotPlan gormmodel.Plan
	if err := db.Where("id = ?", plan.Id).First(&gotPlan).Error; err != nil {
		t.Fatal(err)
	}
	if gotPlan.Status != model.PlanStatusTerminated || !gotPlan.FinishedTime.Valid {
		t.Fatalf("plan status/finished = %d/%v, want terminated/true", gotPlan.Status, gotPlan.FinishedTime.Valid)
	}

	for _, batch := range batches {
		var gotBatch gormmodel.PlanBatch
		if err := db.Where("id = ?", batch.Id).First(&gotBatch).Error; err != nil {
			t.Fatal(err)
		}
		if gotBatch.Status != model.PlanStatusTerminated || !gotBatch.FinishedTime.Valid {
			t.Fatalf("batch %s status/finished = %d/%v, want terminated/true",
				batch.BatchId, gotBatch.Status, gotBatch.FinishedTime.Valid)
		}
	}

	assertTerminatedExecItems(t, db, wantStatus, "人工终止计划")
}

// assertTerminatedExecItems 按 exec_id 校验执行项终态，避免依赖主键排序。
func assertTerminatedExecItems(t *testing.T, db *gorm.DB, wantStatus map[string]int, reason string) {
	t.Helper()
	var items []gormmodel.PlanExecItem
	if err := db.Find(&items).Error; err != nil {
		t.Fatal(err)
	}
	if len(items) != len(wantStatus) {
		t.Fatalf("item count = %d, want %d", len(items), len(wantStatus))
	}
	for _, item := range items {
		want, ok := wantStatus[item.ExecId]
		if !ok {
			t.Fatalf("unexpected exec id %q", item.ExecId)
		}
		if item.Status != want {
			t.Fatalf("item %q status = %d, want %d", item.ExecId, item.Status, want)
		}
		if want == model.StatusTerminated && item.TerminatedReason.String != reason {
			t.Fatalf("item %q terminated reason = %q, want %q", item.ExecId, item.TerminatedReason.String, reason)
		}
	}
}

func TestTerminatePlanBatchDefaultsTerminatedReason(t *testing.T) {
	db := openConfirmTestDB(t)
	svcCtx := newConfirmTestServiceContext(t, db)

	plan := gormmodel.Plan{PlanId: "plan", Status: model.PlanStatusEnabled}
	plan.Id = "plan-pk"
	batch := gormmodel.PlanBatch{PlanPk: plan.Id, PlanId: plan.PlanId, BatchId: "batch", Status: model.PlanStatusEnabled}
	batch.Id = "batch-pk"
	if err := db.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&batch).Error; err != nil {
		t.Fatal(err)
	}
	item := gormmodel.PlanExecItem{
		PlanPk:  plan.Id,
		PlanId:  plan.PlanId,
		BatchPk: batch.Id,
		BatchId: batch.BatchId,
		ExecId:  "exec",
		ItemId:  "item",
		Status:  model.StatusRunning,
	}
	item.Id = "item-pk"
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	confirmRecord := gormmodel.PlanConfirm{
		PlanPk:      plan.Id,
		BatchPk:     batch.Id,
		ConfirmType: model.ConfirmTypeBatch,
		TargetPk:    batch.Id,
		NotifyAt:    time.Now().Add(-time.Minute),
		Deadline:    time.Now().Add(time.Hour),
	}
	if err := db.Create(&confirmRecord).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := NewTerminatePlanBatchLogic(authctx.WithUserID(context.Background(), "tester"), svcCtx).
		TerminatePlanBatch(&trigger.TerminatePlanBatchReq{Id: batch.Id}); err != nil {
		t.Fatal(err)
	}

	const wantReason = "计划批次被终止"
	var gotBatch gormmodel.PlanBatch
	if err := db.Where("id = ?", batch.Id).First(&gotBatch).Error; err != nil {
		t.Fatal(err)
	}
	if gotBatch.TerminatedReason.String != wantReason {
		t.Fatalf("batch terminated reason = %q, want %q", gotBatch.TerminatedReason.String, wantReason)
	}
	var gotItem gormmodel.PlanExecItem
	if err := db.Where("id = ?", item.Id).First(&gotItem).Error; err != nil {
		t.Fatal(err)
	}
	if gotItem.TerminatedReason.String != wantReason {
		t.Fatalf("item terminated reason = %q, want %q", gotItem.TerminatedReason.String, wantReason)
	}
	var gotConfirm gormmodel.PlanConfirm
	if err := db.Where("id = ?", confirmRecord.Id).First(&gotConfirm).Error; err != nil {
		t.Fatal(err)
	}
	if gotConfirm.ConfirmStatus != model.PlanConfirmStatusConfirmed || gotConfirm.Reason.String != wantReason {
		t.Fatalf("confirm status/reason = %d/%q, want confirmed/%q",
			gotConfirm.ConfirmStatus, gotConfirm.Reason.String, wantReason)
	}
}

func TestTerminatePlanDefaultsTerminatedReason(t *testing.T) {
	db := openConfirmTestDB(t)
	svcCtx := newConfirmTestServiceContext(t, db)

	plan := gormmodel.Plan{PlanId: "plan", Status: model.PlanStatusEnabled}
	plan.Id = "plan-pk"
	batch := gormmodel.PlanBatch{PlanPk: plan.Id, PlanId: plan.PlanId, BatchId: "batch", Status: model.PlanStatusEnabled}
	batch.Id = "batch-pk"
	if err := db.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&batch).Error; err != nil {
		t.Fatal(err)
	}
	item := gormmodel.PlanExecItem{
		PlanPk:  plan.Id,
		PlanId:  plan.PlanId,
		BatchPk: batch.Id,
		BatchId: batch.BatchId,
		ExecId:  "exec",
		ItemId:  "item",
		Status:  model.StatusWaiting,
	}
	item.Id = "item-pk"
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	confirmRecord := gormmodel.PlanConfirm{
		PlanPk:      plan.Id,
		BatchPk:     batch.Id,
		ConfirmType: model.ConfirmTypeBatch,
		TargetPk:    batch.Id,
		NotifyAt:    time.Now().Add(-time.Minute),
		Deadline:    time.Now().Add(time.Hour),
	}
	if err := db.Create(&confirmRecord).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := NewTerminatePlanLogic(authctx.WithUserID(context.Background(), "tester"), svcCtx).
		TerminatePlan(&trigger.TerminatePlanReq{Id: plan.Id}); err != nil {
		t.Fatal(err)
	}

	const wantReason = "计划被终止"
	var gotPlan gormmodel.Plan
	if err := db.Where("id = ?", plan.Id).First(&gotPlan).Error; err != nil {
		t.Fatal(err)
	}
	if gotPlan.TerminatedReason.String != wantReason {
		t.Fatalf("plan terminated reason = %q, want %q", gotPlan.TerminatedReason.String, wantReason)
	}
	var gotBatch gormmodel.PlanBatch
	if err := db.Where("id = ?", batch.Id).First(&gotBatch).Error; err != nil {
		t.Fatal(err)
	}
	if gotBatch.TerminatedReason.String != wantReason {
		t.Fatalf("batch terminated reason = %q, want %q", gotBatch.TerminatedReason.String, wantReason)
	}
	var gotItem gormmodel.PlanExecItem
	if err := db.Where("id = ?", item.Id).First(&gotItem).Error; err != nil {
		t.Fatal(err)
	}
	if gotItem.TerminatedReason.String != wantReason {
		t.Fatalf("item terminated reason = %q, want %q", gotItem.TerminatedReason.String, wantReason)
	}
	var gotConfirm gormmodel.PlanConfirm
	if err := db.Where("id = ?", confirmRecord.Id).First(&gotConfirm).Error; err != nil {
		t.Fatal(err)
	}
	if gotConfirm.ConfirmStatus != model.PlanConfirmStatusConfirmed || gotConfirm.Reason.String != wantReason {
		t.Fatalf("confirm status/reason = %d/%q, want confirmed/%q",
			gotConfirm.ConfirmStatus, gotConfirm.Reason.String, wantReason)
	}
}
