package gormmodel

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"zero-service/common/gormx"
	"zero-service/model"
)

func TestLockTriggerItemBlockedByPendingConfirm(t *testing.T) {
	for _, tc := range []struct {
		name        string
		withConfirm bool
		confirmed   bool
		wantClaim   bool
	}{
		{name: "未配置确认时行为不变", wantClaim: true},
		{name: "待确认批次不下发", withConfirm: true},
		{name: "已确认批次正常下发", withConfirm: true, confirmed: true, wantClaim: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := openPlanTestDB(t)
			_, batch := createPlanHierarchy(t, db, []int{model.StatusWaiting})
			if err := db.Model(&PlanExecItem{}).Where("batch_pk = ?", batch.Id).
				Update("next_trigger_time", time.Now().Add(-time.Minute)).Error; err != nil {
				t.Fatal(err)
			}
			if tc.withConfirm {
				confirm := PlanConfirm{
					PlanPk:      batch.PlanPk,
					BatchPk:     batch.Id,
					ConfirmType: model.ConfirmTypeBatch,
					TargetPk:    batch.Id,
					NotifyAt:    time.Now().Add(-time.Minute),
					Deadline:    time.Now().Add(time.Hour),
				}
				if tc.confirmed {
					confirm.ConfirmedTime = sql.NullTime{Time: time.Now(), Valid: true}
				}
				if err := db.Create(&confirm).Error; err != nil {
					t.Fatal(err)
				}
			}

			item, err := LockTriggerItem(context.Background(), db, gormx.DatabaseSQLite, 5*time.Minute)
			if tc.wantClaim {
				if err != nil || item == nil {
					t.Fatalf("LockTriggerItem() = %v, %v; want claimed item", item, err)
				}
				return
			}
			if !errors.Is(err, model.ErrNotFound) {
				t.Fatalf("LockTriggerItem() error = %v, want ErrNotFound", err)
			}
			if item != nil {
				t.Fatalf("LockTriggerItem() item = %v, want nil", item)
			}
		})
	}
}

func TestClaimPlanConfirmNotifyIsIdempotent(t *testing.T) {
	db := openPlanTestDB(t)
	ctx := context.Background()
	now := time.Now()
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
	confirm := PlanConfirm{
		PlanPk:      plan.Id,
		BatchPk:     batch.Id,
		ConfirmType: model.ConfirmTypeBatch,
		TargetPk:    batch.Id,
		NotifyAt:    now.Add(-time.Minute),
		Deadline:    now.Add(time.Hour),
	}
	if err := db.Create(&confirm).Error; err != nil {
		t.Fatal(err)
	}

	got, err := ClaimPlanConfirmNotify(ctx, db, now)
	if err != nil || got == nil {
		t.Fatalf("ClaimPlanConfirmNotify() = %v, %v; want claimed record", got, err)
	}
	second, err := ClaimPlanConfirmNotify(ctx, db, now)
	if err != nil {
		t.Fatal(err)
	}
	if second != nil {
		t.Fatalf("ClaimPlanConfirmNotify() second = %v, want nil", second)
	}
}

func TestClaimPlanConfirmExpireSkipsConfirmed(t *testing.T) {
	db := openPlanTestDB(t)
	ctx := context.Background()
	now := time.Now()

	plan := Plan{PlanId: "plan", Status: model.PlanStatusEnabled}
	plan.Id = "plan-pk"
	if err := db.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"batch-pending", "batch-confirmed"} {
		batch := PlanBatch{PlanPk: plan.Id, PlanId: plan.PlanId, BatchId: id, Status: model.PlanStatusEnabled}
		batch.Id = id
		if err := db.Create(&batch).Error; err != nil {
			t.Fatal(err)
		}
	}

	pending := PlanConfirm{
		PlanPk:      plan.Id,
		BatchPk:     "batch-pending",
		ConfirmType: model.ConfirmTypeBatch,
		TargetPk:    "batch-pending",
		NotifyAt:    now.Add(-time.Hour),
		Deadline:    now.Add(-time.Minute),
	}
	confirmed := PlanConfirm{
		PlanPk:        plan.Id,
		BatchPk:       "batch-confirmed",
		ConfirmType:   model.ConfirmTypeBatch,
		TargetPk:      "batch-confirmed",
		NotifyAt:      now.Add(-time.Hour),
		Deadline:      now.Add(-time.Minute),
		ConfirmedTime: sql.NullTime{Time: now.Add(-time.Minute), Valid: true},
	}
	if err := db.Create(&pending).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&confirmed).Error; err != nil {
		t.Fatal(err)
	}

	got, err := ClaimPlanConfirmExpire(ctx, db, now)
	if err != nil || got == nil {
		t.Fatalf("ClaimPlanConfirmExpire() = %v, %v; want claimed record", got, err)
	}
	if got.BatchPk != pending.BatchPk {
		t.Fatalf("claimed batch = %s, want %s", got.BatchPk, pending.BatchPk)
	}
	second, err := ClaimPlanConfirmExpire(ctx, db, now)
	if err != nil {
		t.Fatal(err)
	}
	if second != nil {
		t.Fatalf("ClaimPlanConfirmExpire() second = %v, want nil", second)
	}
}

