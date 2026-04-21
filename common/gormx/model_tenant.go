package gormx

import (
	"time"
)

type TenantModel struct {
	ID uint `gorm:"primarykey" json:"id"`
	TenantMixin
	AuditMixin
	VersionMixin
	SoftDeleteMixin
	CreatedAt time.Time `gorm:"type:timestamp(6)" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp(6)" json:"updated_at"`
}

type TenantIntIDModel = TenantModel

type TenantStringIDModel struct {
	ID string `gorm:"primarykey;size:36" json:"id"`
	TenantMixin
	AuditMixin
	VersionMixin
	SoftDeleteMixin
	CreatedAt time.Time `gorm:"type:timestamp(6)" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp(6)" json:"updated_at"`
}

type TenantTimeModel struct {
	ID string `gorm:"primarykey;size:36" json:"id"`
	TenantMixin
	AuditWithoutDeleteMixin
	VersionMixin
	CreatedAt time.Time `gorm:"type:timestamp(6)" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp(6)" json:"updated_at"`
}

type TenantIntTimeModel struct {
	ID uint `gorm:"primarykey" json:"id"`
	TenantMixin
	AuditWithoutDeleteMixin
	VersionMixin
	CreatedAt time.Time `gorm:"type:timestamp(6)" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp(6)" json:"updated_at"`
}

type TenantStringTimeModel struct {
	ID string `gorm:"primarykey;size:36" json:"id"`
	TenantMixin
	AuditWithoutDeleteMixin
	VersionMixin
	CreatedAt time.Time `gorm:"type:timestamp(6)" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp(6)" json:"updated_at"`
}

type TenantOnlyModel struct {
	ID uint `gorm:"primarykey" json:"id"`
	TenantMixin
}
