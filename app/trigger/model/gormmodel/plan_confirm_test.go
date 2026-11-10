package gormmodel

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"zero-service/common/gormx"
	"zero-service/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
					confirm.ConfirmStatus = model.PlanConfirmStatusConfirmed
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

func TestFindPendingPlanConfirmNotifyIsIdempotent(t *testing.T) {
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

	got, err := FindPendingPlanConfirmNotify(ctx, db, now)
	if err != nil || got == nil {
		t.Fatalf("FindPendingPlanConfirmNotify() = %v, %v; want pending record", got, err)
	}
	if err := MarkPlanConfirmNotified(ctx, db, got.Id, now); err != nil {
		t.Fatal(err)
	}
	second, err := FindPendingPlanConfirmNotify(ctx, db, now)
	if err != nil {
		t.Fatal(err)
	}
	if second != nil {
		t.Fatalf("FindPendingPlanConfirmNotify() second = %v, want nil", second)
	}
}

func TestFindPendingPlanConfirmNotifySkipsExpiredDeadline(t *testing.T) {
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
		NotifyAt:    now.Add(-time.Hour),
		Deadline:    now.Add(-time.Minute),
	}
	if err := db.Create(&confirm).Error; err != nil {
		t.Fatal(err)
	}

	got, err := FindPendingPlanConfirmNotify(ctx, db, now)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("FindPendingPlanConfirmNotify() = %v, want nil for expired deadline", got)
	}
}

func TestFindPendingPlanConfirmExpireSkipsConfirmed(t *testing.T) {
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
		ConfirmStatus: model.PlanConfirmStatusConfirmed,
		ConfirmedTime: sql.NullTime{Time: now.Add(-time.Minute), Valid: true},
	}
	if err := db.Create(&pending).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&confirmed).Error; err != nil {
		t.Fatal(err)
	}

	got, err := FindPendingPlanConfirmExpire(ctx, db, now)
	if err != nil || got == nil {
		t.Fatalf("FindPendingPlanConfirmExpire() = %v, %v; want pending record", got, err)
	}
	if got.BatchPk != pending.BatchPk {
		t.Fatalf("pending batch = %s, want %s", got.BatchPk, pending.BatchPk)
	}
	if _, err := ClosePlanConfirmsByBatch(ctx, db, got.BatchPk, "system", "expired", now); err != nil {
		t.Fatal(err)
	}
	second, err := FindPendingPlanConfirmExpire(ctx, db, now)
	if err != nil {
		t.Fatal(err)
	}
	if second != nil {
		t.Fatalf("FindPendingPlanConfirmExpire() second = %v, want nil", second)
	}
}

func TestFindPlanConfirmSkipsPausedBatch(t *testing.T) {
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

	if got, err := FindPendingPlanConfirmNotify(ctx, db, now); err != nil || got != nil {
		t.Fatalf("FindPendingPlanConfirmNotify() = %v, %v; want nil, nil", got, err)
	}
	if got, err := FindPendingPlanConfirmExpire(ctx, db, now); err != nil || got != nil {
		t.Fatalf("FindPendingPlanConfirmExpire() = %v, %v; want nil, nil", got, err)
	}
}

func TestBatchConfirmDeadlineUsesEndOfDay(t *testing.T) {
	triggerTime := time.Date(2026, 11, 9, 9, 30, 0, 0, time.Local)
	deadline := BatchConfirmDeadline(triggerTime, 0)
	want := time.Date(2026, 11, 9, 23, 59, 59, 0, deadline.Location())
	if deadline.Year() != want.Year() || deadline.Month() != want.Month() || deadline.Day() != want.Day() {
		t.Fatalf("deadline date = %s, want %s", deadline, want)
	}
	if deadline.Hour() != 23 || deadline.Minute() != 59 || deadline.Second() != 59 {
		t.Fatalf("deadline time = %s, want end of day", deadline)
	}
}

func TestBatchConfirmDeadlineSupportsCrossDay(t *testing.T) {
	triggerTime := time.Date(2026, 11, 9, 23, 0, 0, 0, time.Local)
	deadline := BatchConfirmDeadline(triggerTime, 180)
	if deadline.Day() != 10 || deadline.Hour() != 2 || deadline.Minute() != 0 {
		t.Fatalf("deadline = %s, want 2026-11-10 02:00:00", deadline)
	}
}

func TestPlanConfirmScanSilencesSuccessfulSQLLog(t *testing.T) {
	var buf bytes.Buffer
	logx.SetWriter(logx.NewWriter(&buf))
	defer logx.Reset()

	db := openPlanTestDB(t).Session(&gorm.Session{Logger: gormx.NewGormLogger(gormx.LoggerConfig{
		LogLevel:      logger.Info,
		SlowThreshold: time.Hour,
	})})
	now := time.Now()

	if _, err := FindPendingPlanConfirmNotify(context.Background(), db, now); err != nil {
		t.Fatal(err)
	}
	if _, err := FindPendingPlanConfirmExpire(context.Background(), db, now); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); strings.Contains(got, "plan_confirm AS pc") {
		t.Fatalf("scan query should be silent at info level, got: %s", got)
	}

	// 对照：普通查询仍应记录 info SQL，确认静默只作用于扫描查询。
	var count int64
	if err := db.Table("plan_confirm").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); !strings.Contains(got, "plan_confirm") {
		t.Fatalf("normal query should still log info SQL, got: %s", got)
	}
}

func TestClosePlanConfirmsByBatchClosesAllTypes(t *testing.T) {
	db := openPlanTestDB(t)
	plan, batch := createPlanHierarchy(t, db, nil)
	now := time.Now()
	for _, confirmType := range []int{model.ConfirmTypeBatch, model.ConfirmTypeExecItem} {
		confirm := PlanConfirm{
			PlanPk:      plan.Id,
			BatchPk:     batch.Id,
			ConfirmType: confirmType,
			TargetPk:    fmt.Sprintf("%s-%d", batch.Id, confirmType),
			NotifyAt:    now,
			Deadline:    now.Add(time.Hour),
		}
		if err := db.Create(&confirm).Error; err != nil {
			t.Fatal(err)
		}
	}

	rows, err := ClosePlanConfirmsByBatch(context.Background(), db, batch.Id, "tester", "batch terminated", now)
	if err != nil {
		t.Fatal(err)
	}
	if rows != 2 {
		t.Fatalf("closed rows = %d, want 2 (类型无关)", rows)
	}
	var rowsInDB []PlanConfirm
	if err := db.Find(&rowsInDB).Error; err != nil {
		t.Fatal(err)
	}
	for _, row := range rowsInDB {
		if row.ConfirmStatus != model.PlanConfirmStatusConfirmed {
			t.Fatalf("confirm type %d status = %d, want confirmed", row.ConfirmType, row.ConfirmStatus)
		}
		if row.ConfirmedUser.String != "tester" {
			t.Fatalf("confirm type %d user = %q, want tester", row.ConfirmType, row.ConfirmedUser.String)
		}
	}
}
