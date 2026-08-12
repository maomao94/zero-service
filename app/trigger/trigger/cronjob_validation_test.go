package trigger

import (
	"strings"
	"testing"
)

func TestCronJobTaskCode128RunesBoundary(t *testing.T) {
	code128 := strings.Repeat("a", 128)
	code129 := strings.Repeat("a", 129)

	create := &CreateCronJobReq{
		TaskCode: code128,
		TaskName: "任务",
		Type:     "inspection",
		Rule:     &PlanRulePb{Freq: 3, Hours: []int32{9}, Minutes: []int32{30}},
		DeptCode: "D001",
	}
	if err := create.Validate(); err != nil {
		t.Fatalf("CreateCronJobReq should accept 128-rune task code: %v", err)
	}
	create.TaskCode = code129
	if err := create.Validate(); err == nil {
		t.Fatal("CreateCronJobReq should reject 129-rune task code")
	}

	submit := &SubmitCronJobReq{
		TaskCode: code128,
		TaskName: "任务",
		Rule:     &PlanRulePb{Freq: 3, Hours: []int32{9}, Minutes: []int32{30}},
	}
	if err := submit.Validate(); err != nil {
		t.Fatalf("SubmitCronJobReq should accept 128-rune task code: %v", err)
	}
	submit.TaskCode = code129
	if err := submit.Validate(); err == nil {
		t.Fatal("SubmitCronJobReq should reject 129-rune task code")
	}

	list := &ListCronJobsReq{TaskCode: code128}
	if err := list.Validate(); err != nil {
		t.Fatalf("ListCronJobsReq should accept 128-rune task code: %v", err)
	}
	list.TaskCode = code129
	if err := list.Validate(); err == nil {
		t.Fatal("ListCronJobsReq should reject 129-rune task code")
	}
}
