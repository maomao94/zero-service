package solo

import (
	"context"

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
		return nil, invalidRequestError("knowledge is disabled")
	}
	if req == nil {
		return nil, invalidRequestError("list knowledge documents request is required")
	}
	uid := ctxdata.GetUserId(l.ctx)
	if uid == "" {
		return nil, unauthenticatedError("missing user id")
	}
	baseID, err := requireKnowledgeBaseID(req.BaseId)
	if err != nil {
		return nil, err
	}
	list, err := l.svcCtx.Knowledge.ListDocuments(l.ctx, uid, baseID)
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
