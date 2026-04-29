package gormx

import (
	"context"
	"testing"
)

func TestGenericUserContextFillsStringAuditFields(t *testing.T) {
	db := openTestDB(t, &stringAuditTestModel{})
	ctx := WithUserAndTenantContext(context.Background(), "8d98c7f4-0d90-4b91-a8e8-768d34da1d6a", "tester", "tenant-a")

	record := stringAuditTestModel{Name: "audit"}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	var got stringAuditTestModel
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
	db := openTestDB(t, &uintAuditTestModel{})
	ctx := WithUserAndTenantContext(context.Background(), uint(42), "tester", "tenant-a")

	record := uintAuditTestModel{Name: "audit"}
	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	var got uintAuditTestModel
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
