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
	uid := ctxdata.GetUserId(l.ctx)
	if uid == "" {
		return nil, errors.New("missing user id")
	}
	if strings.TrimSpace(req.Content) == "" {
		return nil, errors.New("content is required")
	}
	src, err := l.svcCtx.Knowledge.IngestDocument(l.ctx, uid, req.BaseId, req.Filename, req.Content)
	if err != nil {
		return nil, err
	}
	return &types.KnowledgeIngestResponse{SourceId: src.ID, Chunks: src.Chunks}, nil
}
