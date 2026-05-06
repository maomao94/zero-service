package logic

import (
	"time"

	"github.com/dromara/carbon/v2"
	"github.com/jinzhu/copier"
	"gorm.io/gorm/clause"

	"zero-service/app/file/file"
	"zero-service/model/gormmodel"
)

const (
	// OssStatusEnabled OSS 配置启用状态
	OssStatusEnabled = 2
)

var ossOrderColumns = map[string]clause.Column{
	"id":          {Name: "id"},
	"create_time": {Name: "create_time"},
	"update_time": {Name: "update_time"},
	"tenant_id":   {Name: "tenant_id"},
	"category":    {Name: "category"},
	"oss_code":    {Name: "oss_code"},
	"status":      {Name: "status"},
}

// ossOrderBy 以白名单机制构造安全的排序子句，防止 SQL 注入。
// 不在白名单中的列名默认按 id 降序排列。
func ossOrderBy(orderBy string) clause.OrderByColumn {
	column, ok := ossOrderColumns[orderBy]
	if !ok {
		column = clause.Column{Name: "id"}
	}
	return clause.OrderByColumn{Column: column, Desc: true}
}

// calcExpires 计算签名 URL 过期时间。若传入值 ≤0 或未指定，默认使用 60 分钟。
func calcExpires(expiresMinutes int32) time.Duration {
	if expiresMinutes > 0 {
		return time.Duration(expiresMinutes) * time.Minute
	}
	return 60 * time.Minute
}

// toPbOss 将 gormmodel.Oss 转换为 protobuf Oss 消息。
// copier 处理大部分字段映射后，手动将 time.Time 格式化为 DATETIME 字符串。
func toPbOss(oss *gormmodel.Oss) *file.Oss {
	var pb file.Oss
	copier.Copy(&pb, oss) // nolint:errcheck
	pb.CreateTime = carbon.CreateFromStdTime(oss.CreateTime).ToDateTimeString()
	pb.UpdateTime = carbon.CreateFromStdTime(oss.UpdateTime).ToDateTimeString()
	return &pb
}
