package gormx

import (
	"context"

	"gorm.io/gorm"
)

func SoftDelete(db *gorm.DB, model any, conds ...any) error {
	return db.Delete(model, conds...).Error
}

func UnscopedDelete(db *gorm.DB, model any) error {
	return db.Unscoped().Delete(model).Error
}

func UnscopedDeleteWithTenant(ctx context.Context, db *gorm.DB, model any, conds ...any) error {
	q := withTenantQuery(ctx, db.WithContext(ctx).Unscoped().Model(model))
	if len(conds) > 0 {
		q = q.Where(conds[0], conds[1:]...)
	}
	return q.Delete(model).Error
}
