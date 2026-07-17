package gormx

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type legacyDeleteTestModel struct {
	LegacyStringBaseModel
	AuditMixin
	TenantMixin
	Name string `gorm:"column:name;uniqueIndex"`
}

func (legacyDeleteTestModel) TableName() string {
	return "legacy_delete_test_models"
}

type stringAuditTestModel struct {
	ID uint `gorm:"primarykey"`
	StringAuditMixin
	Name string `gorm:"column:name"`
}

func (stringAuditTestModel) TableName() string {
	return "string_audit_test_models"
}

type legacyStringIDTestModel struct {
	LegacyStringBaseModel
	Name string `gorm:"column:name"`
}

func (legacyStringIDTestModel) TableName() string {
	return "legacy_string_id_test_models"
}

type uintAuditTestModel struct {
	ID uint `gorm:"primarykey"`
	AuditMixin
	Name string `gorm:"column:name"`
}

func (uintAuditTestModel) TableName() string {
	return "uint_audit_test_models"
}

type pageTestModel struct {
	ID   uint `gorm:"primarykey"`
	Name string
}

func openTestDB(t *testing.T, models ...any) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&_loc=auto&parseTime=true"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		t.Fatalf("open db error = %v", err)
	}
	RegisterCallbacks(db)
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("auto migrate error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql db error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}
