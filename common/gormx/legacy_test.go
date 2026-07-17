package gormx

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
)

type deleteTimeOnlyTestModel struct {
	ID         uint         `gorm:"primarykey"`
	DeleteTime sql.NullTime `gorm:"column:delete_time"`
	Name       string       `gorm:"column:name"`
}

func (deleteTimeOnlyTestModel) TableName() string {
	return "delete_time_only_test_models"
}

type delStateOnlyTestModel struct {
	ID       uint   `gorm:"primarykey"`
	DelState int64  `gorm:"column:del_state"`
	Name     string `gorm:"column:name"`
}

func (delStateOnlyTestModel) TableName() string {
	return "del_state_only_test_models"
}

type isDeletedOnlyTestModel struct {
	ID        uint   `gorm:"primarykey"`
	IsDeleted bool   `gorm:"column:is_deleted"`
	Name      string `gorm:"column:name"`
}

func (isDeletedOnlyTestModel) TableName() string {
	return "is_deleted_only_test_models"
}

func TestLegacySoftDeleteSetsDeleteTimeAndIsDeleted(t *testing.T) {
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
	if err := db.Unscoped().Select("id", "delete_time", "is_deleted").Where("id = ?", record.Id).First(&got).Error; err != nil {
		t.Fatalf("unscoped find error = %v", err)
	}
	if !got.DeleteTime.Valid {
		t.Fatalf("delete_time valid = false, want true")
	}
	if got.IsDeleted != 1 {
		t.Fatalf("is_deleted = %d, want 1", got.IsDeleted)
	}
	if !got.Deleted() {
		t.Fatalf("is deleted = false, want true")
	}
}

func TestLegacyBaseModelHooksFillCreateUpdateAuditAndTenantFields(t *testing.T) {
	db := openTestDB(t, &legacyDeleteTestModel{})
	ctx := WithUserAndTenantContext(context.Background(), uint(7), "creator", "tenant-a")
	record := legacyDeleteTestModel{Name: "legacy"}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	var got legacyDeleteTestModel
	if err := db.First(&got, "id = ?", record.Id).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.CreateUser != 7 || got.UpdateUser != 7 {
		t.Fatalf("audit users = create:%d update:%d, want 7", got.CreateUser, got.UpdateUser)
	}
	if got.CreateName != "creator" || got.UpdateName != "creator" {
		t.Fatalf("audit names = create:%q update:%q, want creator", got.CreateName, got.UpdateName)
	}
	if got.TenantID != "tenant-a" {
		t.Fatalf("tenant_id = %q, want tenant-a", got.TenantID)
	}

	updateCtx := WithUserAndTenantContext(context.Background(), uint(8), "updater", "tenant-a")
	if err := db.WithContext(updateCtx).Model(&got).Update("name", "updated").Error; err != nil {
		t.Fatalf("update error = %v", err)
	}
	var updated legacyDeleteTestModel
	if err := db.First(&updated, "id = ?", record.Id).Error; err != nil {
		t.Fatalf("find updated error = %v", err)
	}
	if updated.CreateUser != 7 || updated.CreateName != "creator" {
		t.Fatalf("create audit changed to %d/%q, want 7/creator", updated.CreateUser, updated.CreateName)
	}
	if updated.UpdateUser != 8 || updated.UpdateName != "updater" {
		t.Fatalf("update audit = %d/%q, want 8/updater", updated.UpdateUser, updated.UpdateName)
	}

	deleteCtx := WithUserAndTenantContext(context.Background(), uint(9), "deleter", "tenant-a")
	if err := db.WithContext(deleteCtx).Delete(&updated).Error; err != nil {
		t.Fatalf("delete error = %v", err)
	}
	var deleted legacyDeleteTestModel
	if err := db.Unscoped().First(&deleted, "id = ?", record.Id).Error; err != nil {
		t.Fatalf("find deleted error = %v", err)
	}
	if !deleted.Deleted() || !deleted.DeleteTime.Valid {
		t.Fatalf("deleted = %v, delete_time valid = %v, want deleted with delete_time", deleted.Deleted(), deleted.DeleteTime.Valid)
	}
}

func TestLegacyStringIDMixinGeneratesUUID(t *testing.T) {
	db := openTestDB(t, &legacyStringIDTestModel{})
	record := legacyStringIDTestModel{Name: "legacy"}

	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}
	if record.Id == "" {
		t.Fatalf("id is empty, want uuid")
	}
	parsed, err := uuid.Parse(record.Id)
	if err != nil {
		t.Fatalf("parse id error = %v", err)
	}
	if parsed.Version() != 7 {
		t.Fatalf("uuid version = %d, want 7", parsed.Version())
	}

	var got legacyStringIDTestModel
	if err := db.First(&got, "id = ?", record.Id).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.Id != record.Id {
		t.Fatalf("id = %q, want %q", got.Id, record.Id)
	}
}

