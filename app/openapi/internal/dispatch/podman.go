package dispatch

import "zero-service/app/podengine/podengine"

// Podman 网关方法。
const (
	MethodPodList   = "pod.list"
	MethodPodGet    = "pod.get"
	MethodPodStats  = "pod.stats"
	MethodImageList = "image.list"
)

func podmanRoutes(client podengine.PodEngineClient) map[string]Handler {
	return map[string]Handler{
		MethodPodList:   relay(client.ListPods),
		MethodPodGet:    relay(client.GetPod),
		MethodPodStats:  relay(client.GetPodStats),
		MethodImageList: relay(client.ListImages),
	}
}
