package logic

import (
	"context"
	"fmt"

	"zero-service/app/podengine/internal/svc"
	"zero-service/app/podengine/podengine"

	"github.com/docker/docker/api/types/container"
	"github.com/zeromicro/go-zero/core/logx"
)

type StopPodLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStopPodLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StopPodLogic {
	return &StopPodLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *StopPodLogic) StopPod(in *podengine.StopPodReq) (*podengine.StopPodRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	dockerClient, ok := l.svcCtx.GetDockerClient(in.Node)
	if !ok {
		return nil, fmt.Errorf("node %s not found", in.Node)
	}

	err = dockerClient.ContainerStop(l.ctx, in.Id, container.StopOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to stop container: %w", err)
	}

	_, err = dockerClient.ContainerInspect(l.ctx, in.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}
	return &podengine.StopPodRes{}, nil
}
