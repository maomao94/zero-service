package gormx

import (
	"context"
	"testing"

	"gorm.io/gorm"
)

type tenantHookModel struct {
	ID uint `gorm:"primarykey"`
	TenantMixin
	AuditMixin
	VersionMixin
	SoftDeleteMixin
	Name string `gorm:"column:name"`
}

func (m *tenantHookModel) BeforeCreate(tx *gorm.DB) error {
	m.BeforeCreateTenant(GetUserContext(tx.Statement.Context))
	return nil
}

func TestTenantMixinAutoFillsWithModelHook(t *testing.T) {
	db := openTestDB(t, &tenantHookModel{})
	ctx := WithUserAndTenantContext(context.Background(), uint(7), "tester", "tenant-b")
	record := tenantHookModel{Name: "model"}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	var got tenantHookModel
	if err := db.First(&got, "id = ?", record.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.TenantID != "tenant-b" {
		t.Fatalf("tenant_id = %q, want tenant-b", got.TenantID)
	}
}
