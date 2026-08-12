package logic

import (
	"context"
	"strings"
	"testing"
	"time"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/gormx"
	"zero-service/common/tool"

	"github.com/alicebob/miniredis/v2"
	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/mathx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreatePlanTaskUsesCalculatedEmptyScheduleWhenSkippingTimeFilter(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&parseTime=true&_loc=auto"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&gormmodel.Plan{}); err != nil {
		t.Fatal(err)
	}

	request := &trigger.CreatePlanTaskReq{
		DeptCode:       "dept",
		PlanId:         "empty-schedule",
		PlanName:       "empty schedule",
		Type:           "test",
		StartTime:      "2099-07-27 00:00:00",
		EndTime:        "2099-07-27 23:59:59",
		Rule:           &trigger.PlanRulePb{Freq: 3, Hours: []int32{9}, Minutes: []int32{30}},
		ExcludeDates:   []string{"2099-07-27"},
		ExecItems:      []*trigger.CreatePlanExecItemPb{{ItemId: "item"}},
		SkipTimeFilter: true,
	}
	calculated, err := NewCalcPlanTaskDateLogic(context.Background(), nil).CalcPlanTaskDate(&trigger.CalcPlanTaskDateReq{
		StartTime:    request.StartTime,
		EndTime:      request.EndTime,
		Rule:         request.Rule,
		ExcludeDates: request.ExcludeDates,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(calculated.PlanDates) != 0 {
		t.Fatalf("calculated dates = %v, want empty", calculated.PlanDates)
	}

	logic := NewCreatePlanTaskLogic(context.Background(), &svc.ServiceContext{DB: &gormx.DB{DB: db}})
	if _, err := logic.CreatePlanTask(request); err != nil {
		t.Fatal(err)
	}
	var plan gormmodel.Plan
	if err := db.Where("plan_id = ?", request.PlanId).First(&plan).Error; err != nil {
		t.Fatal(err)
	}
	if plan.RRuleStr != calculated.GetRruleStr() {
		t.Fatalf("persisted rrule = %q, want calculated rrule %q", plan.RRuleStr, calculated.GetRruleStr())
	}
}

func TestCreatePlanTaskExpandsSpecifiedTimesAndPreservesCompiledSet(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&parseTime=true&_loc=auto"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&gormmodel.Plan{}, &gormmodel.PlanBatch{}, &gormmodel.PlanExecItem{}); err != nil {
		t.Fatal(err)
	}
	redisServer := miniredis.RunT(t)
	serviceCtx := &svc.ServiceContext{
		DB:             &gormx.DB{DB: db},
		IdUtil:         tool.NewIdUtil(redis.New(redisServer.Addr())),
		UnstableExpiry: mathx.NewUnstable(0),
	}
	request := &trigger.CreatePlanTaskReq{
		DeptCode:       "dept",
		PlanId:         "rdate-plan",
		PlanName:       "rdate plan",
		Type:           "test",
		StartTime:      "2099-07-01 00:00:00",
		EndTime:        "2099-07-02 23:59:59",
		Rule:           &trigger.PlanRulePb{Freq: 3, Hours: []int32{9}, Minutes: []int32{0}},
		SpecifiedTimes: []string{"2099-07-01 12:34:56", "2099-07-01 12:34:56"},
		ExcludedTimes:  []string{"2099-07-02 09:00:00"},
		ExecItems:      []*trigger.CreatePlanExecItemPb{{ItemId: "item"}},
		SkipTimeFilter: true,
	}
	calculated, err := NewCalcPlanTaskDateLogic(context.Background(), serviceCtx).CalcPlanTaskDate(&trigger.CalcPlanTaskDateReq{
		StartTime:      request.StartTime,
		EndTime:        request.EndTime,
		Rule:           request.Rule,
		SpecifiedTimes: request.SpecifiedTimes,
		ExcludedTimes:  request.ExcludedTimes,
	})
	if err != nil {
		t.Fatal(err)
	}
	logic := NewCreatePlanTaskLogic(context.Background(), serviceCtx)
	result, err := logic.CreatePlanTask(request)
	if err != nil {
		t.Fatal(err)
	}
	if result.BatchCnt != 2 || result.ExecCnt != 2 {
		t.Fatalf("result counts = batch %d, exec %d; want 2, 2", result.BatchCnt, result.ExecCnt)
	}
	var plan gormmodel.Plan
	if err := db.Where("plan_id = ?", request.PlanId).First(&plan).Error; err != nil {
		t.Fatal(err)
	}
	if plan.RRuleStr != calculated.RruleStr {
		t.Fatalf("persisted rrule = %q, want %q", plan.RRuleStr, calculated.RruleStr)
	}
	for _, want := range []string{
		"RDATE;TZID=Asia/Shanghai:20990701T123456",
		"EXDATE;TZID=Asia/Shanghai:20990702T090000",
	} {
		if !strings.Contains(plan.RRuleStr, want) {
			t.Fatalf("persisted rrule = %q, want substring %q", plan.RRuleStr, want)
		}
	}
	var batches []gormmodel.PlanBatch
	if err := db.Where("plan_pk = ?", plan.Id).Order("plan_trigger_time").Find(&batches).Error; err != nil {
		t.Fatal(err)
	}
	if len(batches) != 2 {
		t.Fatalf("batches = %d, want 2", len(batches))
	}
	location, err := time.LoadLocation(carbon.Shanghai)
	if err != nil {
		t.Fatal(err)
	}
	wantTimes := []time.Time{
		time.Date(2099, 7, 1, 9, 0, 0, 0, location),
		time.Date(2099, 7, 1, 12, 34, 56, 0, location),
	}
	for i := range wantTimes {
		if !batches[i].PlanTriggerTime.Time.Equal(wantTimes[i]) {
			t.Fatalf("stored batch time[%d] = %s, want %s", i, batches[i].PlanTriggerTime.Time, wantTimes[i])
		}
	}
	var items []gormmodel.PlanExecItem
	if err := db.Where("plan_pk = ?", plan.Id).Order("plan_trigger_time").Find(&items).Error; err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("exec items = %d, want 2", len(items))
	}
	for i := range items {
		if !items[i].PlanTriggerTime.Equal(wantTimes[i]) || !items[i].NextTriggerTime.Equal(wantTimes[i]) {
			t.Fatalf("exec item[%d] times = plan %s, next %s; want %s", i, items[i].PlanTriggerTime, items[i].NextTriggerTime, wantTimes[i])
		}
	}
}

