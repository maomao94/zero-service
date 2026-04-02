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
	dockerClient, ok := l.svcCtx.GetDockerClient(in.Node)
	if !ok {
		return nil, fmt.Errorf("node %s not found", in.Node)
	}
	container, err := dockerClient.ContainerInspect(l.ctx, in.Id)
	if err != nil {
		return nil, fmt.Errorf("container not found: %s", in.Id)
	}
	pod := &podengine.PodPb{
		Id:          container.ID,
		Name:        container.Name[1:], // Remove leading slash
		Labels:      container.Config.Labels,
		Phase:       getPodPhase(container.State),
		NetworkMode: string(container.HostConfig.NetworkMode),
		Containers: []*podengine.ContainerPb{
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
	}
	createTime := carbon.Parse(container.Created)
	if createTime.IsValid() {
		pod.CreationTime = createTime.ToDateTimeString()
	}

	if container.State.Running {
		startTime := carbon.Parse(container.State.StartedAt)
		if startTime.IsValid() {
			pod.StartTime = startTime.ToDateTimeString()
		}
	}

	return &podengine.GetPodRes{
		Pod: pod,
	}, nil
}

func getPodPhase(state *container.State) podengine.PodPhasePb {
	if state.Running {
		return podengine.PodPhasePb_POD_PHASE_RUNNING
	} else if state.Status == "exited" {
		if state.ExitCode == 0 {
			return podengine.PodPhasePb_POD_PHASE_SUCCEEDED
		} else {
			return podengine.PodPhasePb_POD_PHASE_FAILED
		}
	} else if state.Status == "created" {
		return podengine.PodPhasePb_POD_PHASE_PENDING
	} else if state.Status == "stopped" {
		return podengine.PodPhasePb_POD_PHASE_STOPPED
	}
	return podengine.PodPhasePb_POD_PHASE_UNKNOWN
}

func getContainerState(state *container.State) *podengine.ContainerStatePb {
	containerState := &podengine.ContainerStatePb{
		Running:      state.Running,
		Terminated:   state.Status == "exited",
		Waiting:      state.Status == "created" || state.Status == "restarting",
		Reason:       state.Status,
		Message:      state.Error,
		FinishedTime: carbon.Parse(state.FinishedAt).ToDateTimeString(),
		ExitCode:     strconv.Itoa(state.ExitCode),
	}
	StartedTime := carbon.Parse(state.StartedAt)
	if StartedTime.IsValid() {
		containerState.StartedTime = StartedTime.ToDateTimeString()
	}
	FinishedTime := carbon.Parse(state.FinishedAt)
	if FinishedTime.IsValid() {
		containerState.FinishedTime = FinishedTime.ToDateTimeString()
	}
	return containerState
}
