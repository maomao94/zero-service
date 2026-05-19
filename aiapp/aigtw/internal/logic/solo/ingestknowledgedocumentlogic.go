package solo

import (
	"context"
	"errors"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type IngestKnowledgeDocumentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewIngestKnowledgeDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IngestKnowledgeDocumentLogic {
	return &IngestKnowledgeDocumentLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *IngestKnowledgeDocumentLogic) IngestKnowledgeDocument(req *types.KnowledgeIngestRequest) (*types.KnowledgeIngestResponse, error) {
	if l.svcCtx.Knowledge == nil {
		return nil, errors.New("knowledge is disabled")
	}
	if req == nil {
		return nil, errors.New("ingest request is required")
	}
	uid := ctxdata.GetUserId(l.ctx)
	if uid == "" {
		return nil, errors.New("missing user id")
	}
	baseID, err := requireKnowledgeBaseID(req.BaseId)
	if err != nil {
		return nil, err
	}
	content, err := requireKnowledgeContent(req.Content)
	if err != nil {
		return nil, err
	}
	src, err := l.svcCtx.Knowledge.IngestDocument(l.ctx, uid, baseID, req.Filename, content)
	if err != nil {
		return nil, err
	}
	return &types.KnowledgeIngestResponse{SourceId: src.ID, Chunks: src.Chunks}, nil
}
