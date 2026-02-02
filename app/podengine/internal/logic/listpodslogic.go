package logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"zero-service/app/podengine/internal/svc"
	"zero-service/app/podengine/podengine"

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
	filter := filters.NewArgs()

	// Add id filter if provided (exact match)
	if in.Id != "" {
		filter.Add("id", in.Id)
	}

	// Add name filter if provided (exact match)
	if in.Name != "" {
		filter.Add("name", in.Name)
	}

	// Add label filters if provided
	for key, value := range in.Labels {
		filter.Add("label", fmt.Sprintf("%s=%s", key, value))
	}

	containers, err := l.svcCtx.DockerClient.ContainerList(l.ctx, container.ListOptions{
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
		phase := podengine.PodPhase_POD_PHASE_UNKNOWN
		if container.State == "running" {
			phase = podengine.PodPhase_POD_PHASE_RUNNING
		} else if container.State == "exited" {
			phase = podengine.PodPhase_POD_PHASE_SUCCEEDED
		} else if container.State == "created" {
			phase = podengine.PodPhase_POD_PHASE_PENDING
		} else if container.State == "stopped" {
			phase = podengine.PodPhase_POD_PHASE_STOPPED
		}

		item := &podengine.ListPodItem{
			Id:         container.ID,
			Name:       containerName,
			Phase:      phase,
			CreateTime: carbon.Parse(time.Unix(container.Created, 0).Format(time.RFC3339)).ToDateTimeString(),
		}
		if container.Created > 0 {
			creatTime := carbon.Parse(time.Unix(container.Created, 0).Format(time.RFC3339))
			if creatTime.IsValid() {
				item.CreateTime = creatTime.ToDateTimeString()
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