func TestCreatePlanTaskFiltersPastSpecifiedTimeUnlessSkipped(t *testing.T) {
	newRequest := func(planID string, skip bool) *trigger.CreatePlanTaskReq {
		return &trigger.CreatePlanTaskReq{
			DeptCode:       "dept",
			PlanId:         planID,
			PlanName:       "past rdate",
			Type:           "test",
			StartTime:      "2025-07-01 00:00:00",
			EndTime:        "2025-07-01 23:59:59",
			Rule:           &trigger.PlanRulePb{Freq: 3, Hours: []int32{9}, Minutes: []int32{0}},
			SpecifiedTimes: []string{"2025-07-01 12:34:56"},
			ExcludedTimes:  []string{"2025-07-01 09:00:00"},
			ExecItems:      []*trigger.CreatePlanExecItemPb{{ItemId: "item"}},
			SkipTimeFilter: skip,
		}
	}

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&parseTime=true&_loc=auto"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&gormmodel.Plan{}, &gormmodel.PlanBatch{}, &gormmodel.PlanExecItem{}); err != nil {
		t.Fatal(err)
	}
	redisServer := miniredis.RunT(t)
	serviceCtx := &svc.ServiceContext{
		DB:             &gormx.DB{DB: db},
		IdUtil:         tool.NewIdUtil(redis.New(redisServer.Addr())),
		UnstableExpiry: mathx.NewUnstable(0),
	}
	logic := NewCreatePlanTaskLogic(context.Background(), serviceCtx)

	if _, err := logic.CreatePlanTask(newRequest("past-filtered", false)); err == nil || !strings.Contains(err.Error(), "没有触发时间") {
		t.Fatalf("skip_time_filter=false error = %v, want no trigger time", err)
	}
	result, err := logic.CreatePlanTask(newRequest("past-preserved", true))
	if err != nil {
		t.Fatal(err)
	}
	if result.BatchCnt != 1 || result.ExecCnt != 1 {
		t.Fatalf("skip_time_filter=true counts = batch %d, exec %d; want 1, 1", result.BatchCnt, result.ExecCnt)
	}
}

func TestCreatePlanTaskKeepsExpandedItemLimit(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&parseTime=true&_loc=auto"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&gormmodel.Plan{}); err != nil {
		t.Fatal(err)
	}
	items := make([]*trigger.CreatePlanExecItemPb, 5)
	for i := range items {
		items[i] = &trigger.CreatePlanExecItemPb{ItemId: "item"}
	}
	request := &trigger.CreatePlanTaskReq{
		DeptCode:       "dept",
		PlanId:         "too-many-items",
		PlanName:       "too many items",
		Type:           "test",
		StartTime:      "2027-01-01 00:00:00",
		EndTime:        "2029-12-31 23:59:59",
		Rule:           &trigger.PlanRulePb{Freq: 3, Hours: []int32{9}, Minutes: []int32{0}},
		ExecItems:      items,
		SkipTimeFilter: true,
	}
	logic := NewCreatePlanTaskLogic(context.Background(), &svc.ServiceContext{DB: &gormx.DB{DB: db}})
	if _, err := logic.CreatePlanTask(request); err == nil || !strings.Contains(err.Error(), "调度项过多") {
		t.Fatalf("expanded item limit error = %v, want too many schedule items", err)
	}
}
