package dispatch

import (
	"context"
	"maps"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/gis/gis"
	"zero-service/app/podengine/podengine"
	"zero-service/common/errx"
	"zero-service/third_party/extproto"

	"google.golang.org/protobuf/proto"
)

// Handler 处理单个 method：data 为该 method 的 protojson 文本。
type Handler func(ctx context.Context, data string) (proto.Message, error)

// Registry 是 method 到下游调用的显式路由表。
type Registry struct {
	routes map[string]Handler
}

// NewRegistry 构建真实下游路由表。
func NewRegistry(podClient podengine.PodEngineClient, modbusClient bridgemodbus.BridgeModbusClient, gisClient gis.GisClient) *Registry {
	routes := make(map[string]Handler)
	if podClient != nil {
		maps.Copy(routes, podmanRoutes(podClient))
	}
	if modbusClient != nil {
		maps.Copy(routes, modbusRoutes(modbusClient))
	}
	if gisClient != nil {
		maps.Copy(routes, gisRoutes(gisClient))
	}
	return &Registry{routes: routes}
}

// Invoke 按 method 分发；未注册的 method 返回 130004。
func (r *Registry) Invoke(ctx context.Context, method string, data string) (proto.Message, error) {
	if r == nil {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_INTERNAL, "路由表未初始化")
	}
	handler, ok := r.routes[method]
	if !ok {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_30_METHOD_NOT_OPEN, "method 未开放: %s", method)
	}
	return handler(ctx, data)
}
