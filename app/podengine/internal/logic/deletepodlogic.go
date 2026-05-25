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
	dockerClient, ok := l.svcCtx.GetDockerClient(in.Node)
	if !ok {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST, "node: "+in.Node)
	}
	removeOptions := container.RemoveOptions{
		RemoveVolumes: in.RemoveVolumes,
		RemoveLinks:   false,
		Force:         in.Force,
	}

	err = dockerClient.ContainerRemove(l.ctx, in.Id, removeOptions)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "failed to delete container")
	}

	return &podengine.DeletePodRes{}, nil
}
