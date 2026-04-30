package gormx

import "testing"

func TestLegacySoftDeleteSetsDeleteTimeAndDelState(t *testing.T) {
	db := openTestDB(t, &legacyDeleteTestModel{})
	record := legacyDeleteTestModel{Name: "legacy"}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	if err := SoftDelete(db, &legacyDeleteTestModel{}, "id = ?", record.Id); err != nil {
		t.Fatalf("soft delete error = %v", err)
	}

	var visible int64
	if err := db.Model(&legacyDeleteTestModel{}).Where("id = ?", record.Id).Count(&visible).Error; err != nil {
		t.Fatalf("count visible error = %v", err)
	}
	if visible != 0 {
		t.Fatalf("visible count = %d, want 0", visible)
	}

	var got legacyDeleteTestModel
	if err := db.Unscoped().Select("id", "delete_time", "del_state").First(&got, record.Id).Error; err != nil {
		t.Fatalf("unscoped find error = %v", err)
	}
	if !got.DeleteTime.Valid {
		t.Fatalf("delete_time valid = false, want true")
	}
	if got.DelState != 1 {
		t.Fatalf("del_state = %d, want 1", got.DelState)
	}
	if !got.IsDeleted() {
		t.Fatalf("is deleted = false, want true")
	}
}

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

func TestHasLegacyDeleteFieldsDoesNotDependOnStatementModel(t *testing.T) {
	db := openTestDB(t, &legacyDeleteTestModel{})
	db.Statement.Model = nil

	if !hasLegacyDeleteFields(db, &legacyDeleteTestModel{}) {
		t.Fatalf("legacy delete fields should be detected from explicit model")
	}
}

func TestLegacyRestoreClearsDeleteTimeAndDelState(t *testing.T) {
	db := openTestDB(t, &legacyDeleteTestModel{})
	record := legacyDeleteTestModel{Name: "legacy"}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}
	if err := SoftDelete(db, &legacyDeleteTestModel{}, "id = ?", record.Id); err != nil {
		t.Fatalf("soft delete error = %v", err)
	}

	if err := Restore(db, &legacyDeleteTestModel{}, "id = ?", record.Id); err != nil {
		t.Fatalf("restore error = %v", err)
	}

	var got legacyDeleteTestModel
	if err := db.Select("id", "delete_time", "del_state").First(&got, record.Id).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.DeleteTime.Valid {
		t.Fatalf("delete_time valid = true, want false")
	}
	if got.DelState != 0 {
		t.Fatalf("del_state = %d, want 0", got.DelState)
	}
	if got.IsDeleted() {
		t.Fatalf("is deleted = true, want false")
	}
}
