package logic

import (
	"context"

	"zero-service/app/podengine/internal/svc"
	"zero-service/app/podengine/podengine"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

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
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST, "node: "+in.Node)
	}

	err = dockerClient.ContainerStop(l.ctx, in.Id, container.StopOptions{})
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "failed to stop container")
	}

	_, err = dockerClient.ContainerInspect(l.ctx, in.Id)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "failed to inspect container after stop")
	}
	return &podengine.StopPodRes{}, nil
}
