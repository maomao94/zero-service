package hooks

import (
	"context"

	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

// NewDeviceRequestHandler 构造 thing/product/{gateway_sn}/requests 上行处理器。
//
// requests 是设备向云端拉取平台侧配置或状态的通道，必须按 method 返回匹配的 output：
// airport_organization_get 返回组织信息占位；airport_bind_status 返回绑定状态；flight_areas_get 返回自定义飞行区列表。
// 当前未接入外部组织/飞行区配置源时返回安全空值，避免已知 method 被错误地回复为空 data。
func NewDeviceRequestHandler() djisdk.RequestHandler {
	return func(ctx context.Context, gatewaySn string, req *djisdk.RequestMessage) (int, any, error) {
		if req == nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] request: nil request payload, sn=%s", gatewaySn)
			return djisdk.PlatformResultHandlerError, nil, nil
		}
		logx.WithContext(ctx).Infof("[dji-cloud] request: method=%s sn=%s tid=%s", req.Method, gatewaySn, req.Tid)
		switch req.Method {
		case djisdk.MethodAirportOrganizationGet:
			return djisdk.PlatformResultOK, map[string]any{
				"organization_id":   "",
				"organization_name": "",
			}, nil
		case djisdk.MethodAirportBindStatus:
			return djisdk.PlatformResultOK, map[string]any{
				"status": 0,
			}, nil
		case djisdk.MethodFlightAreasGet:
			return djisdk.PlatformResultOK, map[string]any{
				"flight_areas": []any{},
			}, nil
		default:
			return djisdk.PlatformResultOK, nil, nil
		}
	}
}
