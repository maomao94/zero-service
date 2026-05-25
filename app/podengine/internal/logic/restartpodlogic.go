package logic

import (
	"context"

	"zero-service/app/podengine/internal/svc"
	"zero-service/app/podengine/podengine"
	"zero-service/common/dockerx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

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
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST, "node: "+in.Node)
	}

	err = dockerClient.ContainerRestart(l.ctx, in.Id, container.StopOptions{})
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "failed to restart container")
	}

	containerInfo, err := dockerClient.ContainerInspect(l.ctx, in.Id)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "failed to inspect container after restart")
	}

	pod := &podengine.PodPb{
		Id:    containerInfo.ID,
		Name:  containerInfo.Name[1:], // Remove leading slash
		Phase: podengine.PodPhasePb_POD_PHASE_RUNNING,
		Containers: []*podengine.ContainerPb{
			{
				Name:  containerInfo.Name[1:],
				Image: containerInfo.Config.Image,
				State: &podengine.ContainerStatePb{
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
