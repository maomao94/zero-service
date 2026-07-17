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

func (m *AuditMixin) BeforeCreateAudit(userCtx *UserContext) {
	if m == nil || userCtx == nil {
		return
	}
	userID := auditUserUint(userCtx)
	if userID == 0 {
		return
	}
	m.CreateUser = userID
	m.CreateName = userCtx.UserName
	m.UpdateUser = userID
	m.UpdateName = userCtx.UserName
}

func (m *AuditMixin) BeforeUpdateAudit(userCtx *UserContext) {
	if m == nil || userCtx == nil {
		return
	}
	userID := auditUserUint(userCtx)
	if userID == 0 {
		return
	}
	m.UpdateUser = userID
	m.UpdateName = userCtx.UserName
}

func (m *AuditMixin) BeforeDeleteAudit(userCtx *UserContext) {
	if m == nil || userCtx == nil {
		return
	}
	userID := auditUserUint(userCtx)
	if userID == 0 {
		return
	}
	m.DeleteUser = userID
	m.DeleteName = userCtx.UserName
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

func (m *StringAuditMixin) BeforeCreateAudit(userCtx *UserContext) {
	if m == nil || userCtx == nil {
		return
	}
	userID := auditUserString(userCtx)
	if userID == "" {
		return
	}
	m.CreateUser = userID
	m.CreateName = userCtx.UserName
	m.UpdateUser = userID
	m.UpdateName = userCtx.UserName
}

func (m *StringAuditMixin) BeforeUpdateAudit(userCtx *UserContext) {
	if m == nil || userCtx == nil {
		return
	}
	userID := auditUserString(userCtx)
	if userID == "" {
		return
	}
	m.UpdateUser = userID
	m.UpdateName = userCtx.UserName
}

func (m *StringAuditMixin) BeforeDeleteAudit(userCtx *UserContext) {
	if m == nil || userCtx == nil {
		return
	}
	userID := auditUserString(userCtx)
	if userID == "" {
		return
	}
	m.DeleteUser = userID
	m.DeleteName = userCtx.UserName
}

// AuditWithoutDeleteMixin 提供 uint 类型的创建和更新审计用户字段，不包含删除审计字段。
type AuditWithoutDeleteMixin struct {
	CreateUser uint   `gorm:"column:create_user" json:"create_user"`
	CreateName string `gorm:"column:create_name;size:64" json:"create_name"`
	UpdateUser uint   `gorm:"column:update_user" json:"update_user"`
	UpdateName string `gorm:"column:update_name;size:64" json:"update_name"`
}

func (m *AuditWithoutDeleteMixin) BeforeCreateAudit(userCtx *UserContext) {
	if m == nil || userCtx == nil {
		return
	}
	userID := auditUserUint(userCtx)
	if userID == 0 {
		return
	}
	m.CreateUser = userID
	m.CreateName = userCtx.UserName
	m.UpdateUser = userID
	m.UpdateName = userCtx.UserName
}

func (m *AuditWithoutDeleteMixin) BeforeUpdateAudit(userCtx *UserContext) {
	if m == nil || userCtx == nil {
		return
	}
	userID := auditUserUint(userCtx)
	if userID == 0 {
		return
	}
	m.UpdateUser = userID
	m.UpdateName = userCtx.UserName
}

// StringAuditWithoutDeleteMixin 提供 string 类型的创建和更新审计用户字段，不包含删除审计字段。
type StringAuditWithoutDeleteMixin struct {
	CreateUser string `gorm:"column:create_user;size:64" json:"create_user"`
	CreateName string `gorm:"column:create_name;size:64" json:"create_name"`
	UpdateUser string `gorm:"column:update_user;size:64" json:"update_user"`
	UpdateName string `gorm:"column:update_name;size:64" json:"update_name"`
}

func (m *StringAuditWithoutDeleteMixin) BeforeCreateAudit(userCtx *UserContext) {
	if m == nil || userCtx == nil {
		return
	}
	userID := auditUserString(userCtx)
	if userID == "" {
		return
	}
	m.CreateUser = userID
	m.CreateName = userCtx.UserName
	m.UpdateUser = userID
	m.UpdateName = userCtx.UserName
}

func (m *StringAuditWithoutDeleteMixin) BeforeUpdateAudit(userCtx *UserContext) {
	if m == nil || userCtx == nil {
		return
	}
	userID := auditUserString(userCtx)
	if userID == "" {
		return
	}
	m.UpdateUser = userID
	m.UpdateName = userCtx.UserName
}

func auditUserUint(userCtx *UserContext) uint {
	if userCtx == nil {
		return 0
	}
	switch v := userCtx.UserID.(type) {
	case uint:
		return v
	case uint64:
		return uint(v)
	case int64:
		if v < 0 {
			return 0
		}
		return uint(v)
	default:
		return 0
	}
}

func auditUserString(userCtx *UserContext) string {
	if userCtx == nil {
		return ""
	}
	if v, ok := userCtx.UserID.(string); ok {
		return v
	}
	return ""
}
