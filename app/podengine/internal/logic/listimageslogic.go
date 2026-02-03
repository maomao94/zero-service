package logic

import (
	"context"
	"fmt"
	"zero-service/app/podengine/internal/svc"
	"zero-service/app/podengine/podengine"
	"zero-service/common/tool"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListImagesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListImagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListImagesLogic {
	return &ListImagesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListImagesLogic) ListImages(in *podengine.ListImagesReq) (*podengine.ListImagesRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	dockerClient, ok := l.svcCtx.GetDockerClient(in.Node)
	if !ok {
		return nil, fmt.Errorf("node %s not found", in.Node)
	}

	filter := filters.NewArgs()

	if len(in.References) > 0 {
		for _, reference := range in.References {
			filter.Add("reference", reference)
		}
	}

	images, err := dockerClient.ImageList(l.ctx, image.ListOptions{
		All:     true,
		Filters: filter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	var items []*podengine.Image
	for _, img := range images {
		// Process image tags
		references := make([]string, 0)
		if len(img.RepoTags) > 0 {
			references = img.RepoTags
		}

		// Get image labels
		labels := make(map[string]string)
		if img.Labels != nil {
			labels = img.Labels
		}

		// Get image digests by inspecting the image
		digests := make([]string, 0)
		if in.IncludeDigests {
			if inspect, err := dockerClient.ImageInspect(l.ctx, img.ID); err == nil {
				digests = inspect.RepoDigests

			}
		}
		image := &podengine.Image{
			Id:          img.ID,
			References:  references,
			Digests:     digests,
			Size:        img.Size,
			SizeDisplay: tool.BinaryBytes(img.Size, 2),
			Labels:      labels,
		}

		if img.Created > 0 {
			image.CreatedAt = carbon.CreateFromTimestamp(img.Created).ToDateTimeString()
		}

		items = append(items, image)
	}

	total := len(items)
	if in.Offset > int32(total) {
		items = []*podengine.Image{}
	} else {
		end := in.Offset + in.Limit
		if end > int32(total) {
			end = int32(total)
		}
		items = items[in.Offset:end]
	}

	return &podengine.ListImagesRes{
		Items: items,
		Total: int32(total),
	}, nil
}
