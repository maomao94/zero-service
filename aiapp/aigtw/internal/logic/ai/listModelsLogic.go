// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package ai

import (
	"context"

	"zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListModelsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 列出可用模型
func NewListModelsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListModelsLogic {
	return &ListModelsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListModelsLogic) ListModels() (resp *types.ListModelsResponse, err error) {
	rpcResp, err := l.svcCtx.AiChatCli.ListModels(l.ctx, &aichat.ListModelsReq{})
	if err != nil {
		return nil, err
	}

	models := make([]types.ModelObject, 0, len(rpcResp.Data))
	for _, m := range rpcResp.Data {
		models = append(models, types.ModelObject{
			Id:      m.Id,
			Object:  m.Object,
			Created: m.Created,
			OwnedBy: m.OwnedBy,
			Metadata: &types.ModelMetadata{
				DisplayName:       m.DisplayName,
				Description:       m.Description,
				SupportsStreaming: m.SupportsStreaming,
			},
		})
	}

	return &types.ListModelsResponse{
		Object: "list",
		Data:   models,
	}, nil
}
