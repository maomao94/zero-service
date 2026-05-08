package gormx

import (
	"context"
	"strings"
	"testing"

	"gorm.io/gorm/clause"
)

type updateOrCreateTestModel struct {
	ID      uint   `gorm:"primarykey"`
	Code    string `gorm:"column:code;uniqueIndex"`
	Name    string `gorm:"column:name"`
	Ignored string `gorm:"column:ignored"`
}

func (updateOrCreateTestModel) TableName() string {
	return "update_or_create_test_models"
}

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

func TestUpdateOrCreateCreatesWhenRecordMissing(t *testing.T) {
	db := &DB{DB: openTestDB(t, &updateOrCreateTestModel{})}

	err := UpdateOrCreate(
		context.Background(),
		db,
		&updateOrCreateTestModel{},
		map[string]any{"code": "A"},
		&updateOrCreateTestModel{Code: "A", Name: "created", Ignored: "keep"},
		map[string]any{"name": "updated"},
	)
	if err != nil {
		t.Fatalf("update or create error = %v", err)
	}

	var got updateOrCreateTestModel
	if err := db.Where("code = ?", "A").First(&got).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.Name != "created" || got.Ignored != "keep" {
		t.Fatalf("got name=%q ignored=%q, want created/keep", got.Name, got.Ignored)
	}
}

func TestUpdateOrCreateUpdatesOnlyRequestedColumnsWhenRecordExists(t *testing.T) {
	db := &DB{DB: openTestDB(t, &updateOrCreateTestModel{})}
	first := updateOrCreateTestModel{Code: "A", Name: "old", Ignored: "keep"}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	err := UpdateOrCreate(
		context.Background(),
		db,
		&updateOrCreateTestModel{},
		map[string]any{"code": "A"},
		&updateOrCreateTestModel{Code: "A", Name: "created", Ignored: "changed"},
		map[string]any{"name": "updated"},
	)
	if err != nil {
		t.Fatalf("update or create error = %v", err)
	}

	var got updateOrCreateTestModel
	if err := db.Where("code = ?", "A").First(&got).Error; err != nil {
		t.Fatalf("find error = %v", err)
	}
	if got.Name != "updated" || got.Ignored != "keep" {
		t.Fatalf("got name=%q ignored=%q, want updated/keep", got.Name, got.Ignored)
	}
}

func TestUpdateOrCreateRejectsEmptyWhere(t *testing.T) {
	db := &DB{DB: openTestDB(t, &updateOrCreateTestModel{})}

	err := UpdateOrCreate(context.Background(), db, &updateOrCreateTestModel{}, nil, &updateOrCreateTestModel{}, map[string]any{"name": "updated"})
	if err == nil || !strings.Contains(err.Error(), "where is empty") {
		t.Fatalf("error = %v, want where is empty", err)
	}
}
