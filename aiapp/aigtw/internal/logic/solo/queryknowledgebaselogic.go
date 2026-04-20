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

type QueryKnowledgeBaseLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryKnowledgeBaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryKnowledgeBaseLogic {
	return &QueryKnowledgeBaseLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *QueryKnowledgeBaseLogic) QueryKnowledgeBase(req *types.KnowledgeQueryRequest) (*types.KnowledgeQueryResponse, error) {
	if l.svcCtx.Knowledge == nil {
		return nil, errors.New("knowledge is disabled")
	}
	uid := ctxdata.GetUserId(l.ctx)
	if uid == "" {
		return nil, errors.New("missing user id")
	}
	q := strings.TrimSpace(req.Query)
	if q == "" {
		return &types.KnowledgeQueryResponse{}, nil
	}
	res, err := l.svcCtx.Knowledge.Search(l.ctx, uid, req.BaseId, q, req.TopK)
	if err != nil {
		return nil, err
	}
	hits := make([]types.KnowledgeCitationInfo, 0, len(res.Hits))
	for _, h := range res.Hits {
		hits = append(hits, types.KnowledgeCitationInfo{
			Text: h.Text, Score: h.Score, SourceId: h.SourceID, Filename: h.Filename,
		})
	}
	return &types.KnowledgeQueryResponse{Hits: hits, Context: res.Context}, nil
}
