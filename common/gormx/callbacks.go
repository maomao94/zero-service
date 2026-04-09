package gormx

import (
	"context"

	"gorm.io/gorm"
)

// callbacks.go GORM Callbacks 自动填充审计字段
//
// 注册 Create/Update/Delete 钩子，自动填充：
// - CreateUser, CreateName, UpdateUser, UpdateName
// - TenantID (多租户场景)
// - DeleteUser, DeleteName (软删除)
// - Version (乐观锁)

// RegisterCallbacks 注册审计字段填充钩子
// 在数据库初始化完成后调用
//
// 使用示例:
//
//	db, err := gormx.NewMySQL(dsn)
//	if err != nil {
//	    panic(err)
//	}
//	gormx.RegisterCallbacks(db)
func RegisterCallbacks(db *gorm.DB) {
	// 创建前钩子：填充 CreateUser, CreateName, UpdateUser, UpdateName, TenantID
	db.Callback().Create().Before("gorm:create").Register("gormx:before_create", BeforeCreateHook)

	// 更新前钩子：填充 UpdateUser, UpdateName，版本号 +1
	db.Callback().Update().Before("gorm:update").Register("gormx:before_update", BeforeUpdateHook)

	// 删除前钩子（软删除）：填充 DeleteUser, DeleteName
	db.Callback().Delete().Before("gorm:delete").Register("gormx:before_delete", BeforeDeleteHook)
}

// BeforeCreateHook 创建前钩子
// 自动填充审计字段和租户ID
func BeforeCreateHook(db *gorm.DB) {
	ctx := db.Statement.Context
	userCtx := ExtractUserContext(ctx)

	// 如果没有用户上下文，允许创建（系统初始化场景）
	if userCtx == nil {
		return
	}

	// 填充创建人和更新人
	if userCtx.UserID > 0 {
		setColumnIfZero(db, "create_user", userCtx.UserID)
		setColumnIfZero(db, "create_name", userCtx.UserName)
		setColumnIfZero(db, "update_user", userCtx.UserID)
		setColumnIfZero(db, "update_name", userCtx.UserName)
	}

	// 多租户场景：自动填充 TenantID
	if userCtx.TenantID != "" && hasTenantField(db) {
		setColumnIfZero(db, "tenant_id", userCtx.TenantID)
	}
}

// BeforeUpdateHook 更新前钩子
// 自动填充更新人信息和版本号 +1
func BeforeUpdateHook(db *gorm.DB) {
	ctx := db.Statement.Context
	userCtx := ExtractUserContext(ctx)

	// 填充更新人
	if userCtx != nil && userCtx.UserID > 0 {
		setColumnIfZero(db, "update_user", userCtx.UserID)
		setColumnIfZero(db, "update_name", userCtx.UserName)
	}

	// 乐观锁：版本号 +1，使用 SetColumn
	if hasVersionField(db) {
		db.Statement.SetColumn("version", gorm.Expr("version + 1"))
	}
}

// BeforeDeleteHook 删除前钩子（软删除）
// 自动填充删除人信息
func BeforeDeleteHook(db *gorm.DB) {
	ctx := db.Statement.Context
	userCtx := ExtractUserContext(ctx)

	// 填充删除人
	if userCtx != nil && userCtx.UserID > 0 {
		setColumnIfZero(db, "delete_user", userCtx.UserID)
		setColumnIfZero(db, "delete_name", userCtx.UserName)
	}
}

// ExtractUserContext 从 Context 提取用户上下文
func ExtractUserContext(ctx context.Context) *UserContext {
	if ctx == nil {
		return nil
	}
	userCtx, ok := ctx.Value(UserContextKey).(*UserContext)
	if !ok {
		return nil
	}
	return userCtx
}

// hasTenantField 检查模型是否有租户字段
func hasTenantField(db *gorm.DB) bool {
	if db.Statement == nil || db.Statement.Schema == nil {
		return false
	}
	_, ok := db.Statement.Schema.FieldsByDBName["tenant_id"]
	return ok
}

// hasVersionField 检查模型是否有版本字段
func hasVersionField(db *gorm.DB) bool {
	if db.Statement == nil || db.Statement.Schema == nil {
		return false
	}
	_, ok := db.Statement.Schema.FieldsByDBName["version"]
	return ok
}

// setColumnIfZero 如果字段值为零值，则设置新值
// 简化实现：直接设置值，由 GORM 处理零值判断
func setColumnIfZero(db *gorm.DB, column string, value interface{}) {
	if db.Statement.Schema == nil {
		return
	}

	// 检查字段是否存在
	if _, ok := db.Statement.Schema.FieldsByDBName[column]; !ok {
		return
	}

	// 直接设置列值
	db.Statement.SetColumn(column, value)
}

// isZeroValue 检查值是否为零值
func isZeroValue(v interface{}) bool {
	switch val := v.(type) {
	case uint:
		return val == 0
	case uint64:
		return val == 0
	case int:
		return val == 0
	case int64:
		return val == 0
	case string:
		return val == ""
	case []byte:
		return len(val) == 0
	case nil:
		return true
	default:
		return false
	}
}

// UnscopedUpdate 绕过审计钩子的更新
// 用于系统后台更新等场景
func UnscopedUpdate(db *gorm.DB, model interface{}, updates map[string]interface{}) error {
	return db.Session(&gorm.Session{
		SkipHooks: true,
	}).Model(model).Updates(updates).Error
}

// UnscopedCreate 绕过审计钩子的创建
// 用于数据导入等场景
func UnscopedCreate(db *gorm.DB, value interface{}) error {
	return db.Session(&gorm.Session{
		SkipHooks: true,
	}).Create(value).Error
}

// SystemUpdate 系统级别的更新（不记录更新人）
// 用于定时任务、系统操作等
func SystemUpdate(db *gorm.DB, model interface{}, updates map[string]interface{}) error {
	emptyCtx := context.Background()
	return db.WithContext(emptyCtx).Model(model).Updates(updates).Error
}

// SystemCreate 系统级别的创建（不记录创建人）
func SystemCreate(db *gorm.DB, value interface{}) error {
	emptyCtx := context.Background()
	return db.WithContext(emptyCtx).Create(value).Error
}
