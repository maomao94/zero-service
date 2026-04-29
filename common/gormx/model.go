package gormx

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

const (
	LegacyDelStateActive  soft_delete.DeletedAt = 0
	LegacyDelStateDeleted soft_delete.DeletedAt = 1
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

// LegacyIDMixin 提供旧表使用的 int64 id 字段。
type LegacyIDMixin struct {
	Id int64 `gorm:"column:id;primaryKey" json:"id"`
}

// LegacyStringIDMixin 提供旧表使用的 string id 字段。
type LegacyStringIDMixin struct {
	Id string `gorm:"column:id;primaryKey;size:36" json:"id"`
}

// LegacyTimeMixin 提供旧表 create_time 和 update_time 字段。
type LegacyTimeMixin struct {
	CreateTime time.Time `gorm:"column:create_time;type:timestamp(6);autoCreateTime:milli" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time;type:timestamp(6);autoUpdateTime:milli" json:"update_time"`
}

// LegacySoftDeleteMixin 提供旧表 delete_time 和 del_state 软删除字段。
type LegacySoftDeleteMixin struct {
	DeleteTime sql.NullTime          `gorm:"column:delete_time;index" json:"-"`
	DelState   soft_delete.DeletedAt `gorm:"column:del_state;softDelete:flag,DeletedAtField:DeleteTime;default:0;index" json:"del_state"`
}

// IsDeleted 判断旧表软删除字段是否表示已删除。
func (m *LegacySoftDeleteMixin) IsDeleted() bool {
	return m.DelState == LegacyDelStateDeleted || m.DeleteTime.Valid
}

// LegacyBaseModel 组合旧表 int64 主键、旧时间字段、旧软删除字段和版本字段。
type LegacyBaseModel struct {
	LegacyIDMixin
	LegacyTimeMixin
	LegacySoftDeleteMixin
	VersionMixin
}

// LegacyStringBaseModel 组合旧表 string 主键、旧时间字段、旧软删除字段和版本字段。
type LegacyStringBaseModel struct {
	LegacyStringIDMixin
	LegacyTimeMixin
	LegacySoftDeleteMixin
	VersionMixin
}
