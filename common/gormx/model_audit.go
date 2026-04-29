package gormx

// BaseModel 是新表默认模型：uint 主键、审计字段、版本字段、标准软删除和时间字段。
type BaseModel struct {
	IDModel
	AuditMixin
	VersionMixin
	SoftDeleteMixin
	TimeMixin
}

// StringBaseModel 是 string 主键的新表默认模型。
type StringBaseModel struct {
	StringIDModel
	AuditMixin
	VersionMixin
	SoftDeleteMixin
	TimeMixin
}

// StringAuditBaseModel 是 uint 主键且审计用户字段为 string 的模型。
type StringAuditBaseModel struct {
	IDModel
	StringAuditMixin
	VersionMixin
	SoftDeleteMixin
	TimeMixin
}

// StringIDStringAuditBaseModel 是 string 主键且审计用户字段为 string 的模型。
type StringIDStringAuditBaseModel struct {
	StringIDModel
	StringAuditMixin
	VersionMixin
	SoftDeleteMixin
	TimeMixin
}

// TimeBaseModel 是带 uint 主键、时间、版本和无删除审计字段的模型。
type TimeBaseModel struct {
	IDModel
	AuditWithoutDeleteMixin
	VersionMixin
	TimeMixin
}
