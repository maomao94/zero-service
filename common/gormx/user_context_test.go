package gormx

import (
	"context"
	"testing"

	"gorm.io/gorm"
)

type stringAuditHookTestModel struct {
	ID uint `gorm:"primarykey"`
	StringAuditMixin
	Name string `gorm:"column:name"`
}

func (stringAuditHookTestModel) TableName() string {
	return "string_audit_hook_test_models"
}

func (m *stringAuditHookTestModel) BeforeCreate(tx *gorm.DB) error {
	m.BeforeCreateAudit(GetUserContext(tx.Statement.Context))
	return nil
}

type uintAuditHookTestModel struct {
	ID uint `gorm:"primarykey"`
	AuditMixin
	Name string `gorm:"column:name"`
}

func (uintAuditHookTestModel) TableName() string {
	return "uint_audit_hook_test_models"
}

func (m *uintAuditHookTestModel) BeforeCreate(tx *gorm.DB) error {
	m.BeforeCreateAudit(GetUserContext(tx.Statement.Context))
	return nil
}

func TestGetUserIDReturnsUint(t *testing.T) {
	ctx := WithUserContext(context.Background(), NewUserContext(uint(42), "tester", "tenant-1"))
	if got := GetUserID(ctx); got != 42 {
		t.Fatalf("GetUserID = %d, want 42", got)
	}
}

func TestGetUserIDReturnsZeroWhenNoContext(t *testing.T) {
	if got := GetUserID(context.Background()); got != 0 {
		t.Fatalf("GetUserID = %d, want 0", got)
	}
}

func TestGetUserIDTextReturnsString(t *testing.T) {
	ctx := WithUserContext(context.Background(), NewStringUserContext("user-abc", "tester", "tenant-1"))
	if got := GetUserIDText(ctx); got != "user-abc" {
		t.Fatalf("GetUserIDText = %q, want user-abc", got)
	}
}

func TestGetUserIDTextReturnsEmptyWhenNoContext(t *testing.T) {
	if got := GetUserIDText(context.Background()); got != "" {
		t.Fatalf("GetUserIDText = %q, want empty", got)
	}
}

func TestNewStringUserContextCreatesCorrectly(t *testing.T) {
	uc := NewStringUserContext("user-x", "tester", "tenant-1")
	if uc.UserID != "user-x" {
		t.Fatalf("UserID = %v, want user-x", uc.UserID)
	}
	if uc.UserName != "tester" {
		t.Fatalf("UserName = %q, want tester", uc.UserName)
	}
	if uc.TenantID != "tenant-1" {
		t.Fatalf("TenantID = %q, want tenant-1", uc.TenantID)
	}
}

func TestGenericUserContextFillsStringAuditFields(t *testing.T) {
	db := openTestDB(t, &stringAuditHookTestModel{})
	ctx := WithUserAndTenantContext(context.Background(), "8d98c7f4-0d90-4b91-a8e8-768d34da1d6a", "tester", "tenant-a")

	record := stringAuditHookTestModel{Name: "audit"}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	var got stringAuditHookTestModel
	if err := db.First(&got, record.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.CreateUser != "8d98c7f4-0d90-4b91-a8e8-768d34da1d6a" {
		t.Fatalf("create_user = %q, want uuid", got.CreateUser)
	}
	if got.UpdateUser != "8d98c7f4-0d90-4b91-a8e8-768d34da1d6a" {
		t.Fatalf("update_user = %q, want uuid", got.UpdateUser)
	}
	if got.CreateName != "tester" || got.UpdateName != "tester" {
		t.Fatalf("names = %q/%q, want tester/tester", got.CreateName, got.UpdateName)
	}
}

func TestGenericUserContextFillsUintAuditFields(t *testing.T) {
	db := openTestDB(t, &uintAuditHookTestModel{})
	ctx := WithUserAndTenantContext(context.Background(), uint(42), "tester", "tenant-a")

	record := uintAuditHookTestModel{Name: "audit"}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	var got uintAuditHookTestModel
	if err := db.First(&got, record.ID).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.CreateUser != 42 || got.UpdateUser != 42 {
		t.Fatalf("users = %d/%d, want 42/42", got.CreateUser, got.UpdateUser)
	}
	id, ok := GetUserIDAs[uint](ctx)
	if !ok || id != 42 {
		t.Fatalf("GetUserIDAs = %v/%v, want 42/true", id, ok)
	}
}
