package solo

import (
	"context"
	"fmt"
	"strings"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type IngestKnowledgeDocumentsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewIngestKnowledgeDocumentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IngestKnowledgeDocumentsLogic {
	return &IngestKnowledgeDocumentsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *IngestKnowledgeDocumentsLogic) IngestKnowledgeDocuments(req *types.KnowledgeIngestBatchRequest) (*types.KnowledgeIngestBatchResponse, error) {
	if l.svcCtx.Knowledge == nil {
		return nil, invalidRequestError("knowledge is disabled")
	}
	if req == nil {
		return nil, invalidRequestError("ingest batch request is required")
	}
	uid := ctxdata.GetUserId(l.ctx)
	if uid == "" {
		return nil, unauthenticatedError("missing user id")
	}
	baseID, err := requireKnowledgeBaseID(req.BaseId)
	if err != nil {
		return nil, err
	}
	if len(req.Items) == 0 {
		return nil, invalidRequestError("items is required")
	}
	results := make([]types.KnowledgeIngestBatchResultItem, 0, len(req.Items))
	for i, it := range req.Items {
		fn := strings.TrimSpace(it.Filename)
		if fn == "" {
			fn = fmt.Sprintf("document_%d.txt", i+1)
		}
		if strings.TrimSpace(it.Content) == "" {
			results = append(results, types.KnowledgeIngestBatchResultItem{Filename: fn, Error: "empty content"})
			continue
		}
		src, err := l.svcCtx.Knowledge.IngestDocument(l.ctx, uid, baseID, fn, strings.TrimSpace(it.Content))
		if err != nil {
			results = append(results, types.KnowledgeIngestBatchResultItem{Filename: fn, Error: err.Error()})
			continue
		}
		results = append(results, types.KnowledgeIngestBatchResultItem{
			Filename: src.Filename,
			SourceId: src.ID,
			Chunks:   src.Chunks,
		})
	}
	return &types.KnowledgeIngestBatchResponse{Results: results}, nil
}
