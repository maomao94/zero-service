package hooks

import (
	"context"

	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
)

// NewOsdHandler 构造 thing/product/{device_sn}/osd 上行处理器。
func NewOsdHandler(onlineCache *collection.Cache) func(ctx context.Context, deviceSn string, data *djisdk.OsdMessage) {
	return func(ctx context.Context, deviceSn string, data *djisdk.OsdMessage) {
		logx.WithContext(ctx).Debugf("[dji-cloud] osd: sn=%s tid=%s ts=%d", deviceSn, data.Tid, data.Timestamp)

		sn := data.Gateway
		if sn == "" {
			sn = deviceSn
		}
		onlineCache.Set(sn, OnlineValue)
	}
}

// HandleStateTelemetry 由 thing/product/{device_sn}/state 上行驱动（物模型类状态，非实时航迹）。
// 与 OSD 区分见 DJI 文档；无云侧 state_reply 于本通道的默认回调。
func HandleStateTelemetry(ctx context.Context, deviceSn string, data *djisdk.StateMessage) {
	logx.WithContext(ctx).Infof("[dji-cloud] state: sn=%s tid=%s ts=%d", deviceSn, data.Tid, data.Timestamp)
}
