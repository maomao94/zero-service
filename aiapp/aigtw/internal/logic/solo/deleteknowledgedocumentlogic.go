package solo

import (
	"context"
	"errors"

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
		return nil, errors.New("knowledge is disabled")
	}
	uid := ctxdata.GetUserId(l.ctx)
	if uid == "" {
		return nil, errors.New("missing user id")
	}
	if err := l.svcCtx.Knowledge.DeleteDocument(l.ctx, uid, req.BaseId, req.SourceId); err != nil {
		return nil, err
	}
	return &types.KnowledgeDeleteDocumentResponse{Success: true}, nil
}
