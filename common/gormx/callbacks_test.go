package gormx

import (
	"context"
	"testing"
)

type callbackTestModel struct {
	ID uint `gorm:"primarykey"`
	TenantMixin
	StringAuditMixin
	VersionMixin
	Name string `gorm:"column:name"`
}

type callbackSoftDeleteTestModel struct {
	ID uint `gorm:"primarykey"`
	TenantMixin
	StringAuditMixin
	VersionMixin
	SoftDeleteMixin
	Name string `gorm:"column:name"`
}

func (callbackSoftDeleteTestModel) TableName() string {
	return "callback_soft_test_models"
}

func (callbackTestModel) TableName() string {
	return "callback_test_models"
}

func TestRegisterCallbacksEnablesHooks(t *testing.T) {
	db := openTestDB(t, &callbackTestModel{})

	// Verify callbacks are registered by creating a record with user context
	ctx := WithUserAndTenantContext(context.Background(), "user-1", "tester", "tenant-1")
	record := callbackTestModel{Name: "test"}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	var got callbackTestModel
	if err := db.First(&got, record.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}

	// Verify audit fields are filled
	if got.CreateUser != "user-1" {
		t.Fatalf("create_user = %q, want user-1", got.CreateUser)
	}
	if got.UpdateUser != "user-1" {
		t.Fatalf("update_user = %q, want user-1", got.UpdateUser)
	}
	if got.CreateName != "tester" {
		t.Fatalf("create_name = %q, want tester", got.CreateName)
	}
	if got.UpdateName != "tester" {
		t.Fatalf("update_name = %q, want tester", got.UpdateName)
	}
	if got.TenantID != "tenant-1" {
		t.Fatalf("tenant_id = %q, want tenant-1", got.TenantID)
	}
}

func TestBeforeCreateHookFillsAuditFields(t *testing.T) {
	db := openTestDB(t, &callbackTestModel{})

	ctx := WithUserAndTenantContext(context.Background(), "user-1", "tester", "tenant-1")
	record := callbackTestModel{Name: "test"}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	var got callbackTestModel
	if err := db.First(&got, record.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}

	if got.CreateUser != "user-1" {
		t.Fatalf("create_user = %q, want user-1", got.CreateUser)
	}
	if got.UpdateUser != "user-1" {
		t.Fatalf("update_user = %q, want user-1", got.UpdateUser)
	}
	if got.CreateName != "tester" {
		t.Fatalf("create_name = %q, want tester", got.CreateName)
	}
	if got.UpdateName != "tester" {
		t.Fatalf("update_name = %q, want tester", got.UpdateName)
	}
	if got.TenantID != "tenant-1" {
		t.Fatalf("tenant_id = %q, want tenant-1", got.TenantID)
	}
}

func TestBeforeCreateHookSkipsWhenNoUserContext(t *testing.T) {
	db := openTestDB(t, &callbackTestModel{})

	record := callbackTestModel{Name: "test"}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	var got callbackTestModel
	if err := db.First(&got, record.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}

	// Audit fields should be empty when no user context
	if got.CreateUser != "" {
		t.Fatalf("create_user = %q, want empty", got.CreateUser)
	}
	if got.UpdateUser != "" {
		t.Fatalf("update_user = %q, want empty", got.UpdateUser)
	}
}

func TestBeforeUpdateHookFillsUpdateFields(t *testing.T) {
	db := openTestDB(t, &callbackTestModel{})

	ctx1 := WithUserAndTenantContext(context.Background(), "user-1", "creator", "tenant-1")
	record := callbackTestModel{Name: "old"}
	if err := db.WithContext(ctx1).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	ctx2 := WithUserAndTenantContext(context.Background(), "user-2", "updater", "tenant-1")
	if err := db.WithContext(ctx2).Model(&record).Update("name", "new").Error; err != nil {
		t.Fatalf("update error = %v", err)
	}

	var got callbackTestModel
	if err := db.First(&got, record.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}

	// Update fields should be filled with new user
	if got.UpdateUser != "user-2" {
		t.Fatalf("update_user = %q, want user-2", got.UpdateUser)
	}
	if got.UpdateName != "updater" {
		t.Fatalf("update_name = %q, want updater", got.UpdateName)
	}

	// Create fields should remain from original user
	if got.CreateUser != "user-1" {
		t.Fatalf("create_user = %q, want user-1", got.CreateUser)
	}
	if got.CreateName != "creator" {
		t.Fatalf("create_name = %q, want creator", got.CreateName)
	}
}

func TestBeforeUpdateHookIncrementsVersion(t *testing.T) {
	db := openTestDB(t, &callbackTestModel{})

	ctx := WithUserAndTenantContext(context.Background(), "user-1", "tester", "tenant-1")
	record := callbackTestModel{Name: "test"}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	// Update multiple times
	for i := 0; i < 3; i++ {
		if err := db.WithContext(ctx).Model(&record).Update("name", "updated").Error; err != nil {
			t.Fatalf("update error = %v", err)
		}
	}

	var got callbackTestModel
	if err := db.First(&got, record.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}

	if got.Version != 3 { // 0 (create) + 3 (updates)
		t.Fatalf("version = %d, want 3", got.Version)
	}
}

func TestBeforeDeleteHookFillsDeleteFieldsOnHardDelete(t *testing.T) {
	db := openTestDB(t, &callbackTestModel{})

	ctx := WithUserAndTenantContext(context.Background(), "user-1", "creator", "tenant-1")
	record := callbackTestModel{Name: "test"}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	ctx2 := WithUserAndTenantContext(context.Background(), "user-2", "deleter", "tenant-1")
	if err := db.WithContext(ctx2).Delete(&record).Error; err != nil {
		t.Fatalf("delete error = %v", err)
	}

	var count int64
	if err := db.Model(&callbackTestModel{}).Where("id = ?", record.ID).Count(&count).Error; err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
}

func TestBeforeUpdateHookFillsDeleteFieldsOnSoftDelete(t *testing.T) {
	db := openTestDB(t, &callbackSoftDeleteTestModel{})

	ctx := WithUserAndTenantContext(context.Background(), "user-1", "creator", "tenant-1")
	record := callbackSoftDeleteTestModel{Name: "test"}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	ctx2 := WithUserAndTenantContext(context.Background(), "user-2", "deleter", "tenant-1")
	if err := db.WithContext(ctx2).Delete(&record).Error; err != nil {
		t.Fatalf("delete error = %v", err)
	}

	var got callbackSoftDeleteTestModel
	if err := db.Unscoped().First(&got, record.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.DeletedAt.Time.IsZero() {
		t.Fatalf("record should be soft-deleted (deleted_at set)")
	}
}

func TestBeforeDeleteHookSkipsWhenNoUserContext(t *testing.T) {
	db := openTestDB(t, &callbackTestModel{})

	record := callbackTestModel{Name: "test"}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	if err := db.Delete(&record).Error; err != nil {
		t.Fatalf("delete error = %v", err)
	}

	var count int64
	if err := db.Model(&callbackTestModel{}).Where("id = ?", record.ID).Count(&count).Error; err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0 (record should be hard deleted)", count)
	}
}
