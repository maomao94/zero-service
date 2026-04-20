package gormx

import (
	"fmt"

	"gorm.io/gorm"
)

// ColumnInfo 列信息
type ColumnInfo struct {
	ColumnName    string `json:"column_name"`
	DataType      string `json:"data_type"`
	DataTypeLong  string `json:"data_type_long"`
	ColumnComment string `json:"column_comment"`
	PrimaryKey    bool   `json:"primary_key"`
	AutoIncrement bool   `json:"auto_increment"`
	Nullable      bool   `json:"nullable"`
	DefaultValue  string `json:"default_value"`
}

// TableInfo 表信息
type TableInfo struct {
	TableName string `json:"table_name"`
}

// DBInfo 数据库信息
type DBInfo struct {
	Database string `json:"database"`
}

// GetTables 获取数据库中的表列表
func GetTables(db *gorm.DB, dbName string) ([]TableInfo, error) {
	var tables []TableInfo
	sql := buildGetTablesSQL(db)
	if err := db.Raw(sql, dbName).Scan(&tables).Error; err != nil {
		return nil, err
	}
	return tables, nil
}

// GetColumns 获取表的列信息
func GetColumns(db *gorm.DB, tableName string) ([]ColumnInfo, error) {
	var columns []ColumnInfo

	// SQLite 不支持参数化 PRAGMA，需要动态构建
	if GetDatabaseTypeFromDialector(db) == DatabaseSQLite {
		sql := fmt.Sprintf(`SELECT
			name AS column_name,
			type AS data_type,
			dflt_value AS default_value,
			'notnull' IN (sql) AS nullable
		FROM pragma_table_info('%s')`, tableName)
		if err := db.Raw(sql).Scan(&columns).Error; err != nil {
			return nil, err
		}
		return columns, nil
	}

	sql := buildGetColumnsSQL(db)
	if err := db.Raw(sql, tableName).Scan(&columns).Error; err != nil {
		return nil, err
	}
	return columns, nil
}

// GetDatabases 获取所有数据库
func GetDatabases(db *gorm.DB) ([]DBInfo, error) {
	var dbs []DBInfo
	sql := buildGetDatabasesSQL(db)
	if err := db.Raw(sql).Scan(&dbs).Error; err != nil {
		return nil, err
	}
	return dbs, nil
}

// CreateTable 创建表
func CreateTable(db *gorm.DB, model any) error {
	return db.AutoMigrate(model)
}

// DropTable 删除表
func DropTable(db *gorm.DB, tableName string) error {
	return db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)).Error
}

// HasTable 检查表是否存在
func HasTable(db *gorm.DB, tableName string) bool {
	return db.Migrator().HasTable(tableName)
}

// ============ 私有方法：根据数据库类型构建 SQL ============

func buildGetTablesSQL(db *gorm.DB) string {
	switch GetDatabaseTypeFromDialector(db) {
	case DatabasePostgres:
		return `SELECT tablename AS table_name FROM pg_catalog.pg_tables WHERE schemaname = 'public'`
	case DatabaseSQLite:
		return `SELECT name AS table_name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`
	default: // MySQL
		return "SELECT TABLE_NAME AS table_name FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE'"
	}
}

func buildGetColumnsSQL(db *gorm.DB) string {
	switch GetDatabaseTypeFromDialector(db) {
	case DatabasePostgres:
		return `
			SELECT
				c.column_name AS column_name,
				c.data_type AS data_type,
				COALESCE(c.character_maximum_length, '') AS data_type_long,
				COALESCE(pd.description, '') AS column_comment,
				CASE WHEN pk.column_name IS NOT NULL THEN true ELSE false END AS primary_key,
				CASE WHEN c.column_default LIKE 'nextval%' THEN true ELSE false END AS auto_increment,
				CASE WHEN c.is_nullable = 'YES' THEN true ELSE false END AS nullable,
				COALESCE(c.column_default, '') AS default_value
			FROM information_schema.columns c
			LEFT JOIN (
				SELECT kcu.column_name
				FROM information_schema.table_constraints tc
				JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
				WHERE tc.table_name = $1 AND tc.constraint_type = 'PRIMARY KEY'
			) pk ON c.column_name = pk.column_name
			LEFT JOIN pg_catalog.pg_statio_all_descriptions pd ON pd.objoid = (SELECT oid FROM pg_class WHERE relname = $1)
			WHERE c.table_name = $1 AND c.table_schema = 'public'
			ORDER BY c.ordinal_position`
	default: // MySQL
		return `
			SELECT
				c.COLUMN_NAME AS column_name,
				c.DATA_TYPE AS data_type,
				COALESCE(c.CHARACTER_MAXIMUM_LENGTH, '') AS data_type_long,
				c.COLUMN_COMMENT AS column_comment,
				CASE WHEN kcu.COLUMN_NAME IS NOT NULL THEN true ELSE false END AS primary_key,
				CASE WHEN c.EXTRA = 'auto_increment' THEN true ELSE false END AS auto_increment,
				CASE WHEN c.IS_NULLABLE = 'YES' THEN true ELSE false END AS nullable,
				COALESCE(c.COLUMN_DEFAULT, '') AS default_value
			FROM INFORMATION_SCHEMA.COLUMNS c
			LEFT JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu
				ON c.TABLE_SCHEMA = kcu.TABLE_SCHEMA
				AND c.TABLE_NAME = kcu.TABLE_NAME
				AND c.COLUMN_NAME = kcu.COLUMN_NAME
				AND kcu.CONSTRAINT_NAME = 'PRIMARY'
			WHERE c.TABLE_NAME = ? AND c.TABLE_SCHEMA = DATABASE()
			ORDER BY c.ORDINAL_POSITION`
	}
}

func buildGetDatabasesSQL(db *gorm.DB) string {
	switch GetDatabaseTypeFromDialector(db) {
	case DatabasePostgres:
		return "SELECT datname AS database FROM pg_catalog.pg_database WHERE datistemplate = false"
	case DatabaseSQLite:
		return "SELECT 'main' AS database"
	default: // MySQL
		return "SELECT SCHEMA_NAME AS database FROM INFORMATION_SCHEMA.SCHEMATA"
	}
}
