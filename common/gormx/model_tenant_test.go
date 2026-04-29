package gormx

import (
	"context"
	"testing"
)

func TestTenantModelWorksWithGormDefaultMigrate(t *testing.T) {
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
