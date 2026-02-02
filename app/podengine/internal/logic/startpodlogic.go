package logic

import (
	"context"
	"fmt"
	"zero-service/app/podengine/internal/svc"
	"zero-service/app/podengine/podengine"
	"zero-service/common/dockerx"

	"github.com/docker/docker/api/types/container"
	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

type StartPodLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStartPodLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartPodLogic {
	return &StartPodLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *StartPodLogic) StartPod(in *podengine.StartPodReq) (*podengine.StartPodRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	dockerClient, ok := l.svcCtx.GetDockerClient(in.Node)
	if !ok {
		return nil, fmt.Errorf("node %s not found", in.Node)
	}

	// Start the container
	err = dockerClient.ContainerStart(l.ctx, in.Id, container.StartOptions{})
	if err != nil {
		l.Errorf("Failed to start container %s: %v", in.Id, err)
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Inspect the container to get updated information
	containerInfo, err := dockerClient.ContainerInspect(l.ctx, in.Id)
	if err != nil {
		l.Errorf("Failed to inspect container %s: %v", in.Id, err)
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	// Build the pod response
	pod := &podengine.Pod{
		Id:    containerInfo.ID,
		Name:  containerInfo.Name[1:], // Remove leading slash
		Phase: podengine.PodPhase_POD_PHASE_RUNNING,
		Containers: []*podengine.Container{
			{
				Name:  containerInfo.Name[1:],
				Image: containerInfo.Config.Image,
				State: &podengine.ContainerState{
					Running:      true,
					Terminated:   false,
					Waiting:      false,
					Reason:       containerInfo.State.Status,
					Message:      "",
					StartedTime:  carbon.Parse(containerInfo.State.StartedAt).ToDateTimeString(),
					FinishedTime: "",
					ExitCode:     "0",
				},
				Ports:        dockerx.ExtractContainerPorts(containerInfo.NetworkSettings),
				Env:          dockerx.ParseContainerEnv(containerInfo.Config.Env),
				Args:         containerInfo.Config.Cmd,
				Resources:    dockerx.ParseContainerResources(containerInfo.HostConfig.Resources),
				VolumeMounts: dockerx.ExtractContainerVolumeMounts(containerInfo.Mounts),
			},
		},
		Labels:       containerInfo.Config.Labels,
		CreationTime: carbon.Parse(containerInfo.Created).ToDateTimeString(),
		StartTime:    carbon.Parse(containerInfo.State.StartedAt).ToDateTimeString(),
	}

	return &podengine.StartPodRes{
		Pod: pod,
	}, nil
}
