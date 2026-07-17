package gormx

import (
	"database/sql"
	"time"

	"zero-service/common/tool"

	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

const (
	LegacyIsDeletedActive  soft_delete.DeletedAt = 0
	LegacyIsDeletedDeleted soft_delete.DeletedAt = 1
)

// LegacyIDMixin 提供旧表使用的 int64 id 字段。
type LegacyIDMixin struct {
	Id int64 `gorm:"column:id;primaryKey" json:"id"`
}

// LegacyStringIDMixin 提供旧表使用的 string id 字段。
type LegacyStringIDMixin struct {
	Id string `gorm:"column:id;primaryKey;size:64" json:"id"`
}

func (m *LegacyStringIDMixin) BeforeCreateID() error {
	if m.Id == "" {
		var err error
		m.Id, err = tool.SimpleUUID()
		return err
	}
	return nil
}

// LegacyTimeMixin 提供旧表 create_time 和 update_time 字段。
type LegacyTimeMixin struct {
	CreateTime time.Time `gorm:"column:create_time;type:timestamp;autoCreateTime:milli" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time;type:timestamp;autoUpdateTime:milli" json:"update_time"`
}

// LegacySoftDeleteMixin 提供旧系统兼容的 delete_time 和 is_deleted 软删除字段。
type LegacySoftDeleteMixin struct {
	DeleteTime sql.NullTime          `gorm:"column:delete_time;type:timestamp;index" json:"-"`
	IsDeleted  soft_delete.DeletedAt `gorm:"column:is_deleted;softDelete:flag,DeletedAtField:DeleteTime;default:0;index" json:"is_deleted"`
}

// Deleted 判断旧系统 is_deleted 字段是否表示已删除。
func (m *LegacySoftDeleteMixin) Deleted() bool {
	return m.IsDeleted == LegacyIsDeletedDeleted
}

// LegacyBaseModel 组合旧表 int64 主键、旧系统时间和软删除字段。
type LegacyBaseModel struct {
	LegacyIDMixin
	LegacyTimeMixin
	LegacySoftDeleteMixin
}

func (m *LegacyBaseModel) BeforeCreate(tx *gorm.DB) error {
	legacyBeforeCreate(tx)
	return nil
}

func (m *LegacyBaseModel) BeforeUpdate(tx *gorm.DB) error {
	legacyBeforeUpdate(tx)
	return nil
}

func (m *LegacyBaseModel) BeforeDelete(tx *gorm.DB) error {
	legacyBeforeDelete(tx)
	return nil
}

// LegacyStringBaseModel 组合旧表 string 主键、旧系统时间和软删除字段。
type LegacyStringBaseModel struct {
	LegacyStringIDMixin
	LegacyTimeMixin
	LegacySoftDeleteMixin
}

func (m *LegacyStringBaseModel) BeforeCreate(tx *gorm.DB) error {
	if err := m.BeforeCreateID(); err != nil {
		return err
	}
	legacyBeforeCreate(tx)
	return nil
}

func (m *LegacyStringBaseModel) BeforeUpdate(tx *gorm.DB) error {
	legacyBeforeUpdate(tx)
	return nil
}

func (m *LegacyStringBaseModel) BeforeDelete(tx *gorm.DB) error {
	legacyBeforeDelete(tx)
	return nil
}

func legacyBeforeCreate(tx *gorm.DB) {
	userCtx := GetUserContext(tx.Statement.Context)
	if userCtx == nil {
		return
	}
	if userID := userCtx.AuditUserValue(); userID != nil {
		setSchemaColumn(tx, "create_user", userID)
		setSchemaColumn(tx, "create_name", userCtx.UserName)
		setSchemaColumn(tx, "update_user", userID)
		setSchemaColumn(tx, "update_name", userCtx.UserName)
	}
	if userCtx.TenantID != "" {
		setSchemaColumn(tx, "tenant_id", userCtx.TenantID)
	}
}

func legacyBeforeUpdate(tx *gorm.DB) {
	userCtx := GetUserContext(tx.Statement.Context)
	if userCtx == nil {
		return
	}
	if userID := userCtx.AuditUserValue(); userID != nil {
		setSchemaColumn(tx, "update_user", userID)
		setSchemaColumn(tx, "update_name", userCtx.UserName)
	}
}

func legacyBeforeDelete(tx *gorm.DB) {
	// Delete audit is intentionally not written here. Calling SetColumn here
	// corrupts soft_delete plugin variables in the generated UPDATE.
}
