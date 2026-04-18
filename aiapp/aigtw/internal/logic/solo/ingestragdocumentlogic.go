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

type IngestRagDocumentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewIngestRagDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IngestRagDocumentLogic {
	return &IngestRagDocumentLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *IngestRagDocumentLogic) IngestRagDocument(req *types.RagIngestRequest) (*types.RagIngestResponse, error) {
	if l.svcCtx.Rag == nil {
		return nil, errors.New("rag is disabled")
	}
	uid := ctxdata.GetUserId(l.ctx)
	if uid == "" {
		return nil, errors.New("missing user id")
	}
	if strings.TrimSpace(req.Content) == "" {
		return nil, errors.New("content is required")
	}
	src, err := l.svcCtx.Rag.IngestText(l.ctx, uid, req.CollectionId, req.Filename, req.Content)
	if err != nil {
		return nil, err
	}
	return &types.RagIngestResponse{SourceId: src.ID, Chunks: src.Chunks}, nil
}
