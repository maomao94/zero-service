package gormx

import (
	"context"
	"testing"

	"gorm.io/gorm"
)

type batchTestModel struct {
	ID   uint   `gorm:"primarykey"`
	Name string `gorm:"column:name"`
}

func (batchTestModel) TableName() string {
	return "batch_test_models"
}

type batchTenantTestModel struct {
	ID uint `gorm:"primarykey"`
	TenantMixin
	Name string `gorm:"column:name"`
}

func (batchTenantTestModel) TableName() string {
	return "batch_tenant_test_models"
}

type batchTenantSoftDeleteTestModel struct {
	ID uint `gorm:"primarykey"`
	TenantMixin
	SoftDeleteMixin
	Name string `gorm:"column:name"`
}

func (batchTenantSoftDeleteTestModel) TableName() string {
	return "batch_tenant_soft_test_models"
}

func TestBatchInsertCreatesRecords(t *testing.T) {
	db := openTestDB(t, &batchTestModel{})

	records := []batchTestModel{
		{Name: "a"},
		{Name: "b"},
		{Name: "c"},
	}
	if err := BatchInsert(db, records, 0); err != nil {
		t.Fatalf("batch insert error = %v", err)
	}

	var count int64
	if err := db.Model(&batchTestModel{}).Count(&count).Error; err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 3 {
		t.Fatalf("count = %d, want 3", count)
	}
}

func TestBatchInsertWithCustomBatchSize(t *testing.T) {
	db := openTestDB(t, &batchTestModel{})

	records := []batchTestModel{
		{Name: "a"},
		{Name: "b"},
		{Name: "c"},
		{Name: "d"},
		{Name: "e"},
	}
	if err := BatchInsert(db, records, 2); err != nil {
		t.Fatalf("batch insert error = %v", err)
	}

	var count int64
	if err := db.Model(&batchTestModel{}).Count(&count).Error; err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 5 {
		t.Fatalf("count = %d, want 5", count)
	}
}

func TestBatchInsertEmptySlice(t *testing.T) {
	db := openTestDB(t, &batchTestModel{})

	if err := BatchInsert(db, []batchTestModel{}, 100); err != nil {
		t.Fatalf("batch insert error = %v", err)
	}
}

