package solo

import (
	"context"
	"errors"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListRagSourcesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListRagSourcesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListRagSourcesLogic {
	return &ListRagSourcesLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ListRagSourcesLogic) ListRagSources(req *types.RagListSourcesRequest) (*types.RagListSourcesResponse, error) {
	if l.svcCtx.Rag == nil {
		return nil, errors.New("rag is disabled")
	}
	uid := ctxdata.GetUserId(l.ctx)
	if uid == "" {
		return nil, errors.New("missing user id")
	}
	list, err := l.svcCtx.Rag.ListSources(l.ctx, uid, req.CollectionId)
	if err != nil {
		return nil, err
	}
	out := make([]types.RagSourceInfo, 0, len(list))
	for _, s := range list {
		out = append(out, types.RagSourceInfo{
			Id: s.ID, Filename: s.Filename, Chunks: s.Chunks, CreatedAt: s.CreatedAt.Unix(),
		})
	}
	return &types.RagListSourcesResponse{Sources: out}, nil
}
