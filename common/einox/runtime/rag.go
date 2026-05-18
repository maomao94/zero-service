package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
)

type Retriever interface {
	Retrieve(ctx context.Context, query string, topK int) ([]*schema.Document, error)
}

type StaticRetriever struct {
	Documents []*schema.Document
	Err       error
}

func (r StaticRetriever) Retrieve(context.Context, string, int) ([]*schema.Document, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return append([]*schema.Document(nil), r.Documents...), nil
}

func DocumentsContext(docs []*schema.Document) string {
	var b strings.Builder
	for i, doc := range docs {
		if doc == nil || strings.TrimSpace(doc.Content) == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n---\n\n")
		}
		fmt.Fprintf(&b, "[%d] ", i+1)
		b.WriteString(strings.TrimSpace(doc.Content))
	}
	return b.String()
}

func AppendSystemContext(system, contextBlock string) string {
	contextBlock = strings.TrimSpace(contextBlock)
	if contextBlock == "" {
		return strings.TrimSpace(system)
	}
	system = strings.TrimSpace(system)
	var b strings.Builder
	if system != "" {
		b.WriteString(system)
		b.WriteString("\n\n")
	}
	b.WriteString("Use the following retrieved context when it is relevant. Cite bracketed source numbers when answering.\n\n")
	b.WriteString(contextBlock)
	return b.String()
}
