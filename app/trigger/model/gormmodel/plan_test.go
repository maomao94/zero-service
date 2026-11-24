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

func TestTerminateUnfinishedExecItems(t *testing.T) {
	statuses := []int{
		model.StatusWaiting,
		model.StatusDelayed,
		model.StatusRunning,
		model.StatusPaused,
		model.StatusCompleted,
		model.StatusTerminated,
	}
	for _, scope := range []string{"plan", "batch"} {
		t.Run(scope, func(t *testing.T) {
			db := openPlanTestDB(t)
			plan, batch := createPlanHierarchy(t, db, statuses)

			// 兄弟批次用于验证终止作用域：批次级终止不能波及同计划的其它批次，计划级终止则应一并收敛。
			siblingBatch := PlanBatch{PlanPk: plan.Id, PlanId: plan.PlanId, BatchId: "batch-sibling", Status: model.PlanStatusEnabled}
			siblingBatch.Id = "batch-pk-sibling"
			if err := db.Create(&siblingBatch).Error; err != nil {
				t.Fatal(err)
			}
			siblingItem := PlanExecItem{
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

			now := time.Now()
			wantRows := int64(4)
			wantSiblingStatus := model.StatusWaiting
			var rows int64
			var err error
			if scope == "plan" {
				wantRows = 5
				wantSiblingStatus = model.StatusTerminated
				rows, err = TerminateUnfinishedExecItemsByPlan(context.Background(), db, plan.Id, "终止原因", "tester", now)
			} else {
				rows, err = TerminateUnfinishedExecItemsByBatch(context.Background(), db, batch.Id, "终止原因", "tester", now)
			}
			if err != nil {
				t.Fatal(err)
			}
			if rows != wantRows {
				t.Fatalf("terminated rows = %d, want %d", rows, wantRows)
			}
			assertExecItemStatusesInBatch(t, db, batch.Id, []int{
				model.StatusTerminated,
				model.StatusTerminated,
				model.StatusTerminated,
				model.StatusTerminated,
				model.StatusCompleted,
				model.StatusTerminated,
			})
			assertExecItemStatusesInBatch(t, db, siblingBatch.Id, []int{wantSiblingStatus})

			var items []PlanExecItem
			if err := db.Where("batch_pk = ?", batch.Id).Order("id").Find(&items).Error; err != nil {
				t.Fatal(err)
			}
			for i := range items[:4] {
				if items[i].TerminatedReason.String != "终止原因" {
					t.Fatalf("item %d terminated reason = %q, want 终止原因", i, items[i].TerminatedReason.String)
				}
				if items[i].UpdateUser.String != "tester" {
					t.Fatalf("item %d update user = %q, want tester", i, items[i].UpdateUser.String)
				}
			}
			// 已完成/已终止的执行项保持原状态与原终止原因。
			if items[4].Status != model.StatusCompleted {
				t.Fatalf("completed item status = %d, want completed", items[4].Status)
			}
			if items[5].TerminatedReason.Valid {
				t.Fatalf("pre-terminated item reason = %q, want empty", items[5].TerminatedReason.String)
			}
		})
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

func TestDeferTriggerItemRestoresStatusAndTime(t *testing.T) {
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

	claimed, err := LockTriggerItem(context.Background(), db, gormx.DatabaseSQLite, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	next := time.Now().Add(time.Minute).Truncate(time.Second)
	if err := DeferTriggerItem(context.Background(), db, claimed, next); err != nil {
		t.Fatal(err)
	}

	var got PlanExecItem
	if err := db.First(&got, "id = ?", item.Id).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusWaiting {
		t.Fatalf("deferred status = %d, want waiting", got.Status)
	}
	if !got.NextTriggerTime.Equal(next) {
		t.Fatalf("deferred next_trigger_time = %s, want %s", got.NextTriggerTime, next)
	}
	// 抢占时会写入 last_trigger_time，推迟后必须回到抢占前的空值，避免展示成“已触发过一次”。
	if got.LastTriggerTime.Valid {
		t.Fatalf("deferred last_trigger_time = %v, want NULL", got.LastTriggerTime)
	}

	// 执行项已被其他链路改为终态时，不应再被限速逻辑回退。
	if err := db.Model(&PlanExecItem{}).Where("id = ?", item.Id).Update("status", model.StatusCompleted).Error; err != nil {
		t.Fatal(err)
	}
	err = DeferTriggerItem(context.Background(), db, claimed, next)
	if !errors.Is(err, model.ErrNoRowsUpdate) {
		t.Fatalf("DeferTriggerItem on terminal item error = %v, want ErrNoRowsUpdate", err)
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
	assertExecItemStatusesOf(t, items, want)
}

// assertExecItemStatusesInBatch 只校验指定批次的执行项，用于验证终止作用域不越界。
func assertExecItemStatusesInBatch(t *testing.T, db *gorm.DB, batchPk string, want []int) {
	t.Helper()
	var items []PlanExecItem
	if err := db.Where("batch_pk = ?", batchPk).Order("id").Find(&items).Error; err != nil {
		t.Fatal(err)
	}
	assertExecItemStatusesOf(t, items, want)
}

func assertExecItemStatusesOf(t *testing.T, items []PlanExecItem, want []int) {
	t.Helper()
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
	if err := db.AutoMigrate(&Plan{}, &PlanBatch{}, &PlanExecItem{}, &PlanConfirm{}); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}
