package dispatch

import (
	"context"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/podengine/podengine"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"google.golang.org/protobuf/proto"
)

// MethodPodList 等是本服务对外开放的 method 名称。
const (
	MethodPodList   = "pod.list"
	MethodPodGet    = "pod.get"
	MethodPodStats  = "pod.stats"
	MethodImageList = "image.list"
)

// Handler 处理单个 method：data 为该 method 的 protojson。
type Handler func(ctx context.Context, data []byte) (proto.Message, error)

// Registry 是 method 到下游调用的显式路由表。
type Registry struct {
	routes map[string]Handler
}

// NewRegistry 构建真实下游路由表。
func NewRegistry(client podengine.PodEngineClient, modbusClient bridgemodbus.BridgeModbusClient) *Registry {
	routes := map[string]Handler{
		MethodPodList:   relayListPods(client),
		MethodPodGet:    relayGetPod(client),
		MethodPodStats:  relayGetPodStats(client),
		MethodImageList: relayListImages(client),
	}
	for method, handler := range newModbusRoutes(modbusClient) {
		routes[method] = handler
	}
	return &Registry{routes: routes}
}

// Invoke 按 method 分发；未注册的 method 返回 130103。
func (r *Registry) Invoke(ctx context.Context, method string, data []byte) (proto.Message, error) {
	if r == nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_00_INTERNAL, "路由表未初始化")
	}
	handler, ok := r.routes[method]
	if !ok {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_30_METHOD_NOT_OPEN, "method 未开放: %s", method)
	}
	return handler(ctx, data)
}

func relayListPods(client podengine.PodEngineClient) Handler {
	return func(ctx context.Context, data []byte) (proto.Message, error) {
		var in podengine.ListPodsReq
		if err := decode(data, &in); err != nil {
			return nil, err
		}
		return client.ListPods(ctx, &in)
	}
}

func relayGetPod(client podengine.PodEngineClient) Handler {
	return func(ctx context.Context, data []byte) (proto.Message, error) {
		var in podengine.GetPodReq
		if err := decode(data, &in); err != nil {
			return nil, err
		}
		return client.GetPod(ctx, &in)
	}
}

func relayGetPodStats(client podengine.PodEngineClient) Handler {
	return func(ctx context.Context, data []byte) (proto.Message, error) {
		var in podengine.GetPodStatsReq
		if err := decode(data, &in); err != nil {
			return nil, err
		}
		return client.GetPodStats(ctx, &in)
	}
}

func relayListImages(client podengine.PodEngineClient) Handler {
	return func(ctx context.Context, data []byte) (proto.Message, error) {
		var in podengine.ListImagesReq
		if err := decode(data, &in); err != nil {
			return nil, err
		}
		return client.ListImages(ctx, &in)
	}
}
