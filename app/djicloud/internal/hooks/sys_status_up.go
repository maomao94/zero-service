package hooks

import (
	"context"
	"encoding/json"

	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
)

// NewStatusHandler 构造 sys/product/{gateway_sn}/status 上行处理器。
func NewStatusHandler(onlineCache *collection.Cache) djisdk.StatusHandler {
	return func(ctx context.Context, gatewaySn string, data *djisdk.StatusMessage) int {
		logx.WithContext(ctx).Infof("[dji-cloud] status: sn=%s method=%s", gatewaySn, data.Method)

		onlineCache.Set(gatewaySn, OnlineValue)

		if data.Method != djisdk.MethodUpdateTopo {
			return djisdk.PlatformResultOK
		}

		raw, err := json.Marshal(data.Data)
		if err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] status marshal data failed: %v", err)
			return djisdk.PlatformResultHandlerError
		}
		var topo djisdk.TopoUpdateData
		if err := json.Unmarshal(raw, &topo); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] status unmarshal topo failed: %v", err)
			return djisdk.PlatformResultHandlerError
		}
		logx.WithContext(ctx).Infof("[dji-cloud] topo update: sn=%s sub_devices=%d", gatewaySn, len(topo.SubDevices))
		return djisdk.PlatformResultOK
	}
}
