package gormx

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type legacyDeleteTestModel struct {
	LegacyBaseModel
	Name string `gorm:"column:name;uniqueIndex"`
}

func (legacyDeleteTestModel) TableName() string {
	return "legacy_delete_test_models"
}

type stringAuditTestModel struct {
	ID uint `gorm:"primarykey"`
	StringAuditMixin
	Name string `gorm:"column:name"`
}

func (stringAuditTestModel) TableName() string {
	return "string_audit_test_models"
}

type uintAuditTestModel struct {
	ID uint `gorm:"primarykey"`
	AuditMixin
	Name string `gorm:"column:name"`
}

func (uintAuditTestModel) TableName() string {
	return "uint_audit_test_models"
}

type tenantStringIDTestModel struct {
	TenantStringIDModel
	Name string `gorm:"column:name"`
}

func (tenantStringIDTestModel) TableName() string {
	return "tenant_string_id_test_models"
}

type pageTestModel struct {
	ID   uint `gorm:"primarykey"`
	Name string
}

func openTestDB(t *testing.T, models ...any) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&_loc=auto&parseTime=true"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db error = %v", err)
	}
	RegisterCallbacks(db)
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("auto migrate error = %v", err)
	}
	return db
}

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

func TestGenericUserContextFillsStringAuditFields(t *testing.T) {
	db := openTestDB(t, &stringAuditTestModel{})
	ctx := WithUserAndTenantContext(context.Background(), "8d98c7f4-0d90-4b91-a8e8-768d34da1d6a", "tester", "tenant-a")

	record := stringAuditTestModel{Name: "audit"}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	var got stringAuditTestModel
	if err := db.First(&got, record.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.CreateUser != "8d98c7f4-0d90-4b91-a8e8-768d34da1d6a" {
		t.Fatalf("create_user = %q, want uuid", got.CreateUser)
	}
	if got.UpdateUser != "8d98c7f4-0d90-4b91-a8e8-768d34da1d6a" {
		t.Fatalf("update_user = %q, want uuid", got.UpdateUser)
	}
	if got.CreateName != "tester" || got.UpdateName != "tester" {
		t.Fatalf("names = %q/%q, want tester/tester", got.CreateName, got.UpdateName)
	}
}

func TestGenericUserContextFillsUintAuditFields(t *testing.T) {
	db := openTestDB(t, &uintAuditTestModel{})
	ctx := WithUserAndTenantContext(context.Background(), uint(42), "tester", "tenant-a")

	record := uintAuditTestModel{Name: "audit"}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	var got uintAuditTestModel
	if err := db.First(&got, record.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.CreateUser != 42 || got.UpdateUser != 42 {
		t.Fatalf("users = %d/%d, want 42/42", got.CreateUser, got.UpdateUser)
	}
	id, ok := GetUserIDAs[uint](ctx)
	if !ok || id != 42 {
		t.Fatalf("GetUserIDAs = %v/%v, want 42/true", id, ok)
	}
}

func TestTenantStringIDModelWorksWithGormDefaultMigrate(t *testing.T) {
	type tenantDefaultMigrateTestModel struct {
		ID uint `gorm:"primarykey"`
		TenantMixin
		AuditMixin
		VersionMixin
		SoftDeleteMixin
		Name string `gorm:"column:name"`
	}

	db := openTestDB(t, &tenantDefaultMigrateTestModel{})
	ctx := WithUserAndTenantContext(context.Background(), uint(7), "tester", "tenant-b")
	record := tenantDefaultMigrateTestModel{Name: "model"}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	var got tenantDefaultMigrateTestModel
	if err := db.First(&got, "id = ?", record.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.TenantID != "tenant-b" {
		t.Fatalf("tenant_id = %q, want tenant-b", got.TenantID)
	}
}

func TestQueryPageNormalizesInvalidParams(t *testing.T) {
	db := openTestDB(t, &pageTestModel{})
	if err := db.Create(&pageTestModel{Name: "a"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	var list []pageTestModel
	page, err := QueryPage(db.Model(&pageTestModel{}).Order("id ASC"), 0, 0, &list)
	if err != nil {
		t.Fatalf("query page error = %v", err)
	}
	if page.Page != DefaultPage || page.PageSize != DefaultPageSize {
		t.Fatalf("page params = %d/%d, want %d/%d", page.Page, page.PageSize, DefaultPage, DefaultPageSize)
	}
	if page.Total != 1 || len(page.Data) != 1 {
		t.Fatalf("page data = total %d len %d, want total 1 len 1", page.Total, len(page.Data))
	}
}
