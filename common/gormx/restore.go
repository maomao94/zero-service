package gormx

import (
	"context"

	"gorm.io/gorm"
)

func Restore(db *gorm.DB, model any, conds ...any) error {
	q := db.Unscoped().Model(model)
	if len(conds) > 0 {
		q = q.Where(conds[0], conds[1:]...)
	}
	if hasLegacyDeleteFields(q, model) {
		return q.Select("delete_time", "del_state").Updates(map[string]any{
			"delete_time": nil,
			"del_state":   int64(0),
		}).Error
	}
	return q.Update("deleted_at", nil).Error
}

func RestoreWithTenant(ctx context.Context, db *gorm.DB, model any, conds ...any) error {
	q := withTenantQuery(ctx, db.WithContext(ctx).Unscoped().Model(model))
	if len(conds) > 0 {
		q = q.Where(conds[0], conds[1:]...)
	}
	if hasLegacyDeleteFields(q, model) {
		return q.Select("delete_time", "del_state").Updates(map[string]any{
			"delete_time": nil,
			"del_state":   int64(0),
		}).Error
	}
	return q.Update("deleted_at", nil).Error
}

func hasLegacyDeleteFields(db *gorm.DB, model any) bool {
	if db == nil || model == nil {
		return false
	}
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		return false
	}
	_, hasDeleteTime := stmt.Schema.FieldsByDBName["delete_time"]
	_, hasDelState := stmt.Schema.FieldsByDBName["del_state"]
	return hasDeleteTime && hasDelState
}
