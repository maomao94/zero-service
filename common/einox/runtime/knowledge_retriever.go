package runtime

import (
	"context"

	"github.com/cloudwego/eino/schema"

	"zero-service/common/einox/knowledge"
)

type KnowledgeRetriever struct {
	service *knowledge.Service
}

func NewKnowledgeRetriever(service *knowledge.Service) Retriever {
	if service == nil {
		return nil
	}
	return KnowledgeRetriever{service: service}
}

func (r KnowledgeRetriever) Retrieve(ctx context.Context, query string, topK int) ([]*schema.Document, error) {
	if r.service == nil {
		return nil, nil
	}
	userID := knowledge.UserIDFrom(ctx)
	baseID := knowledge.KnowledgeBaseIDFrom(ctx)
	if userID == "" || baseID == "" {
		return nil, nil
	}
	result, err := r.service.Search(ctx, userID, baseID, query, topK)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.Documents, nil
}

var _ Retriever = KnowledgeRetriever{}
