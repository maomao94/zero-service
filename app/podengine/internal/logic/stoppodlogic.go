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

	err = l.svcCtx.DockerClient.ContainerStop(l.ctx, in.Id, container.StopOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to stop container: %w", err)
	}

	_, err = l.svcCtx.DockerClient.ContainerInspect(l.ctx, in.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}
	return &podengine.StopPodRes{}, nil
}
