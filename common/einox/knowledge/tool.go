package knowledge

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type searchToolIn struct {
	Query string `json:"query" jsonschema:"description=要向量化检索的问题或关键词"`
	TopK  int    `json:"top_k,omitempty" jsonschema:"description=返回条数；0 表示使用服务默认 topK"`
}

type searchToolHit struct {
	Text     string  `json:"text"`
	Score    float64 `json:"score"`
	Filename string  `json:"filename,omitempty"`
	SourceID string  `json:"sourceId,omitempty"`
}

type searchToolOut struct {
	Context string          `json:"context"`
	Hits    []searchToolHit `json:"hits,omitempty"`
	Notice  string          `json:"notice,omitempty"`
}

// NewSearchTool 构造会话级知识库检索工具；依赖 WithAgentTurn 注入 user / knowledge base。
// svc 为 nil 时不应调用（调用方跳过挂载）。
func NewSearchTool(svc *Service) (tool.BaseTool, error) {
	if svc == nil {
		return nil, nil
	}
	return utils.InferTool(
		"search_knowledge_base",
		"在当前会话已绑定的向量知识库中检索与问题相关的文本片段，返回可引用上下文。"+
			"用户询问上传的文档、内部资料时优先调用；若会话未绑定知识库，会返回说明。",
		func(ctx context.Context, in *searchToolIn) (*searchToolOut, error) {
			q := strings.TrimSpace(in.Query)
			if q == "" {
				return &searchToolOut{Notice: "query 为空"}, nil
			}
			uid := UserIDFrom(ctx)
			baseID := KnowledgeBaseIDFrom(ctx)
			if uid == "" {
				return &searchToolOut{Notice: "服务端未注入用户上下文，无法检索"}, nil
			}
			if baseID == "" {
				return &searchToolOut{Notice: "当前会话未绑定知识库：请在网关侧将知识库 ID 绑定到会话，或创建会话时传入 knowledgeBaseId。"}, nil
			}
			topK := in.TopK
			res, err := svc.Search(ctx, uid, baseID, q, topK)
			if err != nil {
				return nil, err
			}
			out := &searchToolOut{Context: res.Context}
			for _, h := range res.Hits {
				out.Hits = append(out.Hits, searchToolHit{
					Text: h.Text, Score: h.Score, Filename: h.Filename, SourceID: h.SourceID,
				})
			}
			if len(out.Hits) == 0 && out.Context == "" {
				out.Notice = fmt.Sprintf("未在知识库中找到与 %q 相关的内容", q)
			}
			return out, nil
		},
	)
}
