package hooks

import (
	"context"

	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

// OnOsd 设备 OSD 遥测数据上报钩子。
// Topic: thing/product/{device_sn}/osd
// Direction: up（设备→云平台）
// 设备定期推送实时遥测数据（飞行姿态、GPS 坐标、电池电量等）。
// 业务端可在此钩子中实现遥测数据持久化、WebSocket 推送、地图轨迹绘制等逻辑。
func OnOsd(ctx context.Context, deviceSn string, data *djisdk.OsdMessage) {
	logx.WithContext(ctx).Debugf("[dji-gateway] osd: device=%s tid=%s ts=%d", deviceSn, data.Tid, data.Timestamp)
}
