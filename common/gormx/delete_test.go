package gormx

import (
	"context"
	"testing"
)

func TestUnscopedDeleteHardDeletesRecord(t *testing.T) {
	db := openTestDB(t, &legacyDeleteTestModel{})

	record := legacyDeleteTestModel{Name: "victim"}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	if err := UnscopedDelete(db, &legacyDeleteTestModel{LegacyStringBaseModel: LegacyStringBaseModel{LegacyStringIDMixin: LegacyStringIDMixin{Id: record.Id}}}); err != nil {
		t.Fatalf("unscoped delete error = %v", err)
	}

	var count int64
	if err := db.Unscoped().Model(&legacyDeleteTestModel{}).Where("id = ?", record.Id).Count(&count).Error; err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0 after hard delete", count)
	}
}

func TestUnscopedDeleteWithTenantHardDeletesRecord(t *testing.T) {
	db := openTestDB(t, &batchTenantTestModel{})
	ctx := WithTenantContext(context.Background(), "tenant-1")

	record := batchTenantTestModel{TenantMixin: TenantMixin{TenantID: "tenant-1"}, Name: "test"}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	if err := UnscopedDeleteWithTenant(db.WithContext(ctx), &batchTenantTestModel{}, "id = ?", record.ID); err != nil {
		t.Fatalf("unscoped delete error = %v", err)
	}

	var count int64
	if err := db.Unscoped().Model(&batchTenantTestModel{}).Count(&count).Error; err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
}

func TestUnscopedDeleteWithTenantDoesNotAffectOtherTenant(t *testing.T) {
	db := openTestDB(t, &batchTenantTestModel{})
	ctx1 := WithTenantContext(context.Background(), "tenant-1")
	ctx2 := WithTenantContext(context.Background(), "tenant-2")

	record1 := batchTenantTestModel{TenantMixin: TenantMixin{TenantID: "tenant-1"}, Name: "keep"}
	if err := db.WithContext(ctx1).Create(&record1).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	if err := UnscopedDeleteWithTenant(db.WithContext(ctx2), &batchTenantTestModel{}, "id = ?", record1.ID); err != nil {
		t.Fatalf("unscoped delete error = %v", err)
	}

	var count int64
	if err := db.Model(&batchTenantTestModel{}).Where("id = ?", record1.ID).Count(&count).Error; err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1 (other tenant should not delete)", count)
	}
}
