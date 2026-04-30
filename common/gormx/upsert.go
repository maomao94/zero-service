package gormx

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Column(name string) clause.Column {
	return clause.Column{Name: name}
}

func Columns(names ...string) []clause.Column {
	columns := make([]clause.Column, 0, len(names))
	for _, name := range names {
		if name == "" {
			continue
		}
		columns = append(columns, Column(name))
	}
	return columns
}

func Upsert(ctx context.Context, db *DB, data any, columns []clause.Column, updateColumns []string) error {
	if db == nil {
		return errors.New("gormx db is nil")
	}
	if data == nil {
		return errors.New("gormx upsert data is nil")
	}
	if len(columns) == 0 {
		return errors.New("gormx upsert conflict columns is empty")
	}
	conflict := clause.OnConflict{Columns: columns}
	if len(updateColumns) == 0 {
		conflict.DoNothing = true
	} else {
		conflict.DoUpdates = clause.AssignmentColumns(updateColumns)
	}
	return db.WithContext(ctx).Clauses(conflict).Create(data).Error
}

func UpsertByColumns(ctx context.Context, db *DB, data any, columns []clause.Column, updateColumns []string) error {
	return Upsert(ctx, db, data, columns, updateColumns)
}

func UpsertByColumnNames(ctx context.Context, db *DB, data any, columnNames []string, updateColumns []string) error {
	return Upsert(ctx, db, data, Columns(columnNames...), updateColumns)
}

func CreateRecord(ctx context.Context, db *DB, data any) error {
	if db == nil {
		return errors.New("gormx db is nil")
	}
	return db.WithContext(ctx).Create(data).Error
}

func Create(ctx context.Context, db *DB, data any) error {
	return CreateRecord(ctx, db, data)
}

func GormDB(db *DB) (*gorm.DB, error) {
	if db == nil || db.DB == nil {
		return nil, errors.New("gormx db is nil")
	}
	return db.DB, nil
}
