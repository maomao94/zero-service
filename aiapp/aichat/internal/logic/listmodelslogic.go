package logic

import (
	"context"
	"time"

	"zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aichat/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListModelsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListModelsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListModelsLogic {
	return &ListModelsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListModelsLogic) ListModels(in *aichat.ListModelsReq) (*aichat.ListModelsRes, error) {
	var data []*aichat.ModelObjectPb
	now := time.Now().Unix()
	for _, mc := range l.svcCtx.Config.Models {
		data = append(data, &aichat.ModelObjectPb{
			Id:                mc.Id,
			Object:            "model",
			Created:           now,
			OwnedBy:           mc.Provider,
			DisplayName:       mc.DisplayName,
			Description:       mc.Description,
			SupportsStreaming: mc.SupportsStreaming,
			MaxTokens:         int32(mc.MaxTokens),
		})
	}
	return &aichat.ListModelsRes{Data: data}, nil
}
