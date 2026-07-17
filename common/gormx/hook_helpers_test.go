package gormx

import "testing"

func TestSkipHooksCreateDoesNotWriteAuditFields(t *testing.T) {
	db := openTestDB(t, &uintAuditTestModel{})
	ctx := WithUserContext(t.Context(), NewUserContext(uint(50), "creator", "tenant-1"))

	record := uintAuditTestModel{Name: "created"}
	if err := SkipHooksCreate(db.WithContext(ctx), &record); err != nil {
		t.Fatalf("skip hooks create error = %v", err)
	}

	var got uintAuditTestModel
	if err := db.First(&got, record.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.Name != "created" {
		t.Fatalf("name = %q, want created", got.Name)
	}
	if got.CreateUser != 0 {
		t.Fatalf("create_user = %d, want zero", got.CreateUser)
	}
	if got.CreateName != "" {
		t.Fatalf("create_name = %q, want empty", got.CreateName)
	}
}

func TestSkipHooksUpdateDoesNotWriteAuditFields(t *testing.T) {
	db := openTestDB(t, &uintAuditTestModel{})
	createCtx := WithUserContext(t.Context(), NewUserContext(uint(100), "creator", "tenant-1"))
	model := uintAuditTestModel{Name: "old"}
	if err := db.WithContext(createCtx).Create(&model).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	updateCtx := WithUserContext(t.Context(), NewUserContext(uint(200), "updater", "tenant-1"))
	if err := SkipHooksUpdate(db.WithContext(updateCtx), &uintAuditTestModel{ID: model.ID}, map[string]any{"name": "new"}); err != nil {
		t.Fatalf("skip hooks update error = %v", err)
	}

	var got uintAuditTestModel
	if err := db.First(&got, model.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.Name != "new" {
		t.Fatalf("name = %q, want new", got.Name)
	}
	if got.UpdateUser != 0 {
		t.Fatalf("update_user = %d, want zero", got.UpdateUser)
	}
	if got.UpdateName != "" {
		t.Fatalf("update_name = %q, want empty", got.UpdateName)
	}
}
