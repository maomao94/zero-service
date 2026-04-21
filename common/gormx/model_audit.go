package gormx

import (
	"time"
)

type BaseModel struct {
	ID uint `gorm:"primarykey" json:"id"`
	AuditMixin
	VersionMixin
	SoftDeleteMixin
	CreatedAt time.Time `gorm:"type:timestamp(6)" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp(6)" json:"updated_at"`
}

type IntBaseModel = BaseModel

type StringBaseModel struct {
	ID string `gorm:"primarykey;size:36" json:"id"`
	AuditMixin
	VersionMixin
	SoftDeleteMixin
	CreatedAt time.Time `gorm:"type:timestamp(6)" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp(6)" json:"updated_at"`
}

type TimeBaseModel struct {
	ID uint `gorm:"primarykey" json:"id"`
	AuditWithoutDeleteMixin
	VersionMixin
	CreatedAt time.Time `gorm:"type:timestamp(6)" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp(6)" json:"updated_at"`
}
