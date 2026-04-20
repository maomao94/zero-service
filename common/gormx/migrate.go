package gormx

import (
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

func (db *DB) MigrateLegacyFields(tableName string) error {
	fields := []struct {
		name    string
		colType string
		comment string
	}{
		{"id", "bigint NOT NULL AUTO_INCREMENT", "主键ID"},
		{"del_state", "bigint NOT NULL DEFAULT 0", "删除状态"},
		{"version", "bigint NOT NULL DEFAULT 0", "乐观锁版本号"},
	}
	for _, f := range fields {
		if err := addColumnIfNotExists(db.DB, tableName, f.name, f.colType, f.comment); err != nil {
			return err
		}
	}
	if err := addIndexIfNotExists(db.DB, tableName, "idx_del_state", "del_state"); err != nil {
		return err
	}
	logx.Infof("migrate legacy fields for table %s success", tableName)
	return nil
}

func (db *DB) MigrateAuditFields(tableName string) error {
	fields := []struct {
		name    string
		colType string
		comment string
	}{
		{"create_user", "bigint unsigned NOT NULL DEFAULT 0", "创建人ID"},
		{"create_name", "varchar(64) NOT NULL DEFAULT ''", "创建人姓名"},
		{"update_user", "bigint unsigned NOT NULL DEFAULT 0", "更新人ID"},
		{"update_name", "varchar(64) NOT NULL DEFAULT ''", "更新人姓名"},
	}
	for _, f := range fields {
		if err := addColumnIfNotExists(db.DB, tableName, f.name, f.colType, f.comment); err != nil {
			return err
		}
	}
	indexes := []struct {
		name    string
		columns string
	}{
		{"idx_create_user", "create_user"},
		{"idx_update_user", "update_user"},
	}
	for _, idx := range indexes {
		if err := addIndexIfNotExists(db.DB, tableName, idx.name, idx.columns); err != nil {
			return err
		}
	}
	logx.Infof("migrate audit fields for table %s success", tableName)
	return nil
}

func (db *DB) MigrateTenantField(tableName string) error {
	if err := addColumnIfNotExists(db.DB, tableName, "tenant_id", "varchar(12) NOT NULL DEFAULT '000000'", "租户ID"); err != nil {
		return err
	}
	if err := addIndexIfNotExists(db.DB, tableName, "idx_tenant_id", "tenant_id"); err != nil {
		return err
	}
	logx.Infof("migrate tenant field for table %s success", tableName)
	return nil
}

func (db *DB) MigrateAllFields(tableName string) error {
	if err := db.MigrateTenantField(tableName); err != nil {
		return err
	}
	return db.MigrateAuditFields(tableName)
}

func (db *DB) MigrateBatch(tableNames []string, opts ...func(table string) error) {
	for _, table := range tableNames {
		for _, opt := range opts {
			if err := opt(table); err != nil {
				logx.Errorf("migrate %s failed: %v", table, err)
			}
		}
	}
}

func addColumnIfNotExists(db *gorm.DB, tableName, columnName, colType, comment string) error {
	if db.Migrator().HasColumn(tableName, columnName) {
		logx.Infof("column %s.%s already exists, skip", tableName, columnName)
		return nil
	}
	sql := fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s COMMENT '%s'", tableName, columnName, colType, comment)
	if err := db.Exec(sql).Error; err != nil {
		return err
	}
	logx.Infof("add column %s.%s success", tableName, columnName)
	return nil
}

func addIndexIfNotExists(db *gorm.DB, tableName, indexName, columns string) error {
	if db.Migrator().HasIndex(tableName, indexName) {
		logx.Infof("index %s on %s already exists, skip", indexName, tableName)
		return nil
	}
	sql := fmt.Sprintf("ALTER TABLE `%s` ADD INDEX `%s` (`%s`)", tableName, indexName, columns)
	if err := db.Exec(sql).Error; err != nil {
		return err
	}
	logx.Infof("add index %s on %s success", indexName, tableName)
	return nil
}
