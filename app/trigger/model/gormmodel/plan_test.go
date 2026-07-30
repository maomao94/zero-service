package gormmodel

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"zero-service/common/gormx"
	"zero-service/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCountRunningExecItems(t *testing.T) {
	db := openPlanTestDB(t)
	ctx := context.Background()
	planPk := "plan-pk"
	batchPk := "batch-pk"
	statuses := []int{
		model.StatusWaiting,
		model.StatusDelayed,
		model.StatusRunning,
		model.StatusPaused,
		model.StatusCompleted,
		model.StatusTerminated,
	}
	for i, status := range statuses {
		item := PlanExecItem{
			PlanPk:  planPk,
			BatchPk: batchPk,
			ExecId:  fmt.Sprintf("exec-%d", i),
			ItemId:  fmt.Sprintf("item-%d", i),
			Status:  status,
		}
		item.Id = fmt.Sprintf("item-pk-%d", i)
		if err := db.Create(&item).Error; err != nil {
			t.Fatal(err)
		}
	}
	planCount, err := CountRunningExecItemsByPlan(ctx, db, planPk)
	if err != nil {
		t.Fatal(err)
	}
	if planCount != 1 {
		t.Fatalf("plan running count = %d, want 1", planCount)
	}

	batchCount, err := CountRunningExecItemsByBatch(ctx, db, batchPk)
	if err != nil {
		t.Fatal(err)
	}
	if batchCount != 1 {
		t.Fatalf("batch running count = %d, want 1", batchCount)
	}
}

func TestLockTriggerItemRequiresEnabledParents(t *testing.T) {
	for _, tc := range []struct {
		name        string
		planStatus  int
		batchStatus int
		wantClaim   bool
	}{
		{name: "enabled hierarchy", planStatus: model.PlanStatusEnabled, batchStatus: model.PlanStatusEnabled, wantClaim: true},
		{name: "terminated plan", planStatus: model.PlanStatusTerminated, batchStatus: model.PlanStatusEnabled},
		{name: "terminated batch", planStatus: model.PlanStatusEnabled, batchStatus: model.PlanStatusTerminated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := openPlanTestDB(t)
			plan := Plan{PlanId: "plan", Status: tc.planStatus}
			plan.Id = "plan-pk"
			batch := PlanBatch{PlanPk: plan.Id, PlanId: plan.PlanId, BatchId: "batch", Status: tc.batchStatus}
			batch.Id = "batch-pk"
			item := PlanExecItem{
				PlanPk:          plan.Id,
				PlanId:          plan.PlanId,
				BatchPk:         batch.Id,
				BatchId:         batch.BatchId,
				ExecId:          "exec",
				ItemId:          "item",
				Status:          model.StatusWaiting,
				NextTriggerTime: time.Now().Add(-time.Minute),
			}
			item.Id = "item-pk"
			if err := db.Create(&plan).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Create(&batch).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Create(&item).Error; err != nil {
				t.Fatal(err)
			}

			claimed, err := LockTriggerItem(context.Background(), db, gormx.DatabaseSQLite, time.Minute)
			if !tc.wantClaim {
				if !errors.Is(err, model.ErrNotFound) {
					t.Fatalf("LockTriggerItem error = %v, want ErrNotFound", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if claimed.Id != item.Id {
				t.Fatalf("claimed id = %q, want %q", claimed.Id, item.Id)
			}
			var got PlanExecItem
			if err := db.First(&got, "id = ?", item.Id).Error; err != nil {
				t.Fatal(err)
			}
			if got.Status != model.StatusRunning {
				t.Fatalf("claimed status = %d, want running", got.Status)
			}
		})
	}
}

func TestLockTriggerItemUsesCandidateCAS(t *testing.T) {
	db := openPlanTestDB(t)
	plan, batch := createPlanHierarchy(t, db, nil)
	item := PlanExecItem{
		PlanPk:          plan.Id,
		PlanId:          plan.PlanId,
		BatchPk:         batch.Id,
		BatchId:         batch.BatchId,
		ExecId:          "exec",
		ItemId:          "item",
		Status:          model.StatusWaiting,
		NextTriggerTime: time.Now().Add(-time.Minute),
	}
	item.Id = "item-pk"
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}

	candidateTime := item.NextTriggerTime
	candidateStatus := item.Status
	candidateVersion := item.Version.Int64
	if err := db.Model(&PlanExecItem{}).Where("id = ?", item.Id).Updates(map[string]any{
		"status":            model.StatusPaused,
		"next_trigger_time": candidateTime.Add(time.Minute),
		"version":           candidateVersion + 1,
	}).Error; err != nil {
		t.Fatal(err)
	}

	result := db.Model(&PlanExecItem{}).
		Where("id = ?", item.Id).
		Where("next_trigger_time = ?", candidateTime).
		Where("status = ?", candidateStatus).
		Where("version = ?", candidateVersion).
		Updates(map[string]any{"status": model.StatusRunning})
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	if result.RowsAffected != 0 {
		t.Fatalf("stale candidate updated %d rows, want 0", result.RowsAffected)
	}
	var got PlanExecItem
	if err := db.First(&got, "id = ?", item.Id).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusPaused {
		t.Fatalf("item status = %d, want paused", got.Status)
	}
}

func TestUpdateExecItemStatusToRunningDoesNotReviveTerminatedItem(t *testing.T) {
	db := openPlanTestDB(t)
	item := PlanExecItem{ExecId: "exec", ItemId: "item", Status: model.StatusTerminated}
	item.Id = "item-pk"
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}

	err := UpdateExecItemStatusToRunning(context.Background(), db, item.Id, "")
	if !errors.Is(err, model.ErrNoRowsUpdate) {
		t.Fatalf("UpdateExecItemStatusToRunning error = %v, want ErrNoRowsUpdate", err)
	}
	var got PlanExecItem
	if err := db.First(&got, "id = ?", item.Id).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusTerminated {
		t.Fatalf("item status = %d, want terminated", got.Status)
	}
}

