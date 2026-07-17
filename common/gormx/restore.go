package gormx

import "gorm.io/gorm"

func Restore(db *gorm.DB, model any, conds ...any) error {
	q := db.Unscoped().Model(model)
	if len(conds) > 0 {
		q = q.Where(conds[0], conds[1:]...)
	}
	if updates := restoreDeleteFieldUpdates(q, model); len(updates) > 0 {
		return restoreDeleteFields(q, updates)
	}
	return q.Update("deleted_at", nil).Error
}

func RestoreWithTenant(db *gorm.DB, model any, conds ...any) error {
	q := withTenantQueryFromDB(db.Unscoped().Model(model))
	if len(conds) > 0 {
		q = q.Where(conds[0], conds[1:]...)
	}
	if updates := restoreDeleteFieldUpdates(q, model); len(updates) > 0 {
		return restoreDeleteFields(q, updates)
	}
	return q.Update("deleted_at", nil).Error
}

func restoreDeleteFields(db *gorm.DB, updates map[string]any) error {
	return db.Select(mapKeys(updates)).Updates(updates).Error
}

func restoreDeleteFieldUpdates(db *gorm.DB, model any) map[string]any {
	if db == nil || model == nil {
		return nil
	}
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		return nil
	}
	updates := make(map[string]any)
	if field, ok := stmt.Schema.FieldsByDBName["delete_time"]; ok {
		updates[field.DBName] = nil
	}
	if field, ok := stmt.Schema.FieldsByDBName["del_state"]; ok {
		updates[field.DBName] = zeroValue(field.FieldType)
	}
	if field, ok := stmt.Schema.FieldsByDBName["is_deleted"]; ok {
		updates[field.DBName] = zeroValue(field.FieldType)
	}
	return updates
}

func hasLegacyDeleteFields(db *gorm.DB, model any) bool {
	return len(restoreDeleteFieldUpdates(db, model)) > 0
}
