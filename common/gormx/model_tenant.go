package gormx

import (
	"time"

	"gorm.io/gorm"
)

// DefaultTenantID 默认租户ID（用于系统表）
const DefaultTenantID = "000000"

// TenantModel 多租户模型基类
//
// 包含租户隔离字段和所有审计字段，适合多租户 SaaS 应用。
// GORM Callbacks 会自动填充审计字段和租户ID。
// TenantScope 可自动实现租户数据隔离。
//
// 使用示例：
//
//	type UserConfig struct {
//	    gormx.TenantModel
//	    ConfigKey   string
//	    ConfigValue string
//	}
type TenantModel struct {
	// === 主键 ===
	ID uint `gorm:"primarykey" json:"id"`

	// === 租户隔离（核心字段）===
	TenantID string `gorm:"size:12;index;not null;default:'000000'" json:"tenant_id"`

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

// TenantIntIDModel 多租户 + int 主键模型
//
// 使用场景：需要与其他系统对接时使用 int 类型 ID
type TenantIntIDModel struct {
	ID uint `gorm:"primarykey" json:"id"`

	// === 租户隔离 ===
	TenantID string `gorm:"size:12;index;not null;default:'000000'" json:"tenant_id"`

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

// TenantStringIDModel 多租户 + string 主键模型
//
// 使用场景：需要字符串主键（如 UUID）且支持多租户
type TenantStringIDModel struct {
	ID string `gorm:"primarykey;size:36" json:"id"`

	// === 租户隔离 ===
	TenantID string `gorm:"size:12;index;not null;default:'000000'" json:"tenant_id"`

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

// TenantTimeModel 多租户 + 无软删除模型
//
// 使用场景：不需要软删除的多租户表
type TenantTimeModel struct {
	ID uint `gorm:"primarykey" json:"id"`

	// === 租户隔离 ===
	TenantID string `gorm:"size:12;index;not null;default:'000000'" json:"tenant_id"`

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

// TenantIntTimeModel 多租户 + int 主键 + 无软删除模型
type TenantIntTimeModel struct {
	ID uint `gorm:"primarykey" json:"id"`

	// === 租户隔离 ===
	TenantID string `gorm:"size:12;index;not null;default:'000000'" json:"tenant_id"`

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

// TenantStringTimeModel 多租户 + string 主键 + 无软删除模型
type TenantStringTimeModel struct {
	ID string `gorm:"primarykey;size:36" json:"id"`

	// === 租户隔离 ===
	TenantID string `gorm:"size:12;index;not null;default:'000000'" json:"tenant_id"`

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

// TenantTenantOnlyModel 仅租户模型（无审计字段）
//
// 使用场景：轻量级多租户表，不需要审计信息
type TenantOnlyModel struct {
	ID       uint   `gorm:"primarykey" json:"id"`
	TenantID string `gorm:"size:12;index;not null;default:'000000'" json:"tenant_id"`
}

// GetTenantID 获取租户ID
func (m *TenantModel) GetTenantID() string {
	return m.TenantID
}

// SetTenantID 设置租户ID
func (m *TenantModel) SetTenantID(tenantID string) {
	m.TenantID = tenantID
}

// IsDefaultTenant 检查是否为默认租户
func (m *TenantModel) IsDefaultTenant() bool {
	return m.TenantID == "" || m.TenantID == DefaultTenantID
}

// GetVersion 获取当前版本号
func (m *TenantModel) GetVersion() int64 {
	return m.Version
}

// IncrementVersion 版本号 +1
func (m *TenantModel) IncrementVersion() {
	m.Version++
}

// IsDeleted 检查是否已软删除
func (m *TenantModel) IsDeleted() bool {
	return m.DeletedAt.Valid && !m.DeletedAt.Time.IsZero()
}

// TenantFields 获取租户相关字段定义
// 用于 AutoMigrate 自动迁移
func (m *TenantModel) TenantFields() []interface{} {
	return []interface{}{
		&struct {
			TenantID string `gorm:"size:12;index;not null;default:'000000'"`
		}{},
	}
}

// AuditFields 获取审计字段定义
func (m *TenantModel) AuditFields() []interface{} {
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
