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
	return UpsertByColumns(ctx, db, data, columns, updateColumns)
}

func UpsertByColumns(ctx context.Context, db *DB, data any, columns []clause.Column, updateColumns []string) error {
	if db == nil {
		return errors.New("gormx db is nil")
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   columns,
		DoUpdates: clause.AssignmentColumns(updateColumns),
	}).Create(data).Error
}

func UpsertByColumnNames(ctx context.Context, db *DB, data any, columnNames []string, updateColumns []string) error {
	return UpsertByColumns(ctx, db, data, Columns(columnNames...), updateColumns)
}

func UpsertWithDeletedAt(ctx context.Context, db *DB, data any, columns []clause.Column, updateColumns []string) error {
	return UpsertByColumnsWithDeletedAt(ctx, db, data, columns, updateColumns)
}

func UpsertByColumnsWithDeletedAt(ctx context.Context, db *DB, data any, columns []clause.Column, updateColumns []string) error {
	if db == nil {
		return errors.New("gormx db is nil")
	}
	assignments := clause.AssignmentColumns(updateColumns)
	assignments = append(assignments, clause.Assignment{Column: clause.Column{Name: "deleted_at"}, Value: nil})
	return db.WithContext(ctx).Unscoped().Clauses(clause.OnConflict{
		Columns:   columns,
		DoUpdates: assignments,
	}).Create(data).Error
}

func UpsertWithLegacyDelete(ctx context.Context, db *DB, data any, columns []clause.Column, updateColumns []string) error {
	return UpsertByColumnsWithLegacyDelete(ctx, db, data, columns, updateColumns)
}

func UpsertByColumnsWithLegacyDelete(ctx context.Context, db *DB, data any, columns []clause.Column, updateColumns []string) error {
	if db == nil {
		return errors.New("gormx db is nil")
	}
	assignments := clause.AssignmentColumns(updateColumns)
	assignments = append(assignments,
		clause.Assignment{Column: clause.Column{Name: "delete_time"}, Value: nil},
		clause.Assignment{Column: clause.Column{Name: "del_state"}, Value: int64(0)},
	)
	return db.WithContext(ctx).Unscoped().Clauses(clause.OnConflict{
		Columns:   columns,
		DoUpdates: assignments,
	}).Create(data).Error
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
