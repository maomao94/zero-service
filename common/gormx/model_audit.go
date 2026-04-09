package gormx

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel 标准模型基类（无租户）
//
// 包含所有通用审计字段，适合不需要多租户隔离的系统表。
// GORM Callbacks 会自动填充审计字段。
//
// 使用示例：
//
//	type User struct {
//	    gormx.BaseModel
//	    Username string
//	    Email   string
//	}
type BaseModel struct {
	// === 主键 ===
	ID uint `gorm:"primarykey" json:"id"`

	// === 审计字段 ===
	CreateUser uint   `gorm:"index" json:"create_user"`   // 创建人ID
	CreateName string `gorm:"size:64" json:"create_name"` // 创建人姓名
	UpdateUser uint   `gorm:"index" json:"update_user"`   // 更新人ID
	UpdateName string `gorm:"size:64" json:"update_name"` // 更新人姓名
	DeleteUser uint   `gorm:"index" json:"delete_user"`   // 删除人ID
	DeleteName string `gorm:"size:64" json:"delete_name"` // 删除人姓名

	// === 乐观锁 ===
	Version int64 `gorm:"default:0" json:"version"` // 版本号，用于乐观锁

	// === 时间戳 ===
	CreatedAt time.Time      `json:"created_at"`     // 创建时间
	UpdatedAt time.Time      `json:"updated_at"`     // 更新时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}

// IntBaseModel int 主键的标准模型
//
// 使用场景：需要与其他系统对接时使用 int 类型 ID
type IntBaseModel struct {
	ID uint `gorm:"primarykey" json:"id"`

	// === 审计字段 ===
	CreateUser uint   `gorm:"index" json:"create_user"`
	CreateName string `gorm:"size:64" json:"create_name"`
	UpdateUser uint   `gorm:"index" json:"update_user"`
	UpdateName string `gorm:"size:64" json:"update_name"`
	DeleteUser uint   `gorm:"index" json:"delete_user"`
	DeleteName string `gorm:"size:64" json:"delete_name"`

	// === 乐观锁 ===
	Version int64 `gorm:"default:0" json:"version"`

	// === 时间戳 ===
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// StringBaseModel string 主键的标准模型
//
// 使用场景：需要字符串主键（如 UUID）
type StringBaseModel struct {
	ID string `gorm:"primarykey;size:36" json:"id"`

	// === 审计字段 ===
	CreateUser uint   `gorm:"index" json:"create_user"`
	CreateName string `gorm:"size:64" json:"create_name"`
	UpdateUser uint   `gorm:"index" json:"update_user"`
	UpdateName string `gorm:"size:64" json:"update_name"`
	DeleteUser uint   `gorm:"index" json:"delete_user"`
	DeleteName string `gorm:"size:64" json:"delete_name"`

	// === 乐观锁 ===
	Version int64 `gorm:"default:0" json:"version"`

	// === 时间戳 ===
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TimeBaseModel 仅时间戳的标准模型（无软删除）
//
// 使用场景：不需要软删除的表
type TimeBaseModel struct {
	ID uint `gorm:"primarykey" json:"id"`

	// === 审计字段 ===
	CreateUser uint   `gorm:"index" json:"create_user"`
	CreateName string `gorm:"size:64" json:"create_name"`
	UpdateUser uint   `gorm:"index" json:"update_user"`
	UpdateName string `gorm:"size:64" json:"update_name"`

	// === 乐观锁 ===
	Version int64 `gorm:"default:0" json:"version"`

	// === 时间戳 ===
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AuditFields 获取审计字段的 GORM 字段定义
// 用于 AutoMigrate 自动迁移
//
// 返回值：
//   - []interface{}: GORM 字段定义切片
func (m *BaseModel) AuditFields() []interface{} {
	return []interface{}{
		&struct {
			CreateUser uint   `gorm:"index"`
			CreateName string `gorm:"size:64"`
			UpdateUser uint   `gorm:"index"`
			UpdateName string `gorm:"size:64"`
			DeleteUser uint   `gorm:"index"`
			DeleteName string `gorm:"size:64"`
			Version    int64  `gorm:"default:0"`
		}{},
	}
}

// HasVersion 获取当前版本号
func (m *BaseModel) GetVersion() int64 {
	return m.Version
}

// SetVersion 设置版本号
func (m *BaseModel) SetVersion(v int64) {
	m.Version = v
}

// IncrementVersion 版本号 +1
func (m *BaseModel) IncrementVersion() {
	m.Version++
}

// IsDeleted 检查是否已软删除
func (m *BaseModel) IsDeleted() bool {
	return m.DeletedAt.Valid && !m.DeletedAt.Time.IsZero()
}

// GetCreateUser 获取创建人ID
func (m *BaseModel) GetCreateUser() uint {
	return m.CreateUser
}

// GetUpdateUser 获取更新人ID
func (m *BaseModel) GetUpdateUser() uint {
	return m.UpdateUser
}

// GetDeleteUser 获取删除人ID
func (m *BaseModel) GetDeleteUser() uint {
	return m.DeleteUser
}
