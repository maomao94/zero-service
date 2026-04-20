package hooks

import (
	"context"

	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

// OnState 设备状态上报钩子。
// Topic: thing/product/{device_sn}/state
// Direction: up（设备→云平台）
// 设备上报自身状态信息（固件版本、在线状态、设备能力集等），
// 与 OSD 不同，state 侧重于设备元信息而非实时飞行数据。
// 业务端可在此钩子中实现设备状态同步、拓扑更新等逻辑。
func OnState(ctx context.Context, deviceSn string, data *djisdk.OsdMessage) {
	logx.WithContext(ctx).Infof("[dji-gateway] state: sn=%s tid=%s ts=%d", deviceSn, data.Tid, data.Timestamp)
}
