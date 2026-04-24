package hooks

import (
	"context"

	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
)

// OnOsd 设备 OSD 遥测数据上报钩子。
// Topic: thing/product/{device_sn}/osd
// Direction: up（设备→云平台）
// 设备定期推送实时遥测数据（飞行姿态、GPS 坐标、电池电量等）。
// 同时刷新机巢在线缓存 TTL，作为辅助心跳检测。
func OnOsd(onlineCache *collection.Cache) func(ctx context.Context, deviceSn string, data *djisdk.OsdMessage) {
	return func(ctx context.Context, deviceSn string, data *djisdk.OsdMessage) {
		logx.WithContext(ctx).Debugf("[dji-gateway] osd: sn=%s tid=%s ts=%d", deviceSn, data.Tid, data.Timestamp)

		sn := data.Gateway
		if sn == "" {
			sn = deviceSn
		}
		onlineCache.Set(sn, onlineValue)
	}
}
