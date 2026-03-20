// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package ai

import (
	"context"
	"time"

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
	abilities := l.svcCtx.Config.Abilities
	now := time.Now().Unix()

	models := make([]types.ModelObject, 0, len(abilities))
	for _, ab := range abilities {
		models = append(models, types.ModelObject{
			Id:      ab.Id,
			Object:  "model",
			Created: now,
			OwnedBy: "aigtw",
			Metadata: &types.ModelMetadata{
				Ability:          ab.Ability,
				DisplayName:      ab.DisplayName,
				Description:      ab.Description,
				SupportsStreaming: ab.SupportsStreaming,
			},
		})
	}

	return &types.ListModelsResponse{
		Object: "list",
		Data:   models,
	}, nil
}
