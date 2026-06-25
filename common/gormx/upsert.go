package gormx

import (
	"errors"

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

func Upsert(db *DB, data any, columns []clause.Column, updateColumns []string) error {
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
	return db.Clauses(conflict).Create(data).Error
}
