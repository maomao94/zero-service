package gormx

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

const DefaultTenantID = "000000"

type AuditMixin struct {
	CreateUser uint   `gorm:"index" json:"create_user"`
	CreateName string `gorm:"size:64" json:"create_name"`
	UpdateUser uint   `gorm:"index" json:"update_user"`
	UpdateName string `gorm:"size:64" json:"update_name"`
	DeleteUser uint   `gorm:"index" json:"delete_user"`
	DeleteName string `gorm:"size:64" json:"delete_name"`
}

type AuditWithoutDeleteMixin struct {
	CreateUser uint   `gorm:"index" json:"create_user"`
	CreateName string `gorm:"size:64" json:"create_name"`
	UpdateUser uint   `gorm:"index" json:"update_user"`
	UpdateName string `gorm:"size:64" json:"update_name"`
}

type VersionMixin struct {
	Version int64 `gorm:"default:0" json:"version"`
}

func (m *VersionMixin) GetVersion() int64 {
	return m.Version
}

func (m *VersionMixin) SetVersion(v int64) {
	m.Version = v
}

func (m *VersionMixin) IncrementVersion() {
	m.Version++
}

type SoftDeleteMixin struct {
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (m *SoftDeleteMixin) IsDeleted() bool {
	return m.DeletedAt.Valid && !m.DeletedAt.Time.IsZero()
}

type TenantMixin struct {
	TenantID string `gorm:"size:12;index;not null;default:'000000'" json:"tenant_id"`
}

func (m *TenantMixin) GetTenantID() string {
	return m.TenantID
}

func (m *TenantMixin) SetTenantID(tenantID string) {
	m.TenantID = tenantID
}

func (m *TenantMixin) IsDefaultTenant() bool {
	return m.TenantID == "" || m.TenantID == DefaultTenantID
}

type LegacyBaseModel struct {
	Id         int64        `gorm:"column:id;primaryKey" json:"id"`
	CreateTime time.Time    `gorm:"column:create_time;type:timestamp(6);autoCreateTime:milli"`
	UpdateTime time.Time    `gorm:"column:update_time;type:timestamp(6);autoUpdateTime:milli"`
	DeleteTime sql.NullTime `gorm:"column:delete_time;type:timestamp(6);index" json:"-"`
	DelState   int64        `gorm:"column:del_state;default:0"`
	Version    int64        `gorm:"column:version;default:0"`
}

func (m *LegacyBaseModel) GetVersion() int64 {
	return m.Version
}

func (m *LegacyBaseModel) SetVersion(v int64) {
	m.Version = v
}

func (m *LegacyBaseModel) IsDeleted() bool {
	return m.DeleteTime.Valid && !m.DeleteTime.Time.IsZero()
}

type LegacyStringBaseModel struct {
	Id         string       `gorm:"column:id;primaryKey;size:36" json:"id"`
	CreateTime time.Time    `gorm:"column:create_time;type:timestamp(6);autoCreateTime:milli"`
	UpdateTime time.Time    `gorm:"column:update_time;type:timestamp(6);autoUpdateTime:milli"`
	DeleteTime sql.NullTime `gorm:"column:delete_time;type:timestamp(6);index" json:"-"`
	DelState   int64        `gorm:"column:del_state;default:0"`
	Version    int64        `gorm:"column:version;default:0"`
}

func (m *LegacyStringBaseModel) GetVersion() int64 {
	return m.Version
}

func (m *LegacyStringBaseModel) SetVersion(v int64) {
	m.Version = v
}

func (m *LegacyStringBaseModel) IsDeleted() bool {
	return m.DeleteTime.Valid && !m.DeleteTime.Time.IsZero()
}
