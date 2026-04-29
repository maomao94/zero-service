package gormx

// TenantModel 是带租户字段的新表默认 uint 主键模型。
type TenantModel struct {
	IDModel
	TenantMixin
	AuditMixin
	VersionMixin
	SoftDeleteMixin
	TimeMixin
}

// TenantStringIDModel 是带租户字段的新表 string 主键模型。
type TenantStringIDModel struct {
	StringIDModel
	TenantMixin
	AuditMixin
	VersionMixin
	SoftDeleteMixin
	TimeMixin
}

// TenantTimeModel 是带租户字段、string 主键、时间、版本和无删除审计字段的模型。
type TenantTimeModel struct {
	StringIDModel
	TenantMixin
	AuditWithoutDeleteMixin
	VersionMixin
	TimeMixin
}

// TenantOnlyModel 是只需要主键和租户字段的轻量模型。
type TenantOnlyModel struct {
	IDModel
	TenantMixin
}
