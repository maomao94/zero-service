package gormx

import (
	"context"
	"testing"
)

func TestUnscopedDeleteWithTenantHardDeletesRecord(t *testing.T) {
	db := openTestDB(t, &batchTenantTestModel{})
	ctx := WithTenantContext(context.Background(), "tenant-1")

	record := batchTenantTestModel{Name: "test"}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	if err := UnscopedDeleteWithTenant(ctx, db, &batchTenantTestModel{}, "id = ?", record.ID); err != nil {
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

	record1 := batchTenantTestModel{Name: "keep"}
	if err := db.WithContext(ctx1).Create(&record1).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	if err := UnscopedDeleteWithTenant(ctx2, db, &batchTenantTestModel{}, "id = ?", record1.ID); err != nil {
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
