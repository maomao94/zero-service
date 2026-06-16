package gormx

import (
	"context"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type DB struct {
	*gorm.DB
}

func (db *DB) WithContext(ctx context.Context) *DB {
	return &DB{DB: db.DB.WithContext(ctx)}
}

func (db *DB) ExplainSQL(queryFn func(tx *gorm.DB) *gorm.DB) string {
	return db.DB.ToSQL(queryFn)
}

func (db *DB) Transact(fn func(tx *DB) error) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		return fn(&DB{DB: tx})
	})
}

func (db *DB) WithTenant(ctx context.Context) *gorm.DB {
	return db.WithContext(ctx).DB.Scopes(TenantScope(ctx))
}

func (db *DB) WithTenantStrict(ctx context.Context) *gorm.DB {
	return db.WithContext(ctx).DB.Scopes(TenantScopeStrict(ctx))
}

func (db *DB) WithDeleted(ctx context.Context) *gorm.DB {
	return db.WithContext(ctx).DB.Unscoped()
}

func (db *DB) WithTenantDeleted(ctx context.Context) *gorm.DB {
	return db.WithContext(ctx).DB.Scopes(TenantScopeWithDelete(ctx))
}

func (db *DB) AutoMigrate(dst ...any) error {
	if len(dst) == 0 {
		return nil
	}
	if err := db.DB.Session(&gorm.Session{Logger: QuietGormLogger()}).AutoMigrate(dst...); err != nil {
		return err
	}
	logx.Infof("auto migrate %d tables success", len(dst))
	return nil
}

func (db *DB) MustAutoMigrate(dst ...any) {
	if err := db.AutoMigrate(dst...); err != nil {
		logx.Must(errors.Errorf("auto migrate failed: %v", err))
	}
}
