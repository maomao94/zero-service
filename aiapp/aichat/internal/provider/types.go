package provider

// ChatRequest provider 层的统一请求，兼容 OpenAI Chat Completion 格式。
// 通过 ExtraBody 机制支持不同厂商的扩展参数，避免为每个厂商硬编码专属字段。
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	TopP        float64       `json:"top_p,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stop        []string      `json:"stop,omitempty"`
	User        string        `json:"user,omitempty"`
	Stream      bool          `json:"stream"`
	Tools       []ToolDef     `json:"tools,omitempty"`
	// ExtraBody 厂商特有扩展参数，序列化时会合并到 JSON 请求体顶层。
	// 使用 json:"-" 标签使其不参与标准 json.Marshal，由 marshalWithExtraBody() 单独处理合并。
	// 示例：
	//   dashscope（千问）: {"enable_thinking": true}
	//   openai/zhipu（智谱）: {"thinking": {"type": "enabled", "clear_thinking": true}}
	ExtraBody map[string]any `json:"-"`
}

type ChatMessage struct {
	Role             string     `json:"role"`
	Content          string     `json:"content"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallId       string     `json:"tool_call_id,omitempty"`
}

// ChatResponse 非流式完整响应
type ChatResponse struct {
	Id      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk 流式响应单个 chunk
type StreamChunk struct {
	Id      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []ChunkChoice `json:"choices"`
}

type ChunkChoice struct {
	Index        int       `json:"index"`
	Delta        ChatDelta `json:"delta"`
	FinishReason string    `json:"finish_reason"`
}

// ChatDelta 流式增量消息。thinking 模式下，模型先通过 ReasoningContent 逐 chunk 输出
// 推理过程，再通过 Content 输出最终回答。
type ChatDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
	// ReasoningContent 推理思考过程的增量文本。
	// 流式场景中先于 Content 输出，当 ReasoningContent 停止且 Content 开始时，
	// 表示模型已从思考阶段切换到回答阶段。
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

// ToolDef OpenAI tools 请求格式
type ToolDef struct {
	Type     string       `json:"type"` // "function"
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

// ToolCall LLM 响应中的工具调用
type ToolCall struct {
	Id       string           `json:"id"`
	Type     string           `json:"type"` // "function"
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}
