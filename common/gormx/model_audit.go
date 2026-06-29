package gormx

// AuditMixin 提供 uint 类型的创建、更新和删除审计用户字段。
type AuditMixin struct {
	CreateUser uint   `gorm:"column:create_user" json:"create_user"`
	CreateName string `gorm:"column:create_name;size:64" json:"create_name"`
	UpdateUser uint   `gorm:"column:update_user" json:"update_user"`
	UpdateName string `gorm:"column:update_name;size:64" json:"update_name"`
	DeleteUser uint   `gorm:"column:delete_user" json:"delete_user"`
	DeleteName string `gorm:"column:delete_name;size:64" json:"delete_name"`
}

// StringAuditMixin 提供 string 类型的创建、更新和删除审计用户字段。
type StringAuditMixin struct {
	CreateUser string `gorm:"column:create_user;size:64" json:"create_user"`
	CreateName string `gorm:"column:create_name;size:64" json:"create_name"`
	UpdateUser string `gorm:"column:update_user;size:64" json:"update_user"`
	UpdateName string `gorm:"column:update_name;size:64" json:"update_name"`
	DeleteUser string `gorm:"column:delete_user;size:64" json:"delete_user"`
	DeleteName string `gorm:"column:delete_name;size:64" json:"delete_name"`
}

// AuditWithoutDeleteMixin 提供 uint 类型的创建和更新审计用户字段，不包含删除审计字段。
type AuditWithoutDeleteMixin struct {
	CreateUser uint   `gorm:"column:create_user" json:"create_user"`
	CreateName string `gorm:"column:create_name;size:64" json:"create_name"`
	UpdateUser uint   `gorm:"column:update_user" json:"update_user"`
	UpdateName string `gorm:"column:update_name;size:64" json:"update_name"`
}

// StringAuditWithoutDeleteMixin 提供 string 类型的创建和更新审计用户字段，不包含删除审计字段。
type StringAuditWithoutDeleteMixin struct {
	CreateUser string `gorm:"column:create_user;size:64" json:"create_user"`
	CreateName string `gorm:"column:create_name;size:64" json:"create_name"`
	UpdateUser string `gorm:"column:update_user;size:64" json:"update_user"`
	UpdateName string `gorm:"column:update_name;size:64" json:"update_name"`
}
