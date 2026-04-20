package gormx

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

// ============ GORM 标准命名模型 ============

// Model 通用模型基类（uint 主键 + GORM软删除）
//
// 使用示例：
//
//	type User struct {
//	    gormx.Model
//	    Name  string
//	}
type Model struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `gorm:"type:timestamp(6)"`
	UpdatedAt time.Time      `gorm:"type:timestamp(6)"`
	DeletedAt gorm.DeletedAt `gorm:"type:timestamp(6);index" json:"-"`
}

// IntIDModel int 类型主键模型
type IntIDModel struct {
	ID        int            `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `gorm:"type:timestamp(6)"`
	UpdatedAt time.Time      `gorm:"type:timestamp(6)"`
	DeletedAt gorm.DeletedAt `gorm:"type:timestamp(6);index" json:"-"`
}

// StringIDModel string 类型主键模型（UUID）
type StringIDModel struct {
	ID        string         `gorm:"primarykey;size:36" json:"id"`
	CreatedAt time.Time      `gorm:"type:timestamp(6)"`
	UpdatedAt time.Time      `gorm:"type:timestamp(6)"`
	DeletedAt gorm.DeletedAt `gorm:"type:timestamp(6);index" json:"-"`
}

// TimeModel 仅时间戳模型（无软删除）
type TimeModel struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at;type:timestamp(6)"`
	UpdatedAt time.Time `json:"updated_at;type:timestamp(6)"`
}

// ============ 老项目兼容模型（snake_case命名 + DelState） ============

// LegacyBaseModel 兼容老项目数据库结构的基础模型
//
// 字段：Id/CreateTime/UpdateTime/DeleteTime/DelState/Version
// DeleteTime使用gorm.DeletedAt实现GORM原生软删除，同时保留DelState字段
type LegacyBaseModel struct {
	Id         int64        `gorm:"column:id;primaryKey" json:"id"`
	CreateTime time.Time    `gorm:"column:create_time;type:timestamp(6);autoCreateTime:milli"`
	UpdateTime time.Time    `gorm:"column:update_time;type:timestamp(6);autoUpdateTime:milli"`
	DeleteTime sql.NullTime `gorm:"column:delete_time;type:timestamp(6);index" json:"-"`
	DelState   int64        `gorm:"column:del_state;default:0"`
	Version    int64        `gorm:"column:version;default:0"`
}

// LegacyStringBaseModel 老项目兼容模型（string主键）
type LegacyStringBaseModel struct {
	Id         string       `gorm:"column:id;primaryKey;size:36" json:"id"`
	CreateTime time.Time    `gorm:"column:create_time;type:timestamp(6);autoCreateTime:milli"`
	UpdateTime time.Time    `gorm:"column:update_time;type:timestamp(6);autoUpdateTime:milli"`
	DeleteTime sql.NullTime `gorm:"column:delete_time;type:timestamp(6);index" json:"-"`
	DelState   int64        `gorm:"column:del_state;default:0"`
	Version    int64        `gorm:"column:version;default:0"`
}

// GetVersion 获取版本号（乐观锁）
func (m *LegacyBaseModel) GetVersion() int64 {
	return m.Version
}

// SetVersion 设置版本号
func (m *LegacyBaseModel) SetVersion(v int64) {
	m.Version = v
}

// GetVersion 获取版本号（乐观锁）
func (m *LegacyStringBaseModel) GetVersion() int64 {
	return m.Version
}

// SetVersion 设置版本号
func (m *LegacyStringBaseModel) SetVersion(v int64) {
	m.Version = v
}

// ============ Model 辅助方法 ============

// IsDeleted 检查是否已软删除
func (m *Model) IsDeleted() bool {
	return m.DeletedAt.Valid && !m.DeletedAt.Time.IsZero()
}

// IsDeleted 检查是否已软删除
func (m *IntIDModel) IsDeleted() bool {
	return m.DeletedAt.Valid && !m.DeletedAt.Time.IsZero()
}

// IsDeleted 检查是否已软删除
func (m *StringIDModel) IsDeleted() bool {
	return m.DeletedAt.Valid && !m.DeletedAt.Time.IsZero()
}

// IsDeleted 检查是否已软删除
func (m *LegacyBaseModel) IsDeleted() bool {
	return m.DeleteTime.Valid && !m.DeleteTime.Time.IsZero()
}

// IsDeleted 检查是否已软删除
func (m *LegacyStringBaseModel) IsDeleted() bool {
	return m.DeleteTime.Valid && !m.DeleteTime.Time.IsZero()
}
