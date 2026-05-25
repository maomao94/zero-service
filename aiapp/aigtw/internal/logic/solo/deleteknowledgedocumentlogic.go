package solo

import (
	"context"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteKnowledgeDocumentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteKnowledgeDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteKnowledgeDocumentLogic {
	return &DeleteKnowledgeDocumentLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *DeleteKnowledgeDocumentLogic) DeleteKnowledgeDocument(req *types.KnowledgeDeleteDocumentRequest) (*types.KnowledgeDeleteDocumentResponse, error) {
	if l.svcCtx.Knowledge == nil {
		return nil, invalidRequestError("knowledge is disabled")
	}
	if req == nil {
		return nil, invalidRequestError("delete knowledge document request is required")
	}
	uid := ctxdata.GetUserId(l.ctx)
	if uid == "" {
		return nil, unauthenticatedError("missing user id")
	}
	baseID, err := requireKnowledgeBaseID(req.BaseId)
	if err != nil {
		return nil, err
	}
	sourceID, err := requireKnowledgeDocumentID(req.SourceId)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.Knowledge.DeleteDocument(l.ctx, uid, baseID, sourceID); err != nil {
		return nil, err
	}
	return &types.KnowledgeDeleteDocumentResponse{Success: true}, nil
}
