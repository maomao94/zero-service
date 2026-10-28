package dispatch

import (
	"context"

	"zero-service/app/podengine/podengine"

	"google.golang.org/protobuf/proto"
)

// NewMockRegistry 构建固定数据的 Mock 路由表，仅用于 dev/test 全链路验证。
// 仍会解析 data，保证非法 JSON/未知字段得到与真实模式一致的错误。
func NewMockRegistry() *Registry {
	routes := map[string]Handler{
		MethodPodList:   mockListPods,
		MethodPodGet:    mockGetPod,
		MethodPodStats:  mockGetPodStats,
		MethodImageList: mockListImages,
	}
	for method, handler := range newMockModbusRoutes() {
		routes[method] = handler
	}
	return &Registry{routes: routes}
}

func mockListPods(_ context.Context, data string) (proto.Message, error) {
	var in podengine.ListPodsReq
	if err := decode(data, &in); err != nil {
		return nil, err
	}
	return &podengine.ListPodsRes{
		Total: 2,
		Items: []*podengine.ListPodItemPb{
			{
				Id:          "mock-pod-1",
				Name:        "mock-nginx",
				Phase:       podengine.PodPhasePb_POD_PHASE_RUNNING,
				CreateTime:  "2026-10-22T10:00:00Z",
				Image:       "nginx:latest",
				Command:     "nginx -g 'daemon off;'",
				Ports:       []string{"8080:80"},
				State:       "running",
				Status:      "Up 2 hours",
				NetworkMode: "bridge",
			},
			{
				Id:          "mock-pod-2",
				Name:        "mock-redis",
				Phase:       podengine.PodPhasePb_POD_PHASE_STOPPED,
				CreateTime:  "2026-10-22T09:00:00Z",
				Image:       "redis:7-alpine",
				Command:     "redis-server",
				Ports:       []string{"6379:6379"},
				State:       "exited",
				Status:      "Exited (0) 30 minutes ago",
				NetworkMode: "bridge",
			},
		},
	}, nil
}

func mockGetPod(_ context.Context, data string) (proto.Message, error) {
	var in podengine.GetPodReq
	if err := decode(data, &in); err != nil {
		return nil, err
	}
	return &podengine.GetPodRes{Pod: &podengine.PodPb{
		Id:           "mock-pod-1",
		Name:         "mock-nginx",
		Phase:        podengine.PodPhasePb_POD_PHASE_RUNNING,
		NetworkMode:  "bridge",
		CreationTime: "2026-10-22T10:00:00Z",
		StartTime:    "2026-10-22T10:00:01Z",
		Conditions: []*podengine.PodConditionPb{
			{
				Type:               podengine.PodConditionTypePb_POD_CONDITION_TYPE_CONTAINERS_READY,
				Status:             true,
				Reason:             "ContainersReady",
				LastTransitionTime: "2026-10-22T10:00:05Z",
			},
		},
		Containers: []*podengine.ContainerPb{
			{
				Name:  "nginx",
				Image: "nginx:latest",
				State: &podengine.ContainerStatePb{
					Running:     true,
					StartedTime: "2026-10-22T10:00:01Z",
				},
				Ports: []string{"8080:80"},
			},
		},
	}}, nil
}

func mockGetPodStats(_ context.Context, data string) (proto.Message, error) {
	var in podengine.GetPodStatsReq
	if err := decode(data, &in); err != nil {
		return nil, err
	}
	return &podengine.GetPodStatsRes{Stats: []*podengine.ContainerStatsPb{
		{
			ContainerId:               "mock-container-1",
			ContainerName:             "mock-nginx",
			CpuUsagePercent:           1.25,
			CpuUsagePercentDisplay:    "1.25%",
			CpuUsageTotal:             1250000000,
			CpuUsageTotalDisplay:      "1.25s",
			MemoryUsage:               104857600,
			MemoryUsageDisplay:        "100 MiB",
			MemoryLimit:               536870912,
			MemoryLimitDisplay:        "512 MiB",
			MemoryUsagePercent:        19.53,
			MemoryUsagePercentDisplay: "19.53%",
			NetworkRxBytes:            2048,
			NetworkRxBytesDisplay:     "2 KiB",
			NetworkTxBytes:            1024,
			NetworkTxBytesDisplay:     "1 KiB",
			StorageReadBytes:          4096,
			StorageReadBytesDisplay:   "4 KiB",
			StorageWriteBytes:         8192,
			StorageWriteBytesDisplay:  "8 KiB",
			Timestamp:                 "2026-10-22T10:00:00Z",
		},
	}}, nil
}

func mockListImages(_ context.Context, data string) (proto.Message, error) {
	var in podengine.ListImagesReq
	if err := decode(data, &in); err != nil {
		return nil, err
	}
	return &podengine.ListImagesRes{
		Total: 2,
		Items: []*podengine.ImagePb{
			{
				Id:          "sha256:mock-nginx",
				References:  []string{"nginx:latest"},
				Digests:     []string{"sha256:mock-nginx-digest"},
				Size:        142000000,
				SizeDisplay: "135 MiB",
				CreatedAt:   "2026-10-20T08:00:00Z",
			},
			{
				Id:          "sha256:mock-redis",
				References:  []string{"redis:7-alpine"},
				Digests:     []string{"sha256:mock-redis-digest"},
				Size:        41000000,
				SizeDisplay: "39 MiB",
				CreatedAt:   "2026-10-19T08:00:00Z",
			},
		},
	}, nil
}
