package solo

import (
	"context"
	"errors"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListRagCollectionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListRagCollectionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListRagCollectionsLogic {
	return &ListRagCollectionsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ListRagCollectionsLogic) ListRagCollections() (*types.RagListCollectionsResponse, error) {
	if l.svcCtx.Rag == nil {
		return nil, errors.New("rag is disabled")
	}
	uid := ctxdata.GetUserId(l.ctx)
	if uid == "" {
		return nil, errors.New("missing user id")
	}
	list, err := l.svcCtx.Rag.ListCollections(l.ctx, uid)
	if err != nil {
		return nil, err
	}
	out := make([]types.RagCollectionInfo, 0, len(list))
	for _, c := range list {
		out = append(out, types.RagCollectionInfo{Id: c.ID, Name: c.Name, CreatedAt: c.CreatedAt.Unix()})
	}
	return &types.RagListCollectionsResponse{Collections: out}, nil
}
