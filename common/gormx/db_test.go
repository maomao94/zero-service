package gormx

import (
	"context"
	"errors"
	"testing"
)

func TestTransactCommitsOnSuccess(t *testing.T) {
	db := openTestDB(t, &pageTestModel{})

	wrapped := &DB{DB: db}
	if err := wrapped.Transact(func(tx *DB) error {
		return tx.Create(&pageTestModel{Name: "tx-ok"}).Error
	}); err != nil {
		t.Fatalf("transact error = %v", err)
	}

	var got pageTestModel
	if err := db.Where("name = ?", "tx-ok").First(&got).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.Name != "tx-ok" {
		t.Fatalf("name = %q, want tx-ok", got.Name)
	}
}

func TestTransactRollbacksOnError(t *testing.T) {
	db := openTestDB(t, &pageTestModel{})

	wrapped := &DB{DB: db}
	err := wrapped.Transact(func(tx *DB) error {
		if err := tx.Create(&pageTestModel{Name: "tx-fail"}).Error; err != nil {
			return err
		}
		return errors.New("rollback please")
	})
	if err == nil || err.Error() != "rollback please" {
		t.Fatalf("error = %v, want rollback please", err)
	}

	var count int64
	if err := db.Model(&pageTestModel{}).Where("name = ?", "tx-fail").Count(&count).Error; err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0 after rollback", count)
	}
}

func TestWithTenantAddsTenantFilter(t *testing.T) {
	gormDB := openTestDB(t, &tenantScopeTestModel{})
	db := &DB{DB: gormDB}
	ctx := WithTenantContext(context.Background(), "tenant-1")

	if err := gormDB.WithContext(ctx).Create(&tenantScopeTestModel{TenantMixin: TenantMixin{TenantID: "tenant-1"}, Name: "a"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	var got tenantScopeTestModel
	if err := db.WithTenant(ctx).First(&got).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.TenantID != "tenant-1" {
		t.Fatalf("tenant_id = %q, want tenant-1", got.TenantID)
	}
}

func TestWithTenantStrictReturnsEmptyWhenNoTenant(t *testing.T) {
	gormDB := openTestDB(t, &tenantScopeTestModel{})
	db := &DB{DB: gormDB}
	ctx := context.Background()

	if err := gormDB.WithContext(WithTenantContext(ctx, "tenant-1")).Create(&tenantScopeTestModel{TenantMixin: TenantMixin{TenantID: "tenant-1"}, Name: "a"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	var count int64
	if err := db.WithTenantStrict(ctx).Model(&tenantScopeTestModel{}).Count(&count).Error; err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0 when no tenant context", count)
	}
}

func TestWithDeletedReturnsUnscopedQuery(t *testing.T) {
	gormDB := openTestDB(t, &callbackSoftDeleteTestModel{})
	db := &DB{DB: gormDB}
	ctx := WithUserAndTenantContext(context.Background(), "user-1", "tester", "tenant-1")

	record := callbackSoftDeleteTestModel{Name: "soft"}
	if err := gormDB.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	if err := gormDB.WithContext(ctx).Delete(&record).Error; err != nil {
		t.Fatalf("delete error = %v", err)
	}

	var count int64
	if err := gormDB.Model(&callbackSoftDeleteTestModel{}).Count(&count).Error; err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0 after soft delete", count)
	}

	var total int64
	if err := db.WithDeleted(ctx).Model(&callbackSoftDeleteTestModel{}).Count(&total).Error; err != nil {
		t.Fatalf("count unscoped error = %v", err)
	}
	if total != 1 {
		t.Fatalf("count = %d, want 1 with deleted", total)
	}
}

func TestWithTenantDeletedIncludesDeletedForTenant(t *testing.T) {
	gormDB := openTestDB(t, &tenantScopeSoftDeleteTestModel{})
	db := &DB{DB: gormDB}
	ctx := WithTenantContext(context.Background(), "tenant-1")

	record := tenantScopeSoftDeleteTestModel{TenantMixin: TenantMixin{TenantID: "tenant-1"}, Name: "a"}
	if err := gormDB.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	if err := gormDB.WithContext(ctx).Delete(&record).Error; err != nil {
		t.Fatalf("delete error = %v", err)
	}

	var total int64
	if err := db.WithTenantDeleted(ctx).Model(&tenantScopeSoftDeleteTestModel{}).Count(&total).Error; err != nil {
		t.Fatalf("count error = %v", err)
	}
	if total != 1 {
		t.Fatalf("count = %d, want 1 with tenant deleted", total)
	}
}
