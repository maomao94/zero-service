package hooks

import (
	"context"

	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

// OnHmsEventNotify HMS 健康告警上报钩子。
// Topic: thing/product/{gateway_sn}/events
// Direction: up（设备→云平台）
// Method: hms
// 设备上报健康管理系统的告警和状态事件，包含告警级别、模块、告警码等信息。
// 业务端可在此钩子中实现告警持久化、WebSocket 推送、告警通知等逻辑。
func OnHmsEventNotify(ctx context.Context, gatewaySn string, data *djisdk.HmsEventData) {
	logx.WithContext(ctx).Infof("[dji-gateway] hms event: sn=%s count=%d", gatewaySn, len(data.List))
	for _, item := range data.List {
		logx.WithContext(ctx).Infof("[dji-gateway] hms item: hms_id=%s level=%d module=%d in_the_sky=%d code=%s imminent=%v key=%s",
			item.HmsID, item.Level, item.Module, item.InTheSky, item.Code, item.Imminent, item.Key)
	}
}
