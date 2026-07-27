package logic

import (
	"context"
	"testing"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/gormx"

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