func TestLegacyStringIDMixinKeepsPresetID(t *testing.T) {
	db := openTestDB(t, &legacyStringIDTestModel{})
	record := legacyStringIDTestModel{LegacyStringBaseModel: LegacyStringBaseModel{LegacyStringIDMixin: LegacyStringIDMixin{Id: "preset-id"}}, Name: "legacy"}

	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}
	if record.Id != "preset-id" {
		t.Fatalf("id = %q, want preset-id", record.Id)
	}

	var got legacyStringIDTestModel
	if err := db.First(&got, "id = ?", "preset-id").Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.Id != "preset-id" {
		t.Fatalf("stored id = %q, want preset-id", got.Id)
	}
}

func TestRestoreStandardSoftDeleteClearsDeletedAt(t *testing.T) {
	db := openTestDB(t, &batchTenantSoftDeleteTestModel{})
	record := batchTenantSoftDeleteTestModel{Name: "restore-test"}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}
	if err := db.Delete(&record).Error; err != nil {
		t.Fatalf("soft delete error = %v", err)
	}

	if err := Restore(db, &batchTenantSoftDeleteTestModel{}, "id = ?", record.ID); err != nil {
		t.Fatalf("restore error = %v", err)
	}

	var got batchTenantSoftDeleteTestModel
	if err := db.First(&got, record.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.Name != "restore-test" {
		t.Fatalf("name = %q, want restore-test", got.Name)
	}
}

func TestHasLegacyDeleteFieldsDoesNotDependOnStatementModel(t *testing.T) {
	db := openTestDB(t, &legacyDeleteTestModel{})
	db.Statement.Model = nil

	if !hasLegacyDeleteFields(db, &legacyDeleteTestModel{}) {
		t.Fatalf("legacy delete fields should be detected from explicit model")
	}
}

func TestLegacyRestoreClearsDeleteTimeAndIsDeleted(t *testing.T) {
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
	if err := db.Select("id", "delete_time", "is_deleted").Where("id = ?", record.Id).First(&got).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.DeleteTime.Valid {
		t.Fatalf("delete_time valid = true, want false")
	}
	if got.IsDeleted != 0 {
		t.Fatalf("is_deleted = %d, want 0", got.IsDeleted)
	}
	if got.Deleted() {
		t.Fatalf("is deleted = true, want false")
	}
}

func TestRestoreHandlesDeleteTimeOnlyField(t *testing.T) {
	db := openTestDB(t, &deleteTimeOnlyTestModel{})
	record := deleteTimeOnlyTestModel{Name: "legacy"}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}
	if err := db.Model(&deleteTimeOnlyTestModel{}).Where("id = ?", record.ID).Update("delete_time", sql.NullTime{Time: time.Now(), Valid: true}).Error; err != nil {
		t.Fatalf("mark deleted error = %v", err)
	}

	if err := Restore(db, &deleteTimeOnlyTestModel{}, "id = ?", record.ID); err != nil {
		t.Fatalf("restore error = %v", err)
	}

	var got deleteTimeOnlyTestModel
	if err := db.Unscoped().First(&got, record.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.DeleteTime.Valid {
		t.Fatalf("delete_time valid = true, want false")
	}
}

func TestRestoreHandlesDelStateOnlyField(t *testing.T) {
	db := openTestDB(t, &delStateOnlyTestModel{})
	record := delStateOnlyTestModel{Name: "legacy"}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}
	if err := db.Model(&delStateOnlyTestModel{}).Where("id = ?", record.ID).Update("del_state", 1).Error; err != nil {
		t.Fatalf("mark deleted error = %v", err)
	}

	if err := Restore(db, &delStateOnlyTestModel{}, "id = ?", record.ID); err != nil {
		t.Fatalf("restore error = %v", err)
	}

	var got delStateOnlyTestModel
	if err := db.First(&got, record.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.DelState != 0 {
		t.Fatalf("del_state = %d, want 0", got.DelState)
	}
}

func TestRestoreHandlesIsDeletedOnlyField(t *testing.T) {
	db := openTestDB(t, &isDeletedOnlyTestModel{})
	record := isDeletedOnlyTestModel{Name: "legacy"}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}
	if err := db.Model(&isDeletedOnlyTestModel{}).Where("id = ?", record.ID).Update("is_deleted", true).Error; err != nil {
		t.Fatalf("mark deleted error = %v", err)
	}

	if err := Restore(db, &isDeletedOnlyTestModel{}, "id = ?", record.ID); err != nil {
		t.Fatalf("restore error = %v", err)
	}

	var got isDeletedOnlyTestModel
	if err := db.First(&got, record.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.IsDeleted {
		t.Fatalf("is_deleted = true, want false")
	}
}