func TestClaimPlanConfirmSkipsPausedBatch(t *testing.T) {
	db := openPlanTestDB(t)
	ctx := context.Background()
	now := time.Now()
	plan := Plan{PlanId: "plan", Status: model.PlanStatusEnabled}
	plan.Id = "plan-pk"
	batch := PlanBatch{PlanPk: plan.Id, PlanId: plan.PlanId, BatchId: "batch", Status: model.PlanStatusPaused}
	batch.Id = "batch-pk"
	if err := db.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&batch).Error; err != nil {
		t.Fatal(err)
	}
	confirm := PlanConfirm{
		PlanPk:      plan.Id,
		BatchPk:     batch.Id,
		ConfirmType: model.ConfirmTypeBatch,
		TargetPk:    batch.Id,
		NotifyAt:    now.Add(-time.Hour),
		Deadline:    now.Add(-time.Minute),
	}
	if err := db.Create(&confirm).Error; err != nil {
		t.Fatal(err)
	}

	if got, err := ClaimPlanConfirmNotify(ctx, db, now); err != nil || got != nil {
		t.Fatalf("ClaimPlanConfirmNotify() = %v, %v; want nil, nil", got, err)
	}
	if got, err := ClaimPlanConfirmExpire(ctx, db, now); err != nil || got != nil {
		t.Fatalf("ClaimPlanConfirmExpire() = %v, %v; want nil, nil", got, err)
	}
}

func TestTerminateExecItemsByBatch(t *testing.T) {
	db := openPlanTestDB(t)
	ctx := context.Background()
	_, batch := createPlanHierarchy(t, db, []int{
		model.StatusWaiting,
		model.StatusRunning,
		model.StatusCompleted,
		model.StatusTerminated,
	})

	rows, err := TerminateExecItemsByBatch(ctx, db, batch.Id, "expired", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if rows != 2 {
		t.Fatalf("terminated rows = %d, want 2", rows)
	}

	var items []PlanExecItem
	if err := db.Order("id").Find(&items).Error; err != nil {
		t.Fatal(err)
	}
	want := []int{
		model.StatusTerminated,
		model.StatusTerminated,
		model.StatusCompleted,
		model.StatusTerminated,
	}
	for i := range items {
		if items[i].Status != want[i] {
			t.Fatalf("item %d status = %d, want %d", i, items[i].Status, want[i])
		}
	}
	if items[0].TriggerCount != 1 {
		t.Fatalf("terminated item trigger_count = %d, want 1", items[0].TriggerCount)
	}
}

func TestBatchConfirmDeadlineUsesEndOfDay(t *testing.T) {
	triggerTime := time.Date(2026, 11, 9, 9, 30, 0, 0, time.Local)
	deadline := BatchConfirmDeadline(triggerTime)
	want := time.Date(2026, 11, 9, 23, 59, 59, 0, deadline.Location())
	if deadline.Year() != want.Year() || deadline.Month() != want.Month() || deadline.Day() != want.Day() {
		t.Fatalf("deadline date = %s, want %s", deadline, want)
	}
	if deadline.Hour() != 23 || deadline.Minute() != 59 || deadline.Second() != 59 {
		t.Fatalf("deadline time = %s, want end of day", deadline)
	}
}