func TestTerminateParentUpdatesPreserveExecItems(t *testing.T) {
	statuses := []int{
		model.StatusWaiting,
		model.StatusDelayed,
		model.StatusPaused,
		model.StatusCompleted,
		model.StatusTerminated,
	}
	for _, scope := range []string{"plan", "batch"} {
		t.Run(scope, func(t *testing.T) {
			db := openPlanTestDB(t)
			plan, batch := createPlanHierarchy(t, db, statuses)

			var rows int64
			var err error
			if scope == "plan" {
				rows, err = UpdatePlanTerminated(context.Background(), db, plan.Id, "reason", "user", time.Now())
			} else {
				rows, err = UpdatePlanBatchTerminated(context.Background(), db, batch.Id, "reason", "user", time.Now())
			}
			if err != nil || rows != 1 {
				t.Fatalf("terminate rows = %d, err = %v, want 1, nil", rows, err)
			}
			assertExecItemStatuses(t, db, statuses)
		})
	}
}

func TestConditionalExecItemUpdateReportsStateConflict(t *testing.T) {
	db := openPlanTestDB(t)
	item := PlanExecItem{ExecId: "exec", ItemId: "item", Status: model.StatusTerminated}
	item.Id = "item-pk"
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}

	err := UpdateExecItemStatusToCompleted(context.Background(), db, item.Id, "message", "reason", []int{model.StatusRunning}, nil)
	if !errors.Is(err, model.ErrNoRowsUpdate) {
		t.Fatalf("conditional update error = %v, want ErrNoRowsUpdate", err)
	}
}

func createPlanHierarchy(t *testing.T, db *gorm.DB, statuses []int) (Plan, PlanBatch) {
	t.Helper()
	plan := Plan{PlanId: "plan", Status: model.PlanStatusEnabled}
	plan.Id = "plan-pk"
	batch := PlanBatch{PlanPk: plan.Id, PlanId: plan.PlanId, BatchId: "batch", Status: model.PlanStatusEnabled}
	batch.Id = "batch-pk"
	if err := db.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&batch).Error; err != nil {
		t.Fatal(err)
	}
	for i, status := range statuses {
		item := PlanExecItem{
			PlanPk:  plan.Id,
			PlanId:  plan.PlanId,
			BatchPk: batch.Id,
			BatchId: batch.BatchId,
			ExecId:  fmt.Sprintf("exec-%d", i),
			ItemId:  fmt.Sprintf("item-%d", i),
			Status:  status,
		}
		item.Id = fmt.Sprintf("item-pk-%d", i)
		if err := db.Create(&item).Error; err != nil {
			t.Fatal(err)
		}
	}
	return plan, batch
}

func assertExecItemStatuses(t *testing.T, db *gorm.DB, want []int) {
	t.Helper()
	var items []PlanExecItem
	if err := db.Order("id").Find(&items).Error; err != nil {
		t.Fatal(err)
	}
	if len(items) != len(want) {
		t.Fatalf("item count = %d, want %d", len(items), len(want))
	}
	for i := range items {
		if items[i].Status != want[i] {
			t.Fatalf("item %d status = %d, want %d", i, items[i].Status, want[i])
		}
	}
}

func openPlanTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&parseTime=true&_loc=auto"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Plan{}, &PlanBatch{}, &PlanExecItem{}); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}
