package solo

import (
	"context"
	"errors"
	"strings"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryRagCollectionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryRagCollectionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryRagCollectionLogic {
	return &QueryRagCollectionLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *QueryRagCollectionLogic) QueryRagCollection(req *types.RagQueryRequest) (*types.RagQueryResponse, error) {
	if l.svcCtx.Rag == nil {
		return nil, errors.New("rag is disabled")
	}
	uid := ctxdata.GetUserId(l.ctx)
	if uid == "" {
		return nil, errors.New("missing user id")
	}
	q := strings.TrimSpace(req.Query)
	if q == "" {
		return nil, errors.New("query is required")
	}
	res, err := l.svcCtx.Rag.Retrieve(l.ctx, uid, req.CollectionId, q, req.TopK)
	if err != nil {
		return nil, err
	}
	hits := make([]types.RagHitInfo, 0, len(res.Hits))
	for _, h := range res.Hits {
		hits = append(hits, types.RagHitInfo{
			Text: h.Text, Score: h.Score, SourceId: h.SourceID, Filename: h.Filename,
		})
	}
	return &types.RagQueryResponse{Hits: hits, Context: res.Context}, nil
}
