package cron

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/common/gormx"
	"zero-service/common/limitx"
	"zero-service/model"

	"github.com/alicebob/miniredis/v2"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newCronTestService(t *testing.T) *CronService {
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
	if err := db.AutoMigrate(&gormmodel.PlanConfirm{}); err != nil {
		t.Fatal(err)
	}
	redisServer := miniredis.RunT(t)
	return NewCronService(&svc.ServiceContext{
		DB:    &gormx.DB{DB: db},
		Redis: redis.New(redisServer.Addr()),
	})
}

func TestAllowBatchDispatchOnlyLimitsLateConfirmedBatches(t *testing.T) {
	s := newCronTestService(t)
	ctx := context.Background()
	triggerTime := time.Now().Add(-2 * time.Hour)

	plain := &gormmodel.PlanBatch{PlanTriggerTime: sql.NullTime{Time: triggerTime, Valid: true}}
	plain.Id = "batch-plain"
	for i := 0; i < planBatchDispatchBurst*3; i++ {
		if d := s.allowBatchDispatch(ctx, nil, plain); !d.allowed {
			t.Fatalf("plain batch Allow() #%d = denied, want allowed", i+1)
		}
	}

	// 提前确认：确认时间早于批次触发时间，不限速。
	early := &gormmodel.PlanBatch{PlanTriggerTime: sql.NullTime{Time: triggerTime, Valid: true}}
	early.Id = "batch-early"
	createConfirmedRecord(t, s, early.Id, triggerTime.Add(-time.Minute))
	for i := 0; i < planBatchDispatchBurst*3; i++ {
		if d := s.allowBatchDispatch(ctx, nil, early); !d.allowed {
			t.Fatalf("early confirmed batch Allow() #%d = denied, want allowed", i+1)
		}
	}

	// 晚确认：确认时间早于当前时间，但晚于批次触发时间，仍然限速。
	late := &gormmodel.PlanBatch{PlanTriggerTime: sql.NullTime{Time: triggerTime, Valid: true}}
	late.Id = "batch-late"
	createConfirmedRecord(t, s, late.Id, time.Now().Add(-time.Hour))
	// 用 1/1 的令牌桶，避免断言依赖测试执行速度导致令牌被时间补回。
	s.batchLimiter = limitx.NewTokenBucket(
		s.svcCtx.Redis,
		limitx.WithRate(1),
		limitx.WithBurst(1),
		limitx.WithKeyPrefix("test:"),
	)
	if d := s.allowBatchDispatch(ctx, nil, late); !d.allowed {
		t.Fatal("late confirmed batch first Allow() = denied, want allowed within burst")
	}
	d := s.allowBatchDispatch(ctx, nil, late)
	if d.allowed {
		t.Fatal("late confirmed batch Allow() = allowed, want denied after burst")
	}
	if d.wait <= 0 {
		t.Fatalf("late confirmed batch wait = %v, want positive", d.wait)
	}
	// 限速日志依赖这两个时间：命中限速时必须能带出「确认时间」与「批次触发时间」。
	if !d.confirmedAt.After(d.planTriggerAt) {
		t.Fatalf("decision confirmedAt = %v, planTriggerAt = %v; want confirmed after trigger", d.confirmedAt, d.planTriggerAt)
	}
}

// 批次缺少 plan_trigger_time 时无法判断是否“晚确认”，应保持原有下发行为，不被限速误拦。
func TestAllowBatchDispatchSkipsBatchWithoutTriggerTime(t *testing.T) {
	s := newCronTestService(t)
	ctx := context.Background()

	batch := &gormmodel.PlanBatch{}
	batch.Id = "batch-no-trigger"
	createConfirmedRecord(t, s, batch.Id, time.Now())
	s.batchLimiter = limitx.NewTokenBucket(
		s.svcCtx.Redis,
		limitx.WithRate(1),
		limitx.WithBurst(1),
		limitx.WithKeyPrefix("no-trigger:"),
	)

	for i := 0; i < 5; i++ {
		if d := s.allowBatchDispatch(ctx, nil, batch); !d.allowed {
			t.Fatalf("batch without plan_trigger_time Allow() #%d = denied, want allowed", i+1)
		}
	}
}

func createConfirmedRecord(t *testing.T, s *CronService, batchPk string, confirmedAt time.Time) {
	t.Helper()
	record := gormmodel.PlanConfirm{
		PlanPk:        "plan-pk",
		BatchPk:       batchPk,
		ConfirmType:   model.ConfirmTypeBatch,
		TargetPk:      batchPk,
		NotifyAt:      confirmedAt.Add(-time.Hour),
		Deadline:      confirmedAt.Add(time.Hour),
		ConfirmedTime: sql.NullTime{Time: confirmedAt, Valid: true},
		ConfirmStatus: model.PlanConfirmStatusConfirmed,
	}
	if err := s.svcCtx.DB.WithContext(context.Background()).Create(&record).Error; err != nil {
		t.Fatal(err)
	}
}
