package gormx

import (
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// AuditFieldDef 审计字段定义
type AuditFieldDef struct {
	ColumnName string
	Definition string
	Comment    string
}

// TenantFieldDef 租户字段定义
type TenantFieldDef struct {
	ColumnName string
	Definition string
	Comment    string
}

// 审计字段列表
var AuditFieldDefs = []AuditFieldDef{
	{
		ColumnName: "create_user",
		Definition: "bigint unsigned NOT NULL DEFAULT 0",
		Comment:    "创建人ID",
	},
	{
		ColumnName: "create_name",
		Definition: "varchar(64) NOT NULL DEFAULT ''",
		Comment:    "创建人姓名",
	},
	{
		ColumnName: "update_user",
		Definition: "bigint unsigned NOT NULL DEFAULT 0",
		Comment:    "更新人ID",
	},
	{
		ColumnName: "update_name",
		Definition: "varchar(64) NOT NULL DEFAULT ''",
		Comment:    "更新人姓名",
	},
	{
		ColumnName: "delete_user",
		Definition: "bigint unsigned NOT NULL DEFAULT 0",
		Comment:    "删除人ID",
	},
	{
		ColumnName: "delete_name",
		Definition: "varchar(64) NOT NULL DEFAULT ''",
		Comment:    "删除人姓名",
	},
	{
		ColumnName: "version",
		Definition: "bigint NOT NULL DEFAULT 0",
		Comment:    "版本号，用于乐观锁",
	},
}

// TenantField 租户字段定义
var TenantField = struct {
	ColumnName string
	Definition string
	Comment    string
}{
	ColumnName: "tenant_id",
	Definition: "varchar(12) NOT NULL DEFAULT '000000'",
	Comment:    "租户ID",
}

// MigrateAuditFields 为现有表添加审计字段
//
// 使用示例：
//
//	db, _ := gormx.NewMySQL(dsn)
//	err := gormx.MigrateAuditFields(db, "user_config")
func MigrateAuditFields(db *gorm.DB, tableName string) error {
	for _, field := range AuditFieldDefs {
		if err := addColumnIfNotExists(db, tableName, field.ColumnName, field.Definition, field.Comment); err != nil {
			return fmt.Errorf("add column %s failed: %w", field.ColumnName, err)
		}
	}
	logx.Infof("migrate audit fields for table %s success", tableName)
	return nil
}

// MigrateTenantField 为现有表添加租户字段
//
// 使用示例：
//
//	db, _ := gormx.NewMySQL(dsn)
//	err := gormx.MigrateTenantField(db, "user_config")
func MigrateTenantField(db *gorm.DB, tableName string) error {
	field := TenantField
	if err := addColumnIfNotExists(db, tableName, field.ColumnName, field.Definition, field.Comment); err != nil {
		return fmt.Errorf("add column %s failed: %w", field.ColumnName, err)
	}
	logx.Infof("migrate tenant field for table %s success", tableName)
	return nil
}

// MigrateAllFields 为现有表添加所有审计和租户字段
//
// 使用示例：
//
//	db, _ := gormx.NewMySQL(dsn)
//	err := gormx.MigrateAllFields(db, "user_config")
func MigrateAllFields(db *gorm.DB, tableName string) error {
	// 先添加租户字段
	if err := MigrateTenantField(db, tableName); err != nil {
		return err
	}

	// 再添加审计字段
	if err := MigrateAuditFields(db, tableName); err != nil {
		return err
	}

	logx.Infof("migrate all fields for table %s success", tableName)
	return nil
}

// MigrateAuditIndexes 为审计字段添加索引
//
// 使用示例：
//
//	db, _ := gormx.NewMySQL(dsn)
//	err := gormx.MigrateAuditIndexes(db, "user_config")
func MigrateAuditIndexes(db *gorm.DB, tableName string) error {
	indexes := []struct {
		indexName string
		columns   string
	}{
		{"idx_create_user", "create_user"},
		{"idx_update_user", "update_user"},
		{"idx_delete_user", "delete_user"},
	}

	for _, idx := range indexes {
		if err := addIndexIfNotExists(db, tableName, idx.indexName, idx.columns); err != nil {
			return fmt.Errorf("add index %s failed: %w", idx.indexName, err)
		}
	}

	logx.Infof("migrate audit indexes for table %s success", tableName)
	return nil
}

// MigrateTenantIndex 为租户字段添加索引
//
// 使用示例：
//
//	db, _ := gormx.NewMySQL(dsn)
//	err := gormx.MigrateTenantIndex(db, "user_config")
func MigrateTenantIndex(db *gorm.DB, tableName string) error {
	if err := addIndexIfNotExists(db, tableName, "idx_tenant_id", "tenant_id"); err != nil {
		return fmt.Errorf("add tenant index failed: %w", err)
	}
	logx.Infof("migrate tenant index for table %s success", tableName)
	return nil
}

// MigrateAllIndexes 添加所有索引
func MigrateAllIndexes(db *gorm.DB, tableName string) error {
	if err := MigrateTenantIndex(db, tableName); err != nil {
		return err
	}
	return MigrateAuditIndexes(db, tableName)
}

// addColumnIfNotExists 添加列（如果不存在）
func addColumnIfNotExists(db *gorm.DB, tableName, columnName, definition, comment string) error {
	// 检查列是否存在
	var count int64
	err := db.Raw(fmt.Sprintf(
		"SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = '%s' AND column_name = '%s'",
		tableName, columnName,
	)).Scan(&count).Error
	if err != nil {
		return err
	}

	// 列已存在，跳过
	if count > 0 {
		logx.Infof("column %s.%s already exists, skip", tableName, columnName)
		return nil
	}

	// 添加列
	sql := fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s COMMENT '%s'", tableName, columnName, definition, comment)
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	logx.Infof("add column %s.%s success", tableName, columnName)
	return nil
}

// addIndexIfNotExists 添加索引（如果不存在）
func addIndexIfNotExists(db *gorm.DB, tableName, indexName, columns string) error {
	// 检查索引是否存在
	var count int64
	err := db.Raw(fmt.Sprintf(
		"SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = '%s' AND index_name = '%s'",
		tableName, indexName,
	)).Scan(&count).Error
	if err != nil {
		return err
	}

	// 索引已存在，跳过
	if count > 0 {
		logx.Infof("index %s on %s already exists, skip", indexName, tableName)
		return nil
	}

	// 添加索引
	sql := fmt.Sprintf("ALTER TABLE `%s` ADD INDEX `%s` (`%s`)", tableName, indexName, columns)
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	logx.Infof("add index %s on %s success", indexName, tableName)
	return nil
}

// MigrateAll 为表添加所有字段和索引的完整迁移
//
// 使用示例：
//
//	db, _ := gormx.NewMySQL(dsn)
//	err := gormx.MigrateAll(db, "user_config")
func MigrateAll(db *gorm.DB, tableName string) error {
	if err := MigrateAllFields(db, tableName); err != nil {
		return err
	}
	return MigrateAllIndexes(db, tableName)
}

// MigrateBatch 为多个表批量添加审计字段
//
// 使用示例：
//
//	db, _ := gormx.NewMySQL(dsn)
//	tables := []string{"user_config", "system_config", "app_settings"}
//	results := gormx.MigrateBatch(db, tables)
//	for _, r := range results {
//	    if r.Err != nil {
//	        logx.Errorf("migrate %s failed: %v", r.Table, r.Err)
//	    }
//	}
func MigrateBatch(db *gorm.DB, tableNames []string) []MigrateResult {
	results := make([]MigrateResult, len(tableNames))
	for i, table := range tableNames {
		err := MigrateAll(db, table)
		results[i] = MigrateResult{
			Table: table,
			Err:   err,
		}
	}
	return results
}

// MigrateResult 迁移结果
type MigrateResult struct {
	Table string
	Err   error
}

// GenerateMigrationSQL 生成迁移 SQL 语句（用于手动执行）
//
// 使用示例：
//
//	sqls := gormx.GenerateMigrationSQL("user_config")
//	for _, sql := range sqls {
//	    fmt.Println(sql)
//	}
func GenerateMigrationSQL(tableName string) []string {
	var sqls []string

	// 添加租户字段
	sqls = append(sqls, fmt.Sprintf(
		"ALTER TABLE `%s` ADD COLUMN `tenant_id` varchar(12) NOT NULL DEFAULT '000000' COMMENT '租户ID' AFTER `id`;",
		tableName,
	))

	// 添加审计字段
	auditSQLs := []string{
		"ALTER TABLE `%s` ADD COLUMN `create_user` bigint unsigned NOT NULL DEFAULT 0 COMMENT '创建人ID' AFTER `tenant_id`;",
		"ALTER TABLE `%s` ADD COLUMN `create_name` varchar(64) NOT NULL DEFAULT '' COMMENT '创建人姓名' AFTER `create_user`;",
		"ALTER TABLE `%s` ADD COLUMN `update_user` bigint unsigned NOT NULL DEFAULT 0 COMMENT '更新人ID' AFTER `create_name`;",
		"ALTER TABLE `%s` ADD COLUMN `update_name` varchar(64) NOT NULL DEFAULT '' COMMENT '更新人姓名' AFTER `update_user`;",
		"ALTER TABLE `%s` ADD COLUMN `delete_user` bigint unsigned NOT NULL DEFAULT 0 COMMENT '删除人ID' AFTER `update_name`;",
		"ALTER TABLE `%s` ADD COLUMN `delete_name` varchar(64) NOT NULL DEFAULT '' COMMENT '删除人姓名' AFTER `delete_user`;",
		"ALTER TABLE `%s` ADD COLUMN `version` bigint NOT NULL DEFAULT 0 COMMENT '版本号，用于乐观锁' AFTER `delete_name`;",
	}

	for _, sql := range auditSQLs {
		sqls = append(sqls, fmt.Sprintf(sql, tableName))
	}

	// 添加索引
	indexSQLs := []string{
		fmt.Sprintf("ALTER TABLE `%s` ADD INDEX `idx_tenant_id` (`tenant_id`);", tableName),
		fmt.Sprintf("ALTER TABLE `%s` ADD INDEX `idx_create_user` (`create_user`);", tableName),
		fmt.Sprintf("ALTER TABLE `%s` ADD INDEX `idx_update_user` (`update_user`);", tableName),
		fmt.Sprintf("ALTER TABLE `%s` ADD INDEX `idx_delete_user` (`delete_user`);", tableName),
	}
	sqls = append(sqls, indexSQLs...)

	return sqls
}
