package gormx

import (
	"context"
	"testing"
)

func TestRestoreWithTenantRestoresSoftDeletedRecord(t *testing.T) {
	db := openTestDB(t, &batchTenantSoftDeleteTestModel{})
	ctx := WithTenantContext(context.Background(), "tenant-1")

	record := batchTenantSoftDeleteTestModel{Name: "test"}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	if err := db.WithContext(ctx).Delete(&record).Error; err != nil {
		t.Fatalf("delete error = %v", err)
	}

	if err := RestoreWithTenant(ctx, db, &batchTenantSoftDeleteTestModel{}, "id = ?", record.ID); err != nil {
		t.Fatalf("restore error = %v", err)
	}

	var got batchTenantSoftDeleteTestModel
	if err := db.WithContext(ctx).First(&got, record.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.Name != "test" {
		t.Fatalf("name = %q, want test", got.Name)
	}
}

func TestRestoreWithTenantDoesNotAffectOtherTenant(t *testing.T) {
	db := openTestDB(t, &batchTenantSoftDeleteTestModel{})
	ctx1 := WithTenantContext(context.Background(), "tenant-1")
	ctx2 := WithTenantContext(context.Background(), "tenant-2")

	record1 := batchTenantSoftDeleteTestModel{Name: "test"}
	if err := db.WithContext(ctx1).Create(&record1).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}
	if err := db.WithContext(ctx1).Delete(&record1).Error; err != nil {
		t.Fatalf("delete error = %v", err)
	}

	if err := RestoreWithTenant(ctx2, db, &batchTenantSoftDeleteTestModel{}, "id = ?", record1.ID); err != nil {
		t.Fatalf("restore error = %v", err)
	}

	var count int64
	if err := db.Model(&batchTenantSoftDeleteTestModel{}).Where("id = ?", record1.ID).Count(&count).Error; err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0 (other tenant should not restore)", count)
	}
}
