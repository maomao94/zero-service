package gormx

import (
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
