package gormx

import (
	"context"
	"strings"
	"testing"

	"gorm.io/gorm/clause"
)

func TestUpsertRejectsNilData(t *testing.T) {
	db := &DB{DB: openTestDB(t, &pageTestModel{})}

	err := Upsert(context.Background(), db, nil, Columns("name"), []string{"name"})
	if err == nil || !strings.Contains(err.Error(), "data is nil") {
		t.Fatalf("error = %v, want data is nil", err)
	}
}

func TestUpsertRejectsEmptyConflictColumns(t *testing.T) {
	db := &DB{DB: openTestDB(t, &pageTestModel{})}

	err := Upsert(context.Background(), db, &pageTestModel{Name: "tom"}, nil, []string{"name"})
	if err == nil || !strings.Contains(err.Error(), "conflict columns is empty") {
		t.Fatalf("error = %v, want conflict columns is empty", err)
	}
}

func TestUpsertDoesNothingWhenUpdateColumnsEmpty(t *testing.T) {
	db := &DB{DB: openTestDB(t, &pageTestModel{})}
	first := pageTestModel{ID: 1, Name: "old"}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	err := Upsert(context.Background(), db, &pageTestModel{ID: 1, Name: "new"}, []clause.Column{{Name: "id"}}, nil)
	if err != nil {
		t.Fatalf("upsert error = %v", err)
	}

	var got pageTestModel
	if err := db.First(&got, 1).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.Name != "old" {
		t.Fatalf("name = %q, want old", got.Name)
	}
}
