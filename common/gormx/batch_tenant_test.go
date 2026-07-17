package gormx

import (
	"context"
	"testing"

	"gorm.io/gorm"
)

func TestBatchInsertWithTenantSetsTenantID(t *testing.T) {
	db := openTestDB(t, &batchTenantTestModel{})
	ctx := WithTenantContext(context.Background(), "tenant-1")

	records := []batchTenantTestModel{
		{TenantMixin: TenantMixin{TenantID: "tenant-1"}, Name: "a"},
		{TenantMixin: TenantMixin{TenantID: "tenant-1"}, Name: "b"},
	}
	if err := BatchInsertWithTenant(db.WithContext(ctx), records); err != nil {
		t.Fatalf("batch insert error = %v", err)
	}

	var got batchTenantTestModel
	if err := db.First(&got).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.TenantID != "tenant-1" {
		t.Fatalf("tenant_id = %q, want tenant-1", got.TenantID)
	}
}

func TestBatchInsertWithTenantEmptySlice(t *testing.T) {
	db := openTestDB(t, &batchTenantTestModel{})
	ctx := WithTenantContext(context.Background(), "tenant-1")

	if err := BatchInsertWithTenant(db.WithContext(ctx), []batchTenantTestModel{}); err != nil {
		t.Fatalf("batch insert error = %v", err)
	}
}

func TestBatchUpdateByIdsWithTenantUpdatesRecords(t *testing.T) {
	db := openTestDB(t, &batchTenantTestModel{})
	ctx := WithTenantContext(context.Background(), "tenant-1")

	record := batchTenantTestModel{TenantMixin: TenantMixin{TenantID: "tenant-1"}, Name: "old"}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	updates := []Ups{
		{"id": record.ID, "name": "new"},
	}
	if err := BatchUpdateByIdsWithTenant(db.WithContext(ctx), &batchTenantTestModel{}, updates); err != nil {
		t.Fatalf("batch update error = %v", err)
	}

	var got batchTenantTestModel
	if err := db.First(&got, record.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.Name != "new" {
		t.Fatalf("name = %q, want new", got.Name)
	}
}

func TestBatchUpdateByIdsWithTenantEmptySlice(t *testing.T) {
	db := openTestDB(t, &batchTenantTestModel{})
	ctx := WithTenantContext(context.Background(), "tenant-1")

	if err := BatchUpdateByIdsWithTenant(db.WithContext(ctx), &batchTenantTestModel{}, []Ups{}); err != nil {
		t.Fatalf("batch update error = %v", err)
	}
}

func TestBatchDeleteByIdsWithTenantDeletesRecords(t *testing.T) {
	db := openTestDB(t, &batchTenantTestModel{})
	ctx := WithTenantContext(context.Background(), "tenant-1")

	records := []batchTenantTestModel{
		{TenantMixin: TenantMixin{TenantID: "tenant-1"}, Name: "a"},
		{TenantMixin: TenantMixin{TenantID: "tenant-1"}, Name: "b"},
	}
	if err := db.WithContext(ctx).Create(&records).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	ids := []int64{int64(records[0].ID)}
	if err := BatchDeleteByIdsWithTenant(db.WithContext(ctx), &batchTenantTestModel{}, ids); err != nil {
		t.Fatalf("batch delete error = %v", err)
	}

	var count int64
	if err := db.Model(&batchTenantTestModel{}).Count(&count).Error; err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestBatchDeleteByIdsWithTenantEmptySlice(t *testing.T) {
	db := openTestDB(t, &batchTenantTestModel{})
	ctx := WithTenantContext(context.Background(), "tenant-1")

	if err := BatchDeleteByIdsWithTenant(db.WithContext(ctx), &batchTenantTestModel{}, []int64{}); err != nil {
		t.Fatalf("batch delete error = %v", err)
	}
}

func TestBatchDeleteByConditionWithTenantDeletesMatchingRecords(t *testing.T) {
	db := openTestDB(t, &batchTenantTestModel{})
	ctx := WithTenantContext(context.Background(), "tenant-1")

	if err := db.WithContext(ctx).Create(&batchTenantTestModel{TenantMixin: TenantMixin{TenantID: "tenant-1"}, Name: "keep"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}
	if err := db.WithContext(ctx).Create(&batchTenantTestModel{TenantMixin: TenantMixin{TenantID: "tenant-1"}, Name: "delete"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	if err := BatchDeleteByConditionWithTenant(db.WithContext(ctx), &batchTenantTestModel{}, func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ?", "delete")
	}); err != nil {
		t.Fatalf("batch delete error = %v", err)
	}

	var count int64
	if err := db.WithContext(ctx).Model(&batchTenantTestModel{}).Count(&count).Error; err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestBatchUpdateByIdsWithTenantDoesNotAffectOtherTenant(t *testing.T) {
	db := openTestDB(t, &batchTenantTestModel{})
	ctx1 := WithTenantContext(context.Background(), "tenant-1")
	ctx2 := WithTenantContext(context.Background(), "tenant-2")

	record1 := batchTenantTestModel{TenantMixin: TenantMixin{TenantID: "tenant-1"}, Name: "old"}
	if err := db.WithContext(ctx1).Create(&record1).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	updates := []Ups{{"id": record1.ID, "name": "hacked"}}
	if err := BatchUpdateByIdsWithTenant(db.WithContext(ctx2), &batchTenantTestModel{}, updates); err != nil {
		t.Fatalf("batch update error = %v", err)
	}

	var got batchTenantTestModel
	if err := db.First(&got, record1.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.Name != "old" {
		t.Fatalf("name = %q, want old (other tenant should not update)", got.Name)
	}
}

func TestBatchDeleteByIdsWithTenantDoesNotAffectOtherTenant(t *testing.T) {
	db := openTestDB(t, &batchTenantTestModel{})
	ctx1 := WithTenantContext(context.Background(), "tenant-1")
	ctx2 := WithTenantContext(context.Background(), "tenant-2")

	record1 := batchTenantTestModel{TenantMixin: TenantMixin{TenantID: "tenant-1"}, Name: "keep"}
	if err := db.WithContext(ctx1).Create(&record1).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	ids := []int64{int64(record1.ID)}
	if err := BatchDeleteByIdsWithTenant(db.WithContext(ctx2), &batchTenantTestModel{}, ids); err != nil {
		t.Fatalf("batch delete error = %v", err)
	}

	var count int64
	if err := db.Model(&batchTenantTestModel{}).Where("id = ?", record1.ID).Count(&count).Error; err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1 (other tenant should not delete)", count)
	}
}
