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
