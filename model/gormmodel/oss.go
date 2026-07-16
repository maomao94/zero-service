package gormmodel

import "zero-service/common/gormx"

type Oss struct {
	gormx.LegacyStringBaseModel
	gormx.VersionMixin
	TenantId   string `gorm:"column:tenant_id;type:varchar(12);default:000000;uniqueIndex:idx_tid_code;comment:租户ID"`
	Category   int64  `gorm:"column:category;default:0;comment:分类 1-minio 2-qiniu 3-ali 4-tecent"`
	OssCode    string `gorm:"column:oss_code;type:varchar(32);default:'';uniqueIndex:idx_tid_code;comment:资源编号"`
	Endpoint   string `gorm:"column:endpoint;type:varchar(255);default:'';comment:资源地址"`
	AccessKey  string `gorm:"column:access_key;type:varchar(255);default:'';comment:accessKey"`
	SecretKey  string `gorm:"column:secret_key;type:varchar(255);default:'';comment:secretKey"`
	BucketName string `gorm:"column:bucket_name;type:varchar(255);default:'';comment:空间名"`
	AppId      string `gorm:"column:app_id;type:varchar(255);default:'';comment:应用ID"`
	Region     string `gorm:"column:region;type:varchar(255);default:'';comment:地域简称"`
	Remark     string `gorm:"column:remark;type:varchar(255);default:'';comment:备注"`
	Status     int64  `gorm:"column:status;default:0;comment:状态 1-开启 2-关闭"`
}

func (Oss) TableName() string {
	return "oss"
}
