package gormmodel

import (
	"database/sql"

	"zero-service/common/gormx"
)

// SipProviderStatus 供应商状态。
const (
	SipProviderStatusEnabled  = int32(1)
	SipProviderStatusDisabled = int32(2)
)

// LiveSipProvider SIP 供应商配置。
type LiveSipProvider struct {
	gormx.LegacyStringBaseModel

	// 供应商编码（唯一，如 "freeswitch" / "telnyx" / "aliyun"）
	Code string `gorm:"column:code;size:32;not null;uniqueIndex:uq_live_sip_providers_code"`
	// 供应商名称
	Name string `gorm:"column:name;size:128;not null;comment:供应商名称"`
	// SIP 服务器地址（如 "freeswitch:5080" / "sip.telnyx.com"）
	Address string `gorm:"column:address;size:256;not null;comment:SIP服务器地址"`
	// 主叫号码池 JSON 数组（如 ["+8613800138000", "1000"]）
	Numbers string `gorm:"column:numbers;type:text;not null;comment:主叫号码池JSON"`
	// SIP 认证用户名（可选）
	AuthUsername string `gorm:"column:auth_username;size:128;comment:认证用户名"`
	// SIP 认证密码（可选）
	AuthPassword string `gorm:"column:auth_password;size:256;comment:认证密码"`
	// 状态：1-启用 2-禁用
	Status int32 `gorm:"column:status;default:1;not null;comment:状态"`
	// 扩展配置 JSON
	Metadata string `gorm:"column:metadata;type:text;comment:扩展配置JSON"`

	CreateUser sql.NullString `gorm:"column:create_user;size:64;comment:创建人"`
	UpdateUser sql.NullString `gorm:"column:update_user;size:64;comment:更新人"`
	DeptCode   sql.NullString `gorm:"column:dept_code;size:64;comment:机构code"`
}

func (LiveSipProvider) TableName() string {
	return "live_sip_providers"
}
