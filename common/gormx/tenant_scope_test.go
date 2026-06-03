package gormx

import (
	"context"
	"testing"

	"gorm.io/gorm"
)

type tenantScopeTestModel struct {
	ID uint `gorm:"primarykey"`
	TenantMixin
	Name string `gorm:"column:name"`
}

func (tenantScopeTestModel) TableName() string {
	return "tenant_scope_test_models"
}

type tenantScopeSoftDeleteTestModel struct {
	ID uint `gorm:"primarykey"`
	TenantMixin
	SoftDeleteMixin
	Name string `gorm:"column:name"`
}

func (tenantScopeSoftDeleteTestModel) TableName() string {
	return "tenant_scope_soft_test_models"
}

type noTenantModel struct {
	ID   uint   `gorm:"primarykey"`
	Name string `gorm:"column:name"`
}

func (noTenantModel) TableName() string {
	return "no_tenant_models"
}

func TestTenantScopeFiltersByTenantID(t *testing.T) {
	db := openTestDB(t, &tenantScopeTestModel{})

	ctx1 := WithTenantContext(context.Background(), "tenant-1")
	ctx2 := WithTenantContext(context.Background(), "tenant-2")

	if err := db.WithContext(ctx1).Create(&tenantScopeTestModel{Name: "a"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}
	if err := db.WithContext(ctx1).Create(&tenantScopeTestModel{Name: "b"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}
	if err := db.WithContext(ctx2).Create(&tenantScopeTestModel{Name: "c"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	// Query with tenant-1 scope
	var list []tenantScopeTestModel
	if err := db.Scopes(TenantScope(ctx1)).Find(&list).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("count = %d, want 2", len(list))
	}
}

func TestTenantScopeReturnsAllWhenNoTenantContext(t *testing.T) {
	db := openTestDB(t, &tenantScopeTestModel{})

	records := []tenantScopeTestModel{
		{Name: "a"},
		{Name: "b"},
	}
	ctx := WithTenantContext(context.Background(), "tenant-1")
	if err := db.WithContext(ctx).Create(&records).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	// Query without tenant context
	var list []tenantScopeTestModel
	if err := db.Scopes(TenantScope(context.Background())).Find(&list).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("count = %d, want 2", len(list))
	}
}

func TestTenantScopeReturnsAllWhenNoTenantField(t *testing.T) {
	db := openTestDB(t, &noTenantModel{})

	records := []noTenantModel{
		{Name: "a"},
		{Name: "b"},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	ctx := WithTenantContext(context.Background(), "tenant-1")
	var list []noTenantModel
	// Without tenant_id column, the query will fail with a SQL error
	err := db.Scopes(TenantScope(ctx)).Find(&list).Error
	if err == nil {
		t.Fatalf("expected error when using TenantScope on model without tenant_id")
	}
}

func TestTenantScopeStrictReturnsEmptyWhenNoTenantContext(t *testing.T) {
	db := openTestDB(t, &tenantScopeTestModel{})

	records := []tenantScopeTestModel{
		{Name: "a"},
	}
	ctx := WithTenantContext(context.Background(), "tenant-1")
	if err := db.WithContext(ctx).Create(&records).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	// Query without tenant context - should return empty
	var list []tenantScopeTestModel
	if err := db.Scopes(TenantScopeStrict(context.Background())).Find(&list).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("count = %d, want 0", len(list))
	}
}

func TestTenantScopeStrictFiltersByTenantID(t *testing.T) {
	db := openTestDB(t, &tenantScopeTestModel{})

	ctx := WithTenantContext(context.Background(), "tenant-1")
	if err := db.WithContext(ctx).Create(&tenantScopeTestModel{Name: "a"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}
	if err := db.WithContext(ctx).Create(&tenantScopeTestModel{Name: "b"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	var list []tenantScopeTestModel
	if err := db.Scopes(TenantScopeStrict(ctx)).Find(&list).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("count = %d, want 2", len(list))
	}
}

func TestTenantScopeStrictReturnsAllWhenNoTenantField(t *testing.T) {
	db := openTestDB(t, &noTenantModel{})

	records := []noTenantModel{
		{Name: "a"},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	ctx := WithTenantContext(context.Background(), "tenant-1")
	var list []noTenantModel
	err := db.Scopes(TenantScopeStrict(ctx)).Find(&list).Error
	if err == nil {
		t.Fatalf("expected error when using TenantScopeStrict on model without tenant_id")
	}
}

func TestTenantScopeWithDeleteIncludesDeletedRecords(t *testing.T) {
	db := openTestDB(t, &tenantScopeSoftDeleteTestModel{})
	ctx := WithTenantContext(context.Background(), "tenant-1")

	record := tenantScopeSoftDeleteTestModel{Name: "test"}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	if err := db.WithContext(ctx).Delete(&record).Error; err != nil {
		t.Fatalf("delete error = %v", err)
	}

	var list []tenantScopeSoftDeleteTestModel
	if err := db.WithContext(ctx).Scopes(TenantScopeWithDelete(ctx)).Find(&list).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("count = %d, want 1", len(list))
	}
}

func TestTenantScopeWithDeleteReturnsAllWhenNoTenantContext(t *testing.T) {
	db := openTestDB(t, &tenantScopeSoftDeleteTestModel{})

	record := tenantScopeSoftDeleteTestModel{Name: "test"}
	ctx := WithTenantContext(context.Background(), "tenant-1")
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}
	if err := db.WithContext(ctx).Delete(&record).Error; err != nil {
		t.Fatalf("delete error = %v", err)
	}

	var list []tenantScopeSoftDeleteTestModel
	if err := db.Scopes(TenantScopeWithDelete(context.Background())).Find(&list).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("count = %d, want 1 (Unscoped should include deleted records)", len(list))
	}
}

func TestTenantEqFiltersByTenantID(t *testing.T) {
	db := openTestDB(t, &tenantScopeTestModel{})

	ctx1 := WithTenantContext(context.Background(), "tenant-1")
	ctx2 := WithTenantContext(context.Background(), "tenant-2")

	if err := db.WithContext(ctx1).Create(&tenantScopeTestModel{Name: "a"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}
	if err := db.WithContext(ctx1).Create(&tenantScopeTestModel{Name: "b"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}
	if err := db.WithContext(ctx2).Create(&tenantScopeTestModel{Name: "c"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	// Query with TenantEq
	var list []tenantScopeTestModel
	if err := db.Scopes(TenantEq("tenant-1")).Find(&list).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("count = %d, want 2", len(list))
	}
}

func TestTenantEqReturnsAllWhenEmptyTenantID(t *testing.T) {
	db := openTestDB(t, &tenantScopeTestModel{})

	record := tenantScopeTestModel{Name: "test"}
	ctx := WithTenantContext(context.Background(), "tenant-1")
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	// Query with empty tenant ID
	var list []tenantScopeTestModel
	if err := db.Scopes(TenantEq("")).Find(&list).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("count = %d, want 1", len(list))
	}
}

func TestTenantNotEqFiltersByTenantID(t *testing.T) {
	db := openTestDB(t, &tenantScopeTestModel{})

	ctx1 := WithTenantContext(context.Background(), "tenant-1")
	ctx2 := WithTenantContext(context.Background(), "tenant-2")

	if err := db.WithContext(ctx1).Create(&tenantScopeTestModel{Name: "a"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}
	if err := db.WithContext(ctx1).Create(&tenantScopeTestModel{Name: "b"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}
	if err := db.WithContext(ctx2).Create(&tenantScopeTestModel{Name: "c"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	// Query with TenantNotEq
	var list []tenantScopeTestModel
	if err := db.Scopes(TenantNotEq("tenant-1")).Find(&list).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("count = %d, want 1", len(list))
	}
}

func TestTenantInFiltersByTenantIDs(t *testing.T) {
	db := openTestDB(t, &tenantScopeTestModel{})

	ctx1 := WithTenantContext(context.Background(), "tenant-1")
	ctx2 := WithTenantContext(context.Background(), "tenant-2")
	ctx3 := WithTenantContext(context.Background(), "tenant-3")

	if err := db.WithContext(ctx1).Create(&tenantScopeTestModel{Name: "a"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}
	if err := db.WithContext(ctx2).Create(&tenantScopeTestModel{Name: "b"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}
	if err := db.WithContext(ctx3).Create(&tenantScopeTestModel{Name: "c"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	// Query with TenantIn
	var list []tenantScopeTestModel
	if err := db.Scopes(TenantIn("tenant-1", "tenant-2")).Find(&list).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("count = %d, want 2", len(list))
	}
}

func TestTenantInReturnsAllWhenEmptyTenantIDs(t *testing.T) {
	db := openTestDB(t, &tenantScopeTestModel{})

	record := tenantScopeTestModel{Name: "test"}
	ctx := WithTenantContext(context.Background(), "tenant-1")
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	// Query with empty tenant IDs
	var list []tenantScopeTestModel
	if err := db.Scopes(TenantIn()).Find(&list).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("count = %d, want 1", len(list))
	}
}

func TestWithTenantContextSetsTenantID(t *testing.T) {
	ctx := WithTenantContext(context.Background(), "tenant-1")
	userCtx := GetUserContext(ctx)
	if userCtx == nil {
		t.Fatalf("user context should not be nil")
	}
	if userCtx.TenantID != "tenant-1" {
		t.Fatalf("tenant_id = %q, want tenant-1", userCtx.TenantID)
	}
}

func TestWithUserAndTenantContextSetsUserAndTenant(t *testing.T) {
	ctx := WithUserAndTenantContext(context.Background(), uint(42), "tester", "tenant-1")
	userCtx := GetUserContext(ctx)
	if userCtx == nil {
		t.Fatalf("user context should not be nil")
	}
	if userCtx.TenantID != "tenant-1" {
		t.Fatalf("tenant_id = %q, want tenant-1", userCtx.TenantID)
	}
	if userCtx.UserName != "tester" {
		t.Fatalf("user_name = %q, want tester", userCtx.UserName)
	}
}

func TestHasTenantFieldReturnsTrueForTenantModel(t *testing.T) {
	db := openTestDB(t, &tenantScopeTestModel{})
	tx := db.Session(&gorm.Session{}).Model(&tenantScopeTestModel{})
	var list []tenantScopeTestModel
	tx.Find(&list)
	if !HasTenantField(tx) {
		t.Fatalf("should have tenant field")
	}
}

func TestHasTenantFieldReturnsFalseForNoTenantModel(t *testing.T) {
	db := openTestDB(t, &noTenantModel{})
	tx := db.Session(&gorm.Session{}).Model(&noTenantModel{})
	var list []noTenantModel
	tx.Find(&list)
	if HasTenantField(tx) {
		t.Fatalf("should not have tenant field")
	}
}
