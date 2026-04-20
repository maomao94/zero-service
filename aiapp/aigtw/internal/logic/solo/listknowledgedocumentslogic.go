package solo

import (
	"context"
	"errors"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListKnowledgeDocumentsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListKnowledgeDocumentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListKnowledgeDocumentsLogic {
	return &ListKnowledgeDocumentsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ListKnowledgeDocumentsLogic) ListKnowledgeDocuments(req *types.KnowledgeListDocumentsRequest) (*types.KnowledgeListDocumentsResponse, error) {
	if l.svcCtx.Knowledge == nil {
		return nil, errors.New("knowledge is disabled")
	}
	uid := ctxdata.GetUserId(l.ctx)
	if uid == "" {
		return nil, errors.New("missing user id")
	}
	list, err := l.svcCtx.Knowledge.ListDocuments(l.ctx, uid, req.BaseId)
	if err != nil {
		return nil, err
	}
	out := make([]types.KnowledgeDocumentInfo, 0, len(list))
	for _, s := range list {
		out = append(out, types.KnowledgeDocumentInfo{
			Id: s.ID, Filename: s.Filename, Chunks: s.Chunks, CreatedAt: s.CreatedAt.Unix(),
		})
	}
	return &types.KnowledgeListDocumentsResponse{Documents: out}, nil
}