func TestBatchUpdateByIdsUpdatesRecords(t *testing.T) {
	db := openTestDB(t, &batchTestModel{})

	// Create test records
	records := []batchTestModel{
		{Name: "old1"},
		{Name: "old2"},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	updates := []Ups{
		{"id": records[0].ID, "name": "new1"},
		{"id": records[1].ID, "name": "new2"},
	}
	if err := BatchUpdateByIds(db, &batchTestModel{}, updates); err != nil {
		t.Fatalf("batch update error = %v", err)
	}

	var got batchTestModel
	if err := db.First(&got, records[0].ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.Name != "new1" {
		t.Fatalf("name = %q, want new1", got.Name)
	}
}

func TestBatchUpdateByIdsSkipsMissingID(t *testing.T) {
	db := openTestDB(t, &batchTestModel{})

	record := batchTestModel{Name: "old"}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	updates := []Ups{
		{"name": "new"}, // missing id
	}
	if err := BatchUpdateByIds(db, &batchTestModel{}, updates); err != nil {
		t.Fatalf("batch update error = %v", err)
	}

	var got batchTestModel
	if err := db.First(&got, record.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.Name != "old" {
		t.Fatalf("name = %q, want old", got.Name)
	}
}

func TestBatchUpdateByIdsEmptySlice(t *testing.T) {
	db := openTestDB(t, &batchTestModel{})

	if err := BatchUpdateByIds(db, &batchTestModel{}, []Ups{}); err != nil {
		t.Fatalf("batch update error = %v", err)
	}
}

func TestBatchDeleteByIdsDeletesRecords(t *testing.T) {
	db := openTestDB(t, &batchTestModel{})

	records := []batchTestModel{
		{Name: "a"},
		{Name: "b"},
		{Name: "c"},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	ids := []int64{int64(records[0].ID), int64(records[1].ID)}
	if err := BatchDeleteByIds(db, &batchTestModel{}, ids); err != nil {
		t.Fatalf("batch delete error = %v", err)
	}

	var count int64
	if err := db.Model(&batchTestModel{}).Count(&count).Error; err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestBatchDeleteByIdsEmptySlice(t *testing.T) {
	db := openTestDB(t, &batchTestModel{})

	if err := BatchDeleteByIds(db, &batchTestModel{}, []int64{}); err != nil {
		t.Fatalf("batch delete error = %v", err)
	}
}

func TestBatchDeleteByConditionDeletesMatchingRecords(t *testing.T) {
	db := openTestDB(t, &batchTestModel{})

	records := []batchTestModel{
		{Name: "keep"},
		{Name: "delete"},
		{Name: "delete"},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	if err := BatchDeleteByCondition(db, &batchTestModel{}, func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ?", "delete")
	}); err != nil {
		t.Fatalf("batch delete error = %v", err)
	}

	var count int64
	if err := db.Model(&batchTestModel{}).Count(&count).Error; err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestBatchInsertWithTenantSetsTenantID(t *testing.T) {
	db := openTestDB(t, &batchTenantTestModel{})
	ctx := WithTenantContext(context.Background(), "tenant-1")

	records := []batchTenantTestModel{
		{Name: "a"},
		{Name: "b"},
	}
	if err := BatchInsertWithTenant(ctx, db, records); err != nil {
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

	if err := BatchInsertWithTenant(ctx, db, []batchTenantTestModel{}); err != nil {
		t.Fatalf("batch insert error = %v", err)
	}
}

func TestBatchUpdateByIdsWithTenantUpdatesRecords(t *testing.T) {
	db := openTestDB(t, &batchTenantTestModel{})
	ctx := WithTenantContext(context.Background(), "tenant-1")

	record := batchTenantTestModel{Name: "old"}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	updates := []Ups{
		{"id": record.ID, "name": "new"},
	}
	if err := BatchUpdateByIdsWithTenant(ctx, db, &batchTenantTestModel{}, updates); err != nil {
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

	if err := BatchUpdateByIdsWithTenant(ctx, db, &batchTenantTestModel{}, []Ups{}); err != nil {
		t.Fatalf("batch update error = %v", err)
	}
}

func TestBatchDeleteByIdsWithTenantDeletesRecords(t *testing.T) {
	db := openTestDB(t, &batchTenantTestModel{})
	ctx := WithTenantContext(context.Background(), "tenant-1")

	records := []batchTenantTestModel{
		{Name: "a"},
		{Name: "b"},
	}
	if err := db.WithContext(ctx).Create(&records).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	ids := []int64{int64(records[0].ID)}
	if err := BatchDeleteByIdsWithTenant(ctx, db, &batchTenantTestModel{}, ids); err != nil {
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

	if err := BatchDeleteByIdsWithTenant(ctx, db, &batchTenantTestModel{}, []int64{}); err != nil {
		t.Fatalf("batch delete error = %v", err)
	}
}

func TestBatchDeleteByConditionWithTenantDeletesMatchingRecords(t *testing.T) {
	db := openTestDB(t, &batchTenantTestModel{})
	ctx := WithTenantContext(context.Background(), "tenant-1")

	if err := db.WithContext(ctx).Create(&batchTenantTestModel{Name: "keep"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}
	if err := db.WithContext(ctx).Create(&batchTenantTestModel{Name: "delete"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	if err := BatchDeleteByConditionWithTenant(ctx, db, &batchTenantTestModel{}, func(db *gorm.DB) *gorm.DB {
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

func TestBatchUpdateByIdsWithTenantDoesNotAffectOtherTenant(t *testing.T) {
	db := openTestDB(t, &batchTenantTestModel{})
	ctx1 := WithTenantContext(context.Background(), "tenant-1")
	ctx2 := WithTenantContext(context.Background(), "tenant-2")

	record1 := batchTenantTestModel{Name: "old"}
	if err := db.WithContext(ctx1).Create(&record1).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	updates := []Ups{{"id": record1.ID, "name": "hacked"}}
	if err := BatchUpdateByIdsWithTenant(ctx2, db, &batchTenantTestModel{}, updates); err != nil {
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

	record1 := batchTenantTestModel{Name: "keep"}
	if err := db.WithContext(ctx1).Create(&record1).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	ids := []int64{int64(record1.ID)}
	if err := BatchDeleteByIdsWithTenant(ctx2, db, &batchTenantTestModel{}, ids); err != nil {
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
