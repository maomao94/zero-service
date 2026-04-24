package hooks

import (
	"context"
	"encoding/json"

	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
)

const onlineValue = "1"

// OnStatus 设备上下线状态钩子。
// Topic: sys/product/{gateway_sn}/status
// Direction: up（设备→云平台）
// 设备上线/下线/拓扑变更时触发，更新缓存中的在线状态。
func OnStatus(onlineCache *collection.Cache) func(ctx context.Context, gatewaySn string, data *djisdk.StatusMessage) {
	return func(ctx context.Context, gatewaySn string, data *djisdk.StatusMessage) {
		logx.WithContext(ctx).Infof("[dji-gateway] status: sn=%s method=%s", gatewaySn, data.Method)

		raw, err := json.Marshal(data.Data)
		if err != nil {
			logx.WithContext(ctx).Errorf("[dji-gateway] status marshal data failed: %v", err)
			return
		}
		var topo djisdk.TopoUpdateData
		if err := json.Unmarshal(raw, &topo); err != nil {
			logx.WithContext(ctx).Errorf("[dji-gateway] status unmarshal topo failed: %v", err)
			return
		}

		if len(topo.SubDevices) > 0 {
			onlineCache.Set(gatewaySn, onlineValue)
			logx.WithContext(ctx).Infof("[dji-gateway] dock online: sn=%s sub_devices=%d", gatewaySn, len(topo.SubDevices))
		} else {
			onlineCache.Del(gatewaySn)
			logx.WithContext(ctx).Infof("[dji-gateway] dock offline: sn=%s", gatewaySn)
		}
	}
}

// IsOnline 检查机巢是否在线。
func IsOnline(onlineCache *collection.Cache, gatewaySn string) bool {
	_, ok := onlineCache.Get(gatewaySn)
	return ok
}
