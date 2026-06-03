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

func UpdateOrCreate(ctx context.Context, db *DB, model any, where map[string]any, createData any, updateData map[string]any) error {
	if db == nil {
		return errors.New("gormx db is nil")
	}
	if model == nil {
		return errors.New("gormx update or create model is nil")
	}
	if createData == nil {
		return errors.New("gormx update or create data is nil")
	}
	if len(where) == 0 {
		return errors.New("gormx update or create where is empty")
	}

	tx := db.WithContext(ctx).Model(model).Where(where).Updates(updateData)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected > 0 {
		return nil
	}

	if err := db.WithContext(ctx).Create(createData).Error; err == nil {
		return nil
	} else {
		tx = db.WithContext(ctx).Model(model).Where(where).Updates(updateData)
		if tx.Error != nil {
			return tx.Error
		}
		if tx.RowsAffected > 0 {
			return nil
		}
		return err
	}
}

func CreateRecord(ctx context.Context, db *DB, data any) error {
	if db == nil {
		return errors.New("gormx db is nil")
	}
	return db.WithContext(ctx).Create(data).Error
}

func GormDB(db *DB) (*gorm.DB, error) {
	if db == nil || db.DB == nil {
		return nil, errors.New("gormx db is nil")
	}
	return db.DB, nil
}
