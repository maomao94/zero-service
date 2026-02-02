package logic

import (
	"context"
	"fmt"
	"strconv"

	"zero-service/app/podengine/internal/svc"
	"zero-service/app/podengine/podengine"
	"zero-service/common/dockerx"

	"github.com/docker/docker/api/types/container"
	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetPodLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetPodLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPodLogic {
	return &GetPodLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetPodLogic) GetPod(in *podengine.GetPodReq) (*podengine.GetPodRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	container, err := l.svcCtx.DockerClient.ContainerInspect(l.ctx, in.Id)
	if err != nil {
		return nil, fmt.Errorf("container not found: %s", in.Id)
	}
	pod := &podengine.Pod{
		Id:     container.ID,
		Name:   container.Name[1:], // Remove leading slash
		Labels: container.Config.Labels,
		Phase:  getPodPhase(container.State),
		Containers: []*podengine.Container{
			{
				Name:         container.Name[1:],
				Image:        container.Config.Image,
				State:        getContainerState(container.State),
				Ports:        dockerx.ExtractContainerPorts(container.NetworkSettings),
				Env:          dockerx.ParseContainerEnv(container.Config.Env),
				Args:         container.Config.Cmd,
				Resources:    dockerx.ParseContainerResources(container.HostConfig.Resources),
				VolumeMounts: dockerx.ExtractContainerVolumeMounts(container.Mounts),
			},
		},
		CreationTime: carbon.Parse(container.Created).ToDateTimeString(),
	}

	if container.State.Running {
		pod.StartTime = carbon.Parse(container.State.StartedAt).ToDateTimeString()
	}

	return &podengine.GetPodRes{
		Pod: pod,
	}, nil
}

func getPodPhase(state *container.State) podengine.PodPhase {
	if state.Running {
		return podengine.PodPhase_POD_PHASE_RUNNING
	} else if state.Status == "exited" {
		if state.ExitCode == 0 {
			return podengine.PodPhase_POD_PHASE_SUCCEEDED
		} else {
			return podengine.PodPhase_POD_PHASE_FAILED
		}
	} else if state.Status == "created" {
		return podengine.PodPhase_POD_PHASE_PENDING
	} else if state.Status == "stopped" {
		return podengine.PodPhase_POD_PHASE_STOPPED
	}
	return podengine.PodPhase_POD_PHASE_UNKNOWN
}

func getContainerState(state *container.State) *podengine.ContainerState {
	return &podengine.ContainerState{
		Running:      state.Running,
		Terminated:   state.Status == "exited",
		Waiting:      state.Status == "created" || state.Status == "restarting",
		Reason:       state.Status,
		Message:      state.Error,
		StartedTime:  carbon.Parse(state.StartedAt).ToDateTimeString(),
		FinishedTime: carbon.Parse(state.FinishedAt).ToDateTimeString(),
		ExitCode:     strconv.Itoa(state.ExitCode),
	}
}
