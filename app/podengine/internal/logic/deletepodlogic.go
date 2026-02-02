package logic

import (
	"context"
	"fmt"

	"zero-service/app/podengine/internal/svc"
	"zero-service/app/podengine/podengine"

	"github.com/docker/docker/api/types/container"
	"github.com/zeromicro/go-zero/core/logx"
)

type DeletePodLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeletePodLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeletePodLogic {
	return &DeletePodLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeletePodLogic) DeletePod(in *podengine.DeletePodReq) (*podengine.DeletePodRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	removeOptions := container.RemoveOptions{
		RemoveVolumes: in.RemoveVolumes,
		RemoveLinks:   true,
		Force:         in.Force,
	}

	err = l.svcCtx.DockerClient.ContainerRemove(l.ctx, in.Id, removeOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to delete container: %w", err)
	}

	return &podengine.DeletePodRes{}, nil
}
