package logic

import (
	"context"
	"fmt"
	"strings"
	"zero-service/app/podengine/internal/svc"
	"zero-service/app/podengine/podengine"
	"zero-service/common/dockerx"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListPodsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListPodsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPodsLogic {
	return &ListPodsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListPodsLogic) ListPods(in *podengine.ListPodsReq) (*podengine.ListPodsRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	dockerClient, ok := l.svcCtx.GetDockerClient(in.Node)
	if !ok {
		return nil, fmt.Errorf("node %s not found", in.Node)
	}
	filter := filters.NewArgs()

	// Add id filter if provided (exact match)
	if len(in.Ids) > 0 {
		for _, id := range in.Ids {
			filter.Add("id", id)
		}
	}

	// Add name filter if provided (exact match)
	if len(in.Names) > 0 {
		for _, name := range in.Names {
			filter.Add("name", name)
		}
	}

	// Add label filters if provided
	for key, value := range in.Labels {
		filter.Add("label", fmt.Sprintf("%s=%s", key, value))
	}

	containers, err := dockerClient.ContainerList(l.ctx, container.ListOptions{
		All:     true,
		Filters: filter,
	})
	if err != nil {
		l.Errorf("Failed to list containers: %v", err)
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var items []*podengine.ListPodItem
	for _, container := range containers {
		containerName := strings.TrimPrefix(container.Names[0], "/")
		phase := getPodPhaseFromStatus(container.State)

		// 构建端口字符串
		var ports []string
		for _, port := range container.Ports {
			if port.PublicPort > 0 {
				ports = append(ports, fmt.Sprintf("%s:%d->%d/%s", port.IP, port.PublicPort, port.PrivatePort, port.Type))
			} else {
				ports = append(ports, fmt.Sprintf("%d/%s", port.PrivatePort, port.Type))
			}
		}

		item := &podengine.ListPodItem{
			Id:          container.ID,
			Name:        containerName,
			Phase:       phase,
			Image:       container.Image,
			ImageId:     container.ImageID,
			Command:     container.Command,
			Ports:       ports,
			SizeRw:      container.SizeRw,
			SizeRootFs:  container.SizeRootFs,
			Labels:      container.Labels,
			State:       container.State,
			Status:      container.Status,
			NetworkMode: container.HostConfig.NetworkMode,
			Mounts:      dockerx.ExtractContainerVolumeMounts(container.Mounts),
		}
		if container.Created > 0 {
			createTime := carbon.CreateFromTimestamp(container.Created)
			if createTime.IsValid() {
				item.CreateTime = createTime.ToDateTimeString()
			}
		}
		items = append(items, item)
	}
	total := len(items)
	if in.Offset > int32(total) {
		items = []*podengine.ListPodItem{}
	} else {
		end := in.Offset + in.Limit
		if end > int32(total) {
			end = int32(total)
		}
		items = items[in.Offset:end]
	}

	return &podengine.ListPodsRes{
		Items: items,
		Total: int32(total),
	}, nil
}

func getPodPhaseFromStatus(state string) podengine.PodPhase {
	switch state {
	case "running":
		return podengine.PodPhase_POD_PHASE_RUNNING
	case "exited":
		return podengine.PodPhase_POD_PHASE_SUCCEEDED
	case "created":
		return podengine.PodPhase_POD_PHASE_PENDING
	case "stopped":
		return podengine.PodPhase_POD_PHASE_STOPPED
	default:
		return podengine.PodPhase_POD_PHASE_UNKNOWN
	}
}
