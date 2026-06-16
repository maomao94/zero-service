package gormx

import "testing"

func TestSkipHooksUpdateUpdatesWithoutCallbacks(t *testing.T) {
	db := openTestDB(t, &uintAuditTestModel{})
	ctx := WithUserContext(t.Context(), NewUserContext(uint(100), "creator", "tenant-1"))
	model := uintAuditTestModel{Name: "old"}
	if err := db.WithContext(ctx).Create(&model).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	if err := SkipHooksUpdate(db, &uintAuditTestModel{ID: model.ID}, map[string]any{"name": "new"}); err != nil {
		t.Fatalf("skip hooks update error = %v", err)
	}

	var got uintAuditTestModel
	if err := db.First(&got, model.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.Name != "new" {
		t.Fatalf("name = %q, want new", got.Name)
	}
	if got.UpdateUser != 100 {
		t.Fatalf("update user should keep create callback value when update hooks are skipped")
	}
}
