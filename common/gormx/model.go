package gormx

import (
	"time"

	"gorm.io/gorm"
)

// IDModel 提供新表默认使用的 uint 主键。
type IDModel struct {
	ID uint `gorm:"primarykey" json:"id"`
}

// StringIDModel 提供适合 UUID 或外部 ID 的 string 主键。
type StringIDModel struct {
	ID string `gorm:"primarykey;size:36" json:"id"`
}

// TimeMixin 提供 created_at 和 updated_at 字段，并保留 MySQL timestamp(6) 精度。
type TimeMixin struct {
	CreatedAt time.Time `gorm:"type:timestamp(6)" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp(6)" json:"updated_at"`
}

// SoftDeleteMixin 启用 GORM 标准 deleted_at 软删除。
type SoftDeleteMixin struct {
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// VersionMixin 提供 version 字段，用于业务侧乐观锁或版本记录。
type VersionMixin struct {
	Version int64 `gorm:"column:version;default:0" json:"version"`
}

// TenantMixin 提供 tenant_id 字段，用于租户隔离模型。
type TenantMixin struct {
	TenantID string `gorm:"column:tenant_id;size:64;index" json:"tenant_id"`
}
