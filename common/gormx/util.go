package gormx

import (
	"reflect"

	"gorm.io/gorm"
)

func setSchemaColumn(db *gorm.DB, column string, value any) {
	if db.Statement.Schema == nil {
		return
	}
	if _, ok := db.Statement.Schema.FieldsByDBName[column]; !ok {
		return
	}
	db.Statement.SetColumn(column, value)
}

func mapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

func zeroValue(fieldType reflect.Type) any {
	return reflect.Zero(fieldType).Interface()
}
