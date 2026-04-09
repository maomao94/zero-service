package router

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// IntentResult 意图识别结果
type IntentResult struct {
	Intent        string   `json:"intent"`         // fast/deep/multi
	Confidence    float64  `json:"confidence"`     // 置信度 0.0-1.0
	Reasoning     string   `json:"reasoning"`      // 判断理由
	SubAgents     []string `json:"sub_agents"`     // 需要调用的子 Agent
	SelectedAgent string   `json:"selected_agent"` // 选中的 Agent 类型
}

// IntentClassifier 意图分类器
type IntentClassifier struct {
	model  model.BaseChatModel
	prompt string
}

// intentPrompt 意图分类提示词
const intentPrompt = `你是一个智能助手路由器。根据用户输入，判断最合适的响应模式：

可选模式：
- fast: 简单问答，不需要工具调用，一次性回复
- deep: 深度分析，需要多轮推理或复杂工具调用
- multi: 多 Agent 协作，需要多个子 Agent 配合完成

请以 JSON 格式返回分析结果：
{
    "intent": "模式(fast/deep/multi)",
    "confidence": 置信度(0.0-1.0),
    "reasoning": "判断理由",
    "sub_agents": ["需要的子Agent名称列表，如果不需要则为空数组"]
}

用户输入: {{.input}}`

// NewIntentClassifier 创建意图分类器
func NewIntentClassifier(model model.BaseChatModel) *IntentClassifier {
	return &IntentClassifier{
		model:  model,
		prompt: intentPrompt,
	}
}

// Classify 执行意图分类
func (c *IntentClassifier) Classify(ctx context.Context, query string) (*IntentResult, error) {
	// 构建提示
	prompt := strings.Replace(c.prompt, "{{.input}}", query, 1)

	// 调用模型
	messages := []*schema.Message{
		{Role: schema.System, Content: "你是一个JSON解析器，只返回JSON格式的输出，不要包含任何其他文字。"},
		{Role: schema.User, Content: prompt},
	}

	resp, err := c.model.Generate(ctx, messages)
	if err != nil {
		// 降级：使用简单关键词匹配
		return c.fallbackClassify(query), nil
	}

	// 解析响应
	var result IntentResult
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		// 解析失败，使用降级方案
		return c.fallbackClassify(query), nil
	}

	// 验证意图有效性
	if result.Intent == "" {
		result.Intent = "fast"
		result.Confidence = 0.5
	}

	return &result, nil
}

// fallbackClassify 降级分类：基于关键词
func (c *IntentClassifier) fallbackClassify(query string) *IntentResult {
	query = strings.ToLower(query)

	// 多 Agent 协作关键词
	multiKeywords := []string{"多个", "协作", "团队", "并行", "同时", "分工", "合作"}
	for _, kw := range multiKeywords {
		if strings.Contains(query, kw) {
			return &IntentResult{
				Intent:     "multi",
				Confidence: 0.8,
				Reasoning:  "检测到多 Agent 协作关键词: " + kw,
				SubAgents:  []string{"researcher", "coder", "writer"},
			}
		}
	}

	// 深度分析关键词
	deepKeywords := []string{"规划", "计划", "分析", "详细", "深度", "复杂", "研究", "报告", "文档", "架构", "设计"}
	for _, kw := range deepKeywords {
		if strings.Contains(query, kw) {
			return &IntentResult{
				Intent:     "deep",
				Confidence: 0.8,
				Reasoning:  "检测到深度分析关键词: " + kw,
				SubAgents:  []string{},
			}
		}
	}

	// 简单问答
	return &IntentResult{
		Intent:     "fast",
		Confidence: 0.9,
		Reasoning:  "简单问答，直接响应",
		SubAgents:  []string{},
	}
}
