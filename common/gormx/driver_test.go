package gormx

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDatabaseTypeOfUsesDBDialector(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db error = %v", err)
	}

	db.Statement = nil
	if got := DatabaseTypeOf(db); got != DatabaseSQLite {
		t.Fatalf("database type = %s, want %s", got, DatabaseSQLite)
	}
}
