package gormx

import (
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestExplainSQLReturnsReadableSQL(t *testing.T) {
	dialector := gorm.Dialector(sqlite.Open("file:" + t.Name() + "?mode=memory&cache=shared"))
	db, err := OpenWithDialector(&dialector, WithoutOpenTelemetry())
	if err != nil {
		t.Fatalf("open db error = %v", err)
	}

	sql := db.ExplainSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&pageTestModel{}).Where("name = ?", "tom").Find(&[]pageTestModel{})
	})

	if !strings.Contains(sql, "SELECT") {
		t.Fatalf("sql should contain SELECT, got %q", sql)
	}
	if !strings.Contains(sql, "tom") {
		t.Fatalf("sql should contain query value, got %q", sql)
	}
}
