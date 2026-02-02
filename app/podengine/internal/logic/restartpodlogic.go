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

type RestartPodLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRestartPodLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RestartPodLogic {
	return &RestartPodLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RestartPodLogic) RestartPod(in *podengine.RestartPodReq) (*podengine.RestartPodRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	dockerClient, ok := l.svcCtx.GetDockerClient(in.Node)
	if !ok {
		return nil, fmt.Errorf("node %s not found", in.Node)
	}

	err = dockerClient.ContainerRestart(l.ctx, in.Id, container.StopOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to restart container: %w", err)
	}

	containerInfo, err := dockerClient.ContainerInspect(l.ctx, in.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

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

	return &podengine.RestartPodRes{
		Pod: pod,
	}, nil
}
