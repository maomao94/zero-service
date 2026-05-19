package solo

import (
	"context"
	"testing"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/common/ctxdata"
	einoxkb "zero-service/common/einox/knowledge"
)

func TestKnowledgeLogicsRequireRequestBeforeSDK(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxdata.CtxUserIdKey, "user-1")
	svcCtx := &svc.ServiceContext{Knowledge: nil}

	cases := []struct {
		name string
		run  func() error
		want string
	}{
		{name: "create disabled precedence", run: func() error {
			_, err := NewCreateKnowledgeBaseLogic(ctx, svcCtx).CreateKnowledgeBase(nil)
			return err
		}, want: "knowledge is disabled"},
		{name: "delete disabled precedence", run: func() error {
			_, err := NewDeleteKnowledgeBaseLogic(ctx, svcCtx).DeleteKnowledgeBase(nil)
			return err
		}, want: "knowledge is disabled"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertValidationError(t, tc.run(), tc.want)
		})
	}
}

func TestKnowledgeValidationHelpers(t *testing.T) {
	if _, err := requireKnowledgeBaseID(" \t"); err == nil {
		t.Fatal("requireKnowledgeBaseID() error = nil, want required error")
	}
	if got, err := requireKnowledgeBaseID(" base-1 "); err != nil || got != "base-1" {
		t.Fatalf("requireKnowledgeBaseID() = (%q, %v), want trimmed base", got, err)
	}
	if _, err := requireKnowledgeDocumentID(" \t"); err == nil {
		t.Fatal("requireKnowledgeDocumentID() error = nil, want required error")
	}
	if got, err := requireKnowledgeDocumentID(" source-1 "); err != nil || got != "source-1" {
		t.Fatalf("requireKnowledgeDocumentID() = (%q, %v), want trimmed source", got, err)
	}
	if _, err := requireKnowledgeQuery(" \t"); err == nil {
		t.Fatal("requireKnowledgeQuery() error = nil, want required error")
	}
	if got, err := requireKnowledgeQuery(" what is eino? "); err != nil || got != "what is eino?" {
		t.Fatalf("requireKnowledgeQuery() = (%q, %v), want trimmed query", got, err)
	}
	if _, err := requireKnowledgeContent(" \t"); err == nil {
		t.Fatal("requireKnowledgeContent() error = nil, want required error")
	}
	if got, err := requireKnowledgeContent(" content "); err != nil || got != "content" {
		t.Fatalf("requireKnowledgeContent() = (%q, %v), want trimmed content", got, err)
	}
}

func TestKnowledgeBaseIDValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		run  func() error
	}{
		{name: "delete base", run: func() error {
			_, err := NewDeleteKnowledgeBaseLogic(knowledgeTestContext(), knowledgeValidationSvc()).DeleteKnowledgeBase(&types.KnowledgeDeleteBaseRequest{})
			return err
		}},
		{name: "ingest single", run: func() error {
			_, err := NewIngestKnowledgeDocumentLogic(knowledgeTestContext(), knowledgeValidationSvc()).IngestKnowledgeDocument(&types.KnowledgeIngestRequest{Content: "hello"})
			return err
		}},
		{name: "ingest batch", run: func() error {
			_, err := NewIngestKnowledgeDocumentsLogic(knowledgeTestContext(), knowledgeValidationSvc()).IngestKnowledgeDocuments(&types.KnowledgeIngestBatchRequest{Items: []types.KnowledgeIngestItem{{Content: "hello"}}})
			return err
		}},
		{name: "query", run: func() error {
			_, err := NewQueryKnowledgeBaseLogic(knowledgeTestContext(), knowledgeValidationSvc()).QueryKnowledgeBase(&types.KnowledgeQueryRequest{Query: "hello"})
			return err
		}},
		{name: "list documents", run: func() error {
			_, err := NewListKnowledgeDocumentsLogic(knowledgeTestContext(), knowledgeValidationSvc()).ListKnowledgeDocuments(&types.KnowledgeListDocumentsRequest{})
			return err
		}},
		{name: "delete document", run: func() error {
			_, err := NewDeleteKnowledgeDocumentLogic(knowledgeTestContext(), knowledgeValidationSvc()).DeleteKnowledgeDocument(&types.KnowledgeDeleteDocumentRequest{SourceId: "source-1"})
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertValidationError(t, tc.run(), "baseId is required")
		})
	}
}

func TestKnowledgeQueryRequiresQueryBeforeSDK(t *testing.T) {
	_, err := NewQueryKnowledgeBaseLogic(knowledgeTestContext(), knowledgeValidationSvc()).QueryKnowledgeBase(&types.KnowledgeQueryRequest{BaseId: "base-1"})
	assertValidationError(t, err, "query is required")
}

func TestKnowledgeIngestRequiresContentBeforeSDK(t *testing.T) {
	_, err := NewIngestKnowledgeDocumentLogic(knowledgeTestContext(), knowledgeValidationSvc()).IngestKnowledgeDocument(&types.KnowledgeIngestRequest{BaseId: "base-1"})
	assertValidationError(t, err, "content is required")
}

func TestKnowledgeBatchIngestRequiresItemsBeforeSDK(t *testing.T) {
	_, err := NewIngestKnowledgeDocumentsLogic(knowledgeTestContext(), knowledgeValidationSvc()).IngestKnowledgeDocuments(&types.KnowledgeIngestBatchRequest{BaseId: "base-1"})
	assertValidationError(t, err, "items is required")
}

func TestKnowledgeDeleteDocumentRequiresSourceIDBeforeSDK(t *testing.T) {
	_, err := NewDeleteKnowledgeDocumentLogic(knowledgeTestContext(), knowledgeValidationSvc()).DeleteKnowledgeDocument(&types.KnowledgeDeleteDocumentRequest{BaseId: "base-1"})
	assertValidationError(t, err, "sourceId is required")
}

func knowledgeTestContext() context.Context {
	return context.WithValue(context.Background(), ctxdata.CtxUserIdKey, "user-1")
}

func knowledgeValidationSvc() *svc.ServiceContext {
	return &svc.ServiceContext{Knowledge: nonNilKnowledgeForValidation()}
}

func nonNilKnowledgeForValidation() *einoxkb.Service {
	return &einoxkb.Service{}
}
