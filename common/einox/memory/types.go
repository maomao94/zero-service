package memory

import (
	"encoding/json"
	"time"

	"github.com/cloudwego/eino/schema"
)

// =============================================================================
// 聊天记忆系统（与模型无关，所有聊天都需要）
// =============================================================================

// UserMemory 用户记忆结构
//
// 每个用户一条记录，使用 Markdown 格式存储所有记忆内容。
// 记忆由 AI 自动分析提取，存储用户的偏好、习惯、重要信息等。
type UserMemory struct {
	ID         string    `json:"id"`         // 记忆ID
	TenantID   string    `json:"tenantId"`   // 租户ID（用于权限隔离）
	UserID     string    `json:"userId"`     // 用户ID（主键）
	Memory     string    `json:"memory"`     // 记忆内容（Markdown 格式）
	Vector     []float32 `json:"-"`          // 记忆向量（用于语义检索，不序列化）
	Permission string    `json:"permission"` // 权限级别：private(私有)、shared(共享)、public(公开)
	CreatedAt  time.Time `json:"createdAt"`  // 创建时间
	UpdatedAt  time.Time `json:"updatedAt"`  // 最后更新时间
}

// MemorySearchResult 记忆搜索结果
type MemorySearchResult struct {
	Memory   *UserMemory `json:"memory"`   // 匹配的记忆
	Score    float64     `json:"score"`    // 匹配得分（0-1，越高越匹配）
	Distance float64     `json:"distance"` // 向量距离（越小越匹配）
}

// SemanticRetrievalConfig 语义检索配置
type SemanticRetrievalConfig struct {
	Enabled      bool    `json:"enabled"`      // 是否启用语义检索
	TopK         int     `json:"topK"`         // 返回最匹配的TopK条结果
	Threshold    float64 `json:"threshold"`    // 匹配阈值，低于此分数的结果不返回
	VectorDim    int     `json:"vectorDim"`    // 向量维度
	DistanceType string  `json:"distanceType"` // 距离类型：cosine(余弦)、euclidean(欧氏)
}

// SessionSummary 会话摘要结构
//
// 存储对话会话的智能摘要，支持增量更新。
type SessionSummary struct {
	ID        string    `json:"id"`        // 摘要ID
	SessionID string    `json:"sessionId"` // 会话ID
	UserID    string    `json:"userId"`    // 用户ID
	Summary   string    `json:"summary"`   // 摘要内容
	CreatedAt time.Time `json:"createdAt"` // 创建时间
	UpdatedAt time.Time `json:"updatedAt"` // 最后更新时间
}

// ConversationMessage 对话消息结构
//
// 存储完整的对话历史，与模型无关。
// 所有聊天都需要记忆，用于多轮对话的上下文保持。
type ConversationMessage struct {
	ID        string                    `json:"id"`              // 消息ID
	SessionID string                    `json:"sessionId"`       // 会话ID
	UserID    string                    `json:"userId"`          // 用户ID
	Role      string                    `json:"role"`            // 角色 (user/assistant/system/tool)
	Content   string                    `json:"content"`         // 消息内容
	Parts     []schema.MessageInputPart `json:"parts,omitempty"` // 多部分内容
	// 工具调用相关
	ToolCalls  []schema.ToolCall `json:"toolCalls,omitempty"`  // 工具调用列表
	ToolCallID string            `json:"toolCallId,omitempty"` // 工具调用ID（tool 角色）
	ToolName   string            `json:"toolName,omitempty"`   // 工具名（tool 角色）
	// 思考过程
	ReasoningContent string    `json:"reasoningContent,omitempty"` // 深度思考内容
	CreatedAt        time.Time `json:"createdAt"`                  // 创建时间
}

// ToSchemaMessage 将 ConversationMessage 转换为 schema.Message
func (m *ConversationMessage) ToSchemaMessage() *schema.Message {
	msg := &schema.Message{
		Role:             schema.RoleType(m.Role),
		Content:          m.Content,
		ReasoningContent: m.ReasoningContent,
		ToolCalls:        m.ToolCalls,
		ToolCallID:       m.ToolCallID,
		ToolName:         m.ToolName,
	}
	if len(m.Parts) > 0 {
		msg.UserInputMultiContent = m.Parts
	}
	return msg
}

// FromSchemaMessage 从 schema.Message 创建 ConversationMessage
func FromSchemaMessage(sessionID, userID string, msg *schema.Message) *ConversationMessage {
	return &ConversationMessage{
		SessionID:        sessionID,
		UserID:           userID,
		Role:             string(msg.Role),
		Content:          msg.Content,
		Parts:            msg.UserInputMultiContent,
		ToolCalls:        msg.ToolCalls,
		ToolCallID:       msg.ToolCallID,
		ToolName:         msg.ToolName,
		ReasoningContent: msg.ReasoningContent,
		CreatedAt:        time.Now(),
	}
}

// =============================================================================
// 中断/恢复记忆系统（用于 Human-in-the-Loop）
// =============================================================================

// CheckpointState 检查点状态
//
// 用于 Human-in-the-Loop 场景，保存 Agent 执行状态，
// 支持中断后恢复执行。
type CheckpointState struct {
	ID        string `json:"id"`        // 检查点ID
	SessionID string `json:"sessionId"` // 会话ID
	UserID    string `json:"userId"`    // 用户ID
	// 序列化的状态数据
	StateData []byte `json:"stateData"` // Agent 内部状态（JSON 序列化）
	// 中断信息
	InterruptInfo *InterruptInfo `json:"interruptInfo,omitempty"` // 中断信息
	// 元数据
	CreatedAt time.Time `json:"createdAt"` // 创建时间
	UpdatedAt time.Time `json:"updatedAt"` // 更新时间
}

// InterruptInfo 中断信息
//
// 当 Agent 需要人工介入时，保存中断信息。
type InterruptInfo struct {
	// 中断ID
	ID string `json:"id"`
	// 中断类型
	Type InterruptType `json:"type"`
	// 中断原因
	Reason string `json:"reason"`
	// 中断上下文（需要用户提供的信息）
	Context map[string]any `json:"context,omitempty"`
	// 待审批的工具调用
	ToolCall *schema.ToolCall `json:"toolCall,omitempty"`
	// 待审批的工具参数
	ToolArguments string `json:"toolArguments,omitempty"`
	// 用户问题（需要用户回答）
	Question string `json:"question,omitempty"`
	// 选项（如果需要用户选择）
	Options []string `json:"options,omitempty"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
}

// InterruptType 中断类型
type InterruptType string

const (
	// InterruptTypeApproval 工具调用审批中断
	InterruptTypeApproval InterruptType = "approval"
	// InterruptTypeClarification 需要用户澄清
	InterruptTypeClarification InterruptType = "clarification"
	// InterruptTypeFeedback 需要用户反馈
	InterruptTypeFeedback InterruptType = "feedback"
	// InterruptTypeChoice 需要用户选择
	InterruptTypeChoice InterruptType = "choice"
	// InterruptTypeCustom 自定义中断
	InterruptTypeCustom InterruptType = "custom"
)

// ResumeInfo 恢复信息
//
// 用户响应中断后，提供恢复执行所需的信息。
type ResumeInfo struct {
	// 中断ID
	InterruptID string `json:"interruptId"`
	// 是否批准（用于审批类型中断）
	Approved bool `json:"approved,omitempty"`
	// 用户输入（用于澄清/反馈类型中断）
	UserInput string `json:"userInput,omitempty"`
	// 用户选择（用于选择类型中断）
	SelectedOption string `json:"selectedOption,omitempty"`
	// 自定义数据
	CustomData map[string]any `json:"customData,omitempty"`
	// 时间戳
	Timestamp time.Time `json:"timestamp"`
}

// ToJSON 序列化为 JSON
func (r *ResumeInfo) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// FromJSON 从 JSON 反序列化
func ResumeInfoFromJSON(data []byte) (*ResumeInfo, error) {
	var info ResumeInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// =============================================================================
// 记忆检索配置
// =============================================================================

// MemoryRetrieval 记忆检索方式
type MemoryRetrieval string

const (
	RetrievalLastN    MemoryRetrieval = "last_n"   // 检索最近的N条记忆
	RetrievalFirstN   MemoryRetrieval = "first_n"  // 检索最早的N条记忆
	RetrievalSemantic MemoryRetrieval = "semantic" // 语义检索（基于相似性）
)

// =============================================================================
// 配置定义
// =============================================================================

// MemoryConfig 记忆配置
type MemoryConfig struct {
	// ================================
	// 聊天记忆配置（与模型无关）
	// ================================
	// 是否启用对话历史存储（默认启用，所有聊天都需要）
	EnableConversationHistory bool `json:"enableConversationHistory"`
	// 对话历史数量限制
	ConversationHistoryLimit int `json:"conversationHistoryLimit"`
	// 是否启用用户记忆
	EnableUserMemories bool `json:"enableUserMemories"`
	// 是否启用会话摘要
	EnableSessionSummary bool `json:"enableSessionSummary"`
	// 用户记忆检索方式（EnableUserMemories 开启时生效）
	Retrieval MemoryRetrieval `json:"retrieval"`
	// 记忆数量限制
	MemoryLimit int `json:"memoryLimit"`
	// 异步处理的 goroutine 池大小
	AsyncWorkerPoolSize int `json:"asyncWorkerPoolSize"`
	// 语义检索配置
	SemanticRetrieval SemanticRetrievalConfig `json:"semanticRetrieval"`
	// 默认权限级别
	DefaultPermission string `json:"defaultPermission"` // 默认 private

	// ================================
	// 中断/恢复记忆配置
	// ================================
	// 是否启用检查点（用于 Human-in-the-Loop）
	EnableCheckpoint bool `json:"enableCheckpoint"`
	// 检查点保留时间（小时）
	CheckpointRetentionTime int `json:"checkpointRetentionTime"`

	// 摘要触发配置
	SummaryTrigger SummaryTriggerConfig `json:"summaryTrigger"`

	// 清理配置
	Cleanup CleanupConfig `json:"cleanup"`
}

// DefaultMemoryConfig 返回默认配置
func DefaultMemoryConfig() *MemoryConfig {
	return &MemoryConfig{
		EnableConversationHistory: true,
		ConversationHistoryLimit:  100,
		EnableUserMemories:        true,
		EnableSessionSummary:      false,
		Retrieval:                 RetrievalLastN,
		MemoryLimit:               20,
		AsyncWorkerPoolSize:       5,
		DefaultPermission:         "private",
		SemanticRetrieval: SemanticRetrievalConfig{
			Enabled:      false,
			TopK:         5,
			Threshold:    0.7,
			VectorDim:    1536, // OpenAI ada-002 维度
			DistanceType: "cosine",
		},
		EnableCheckpoint:        true,
		CheckpointRetentionTime: 24, // 24小时
		SummaryTrigger: SummaryTriggerConfig{
			Strategy:         TriggerSmart,
			MessageThreshold: 10,
			MinInterval:      600, // 600秒最小间隔
		},
		Cleanup: CleanupConfig{
			SessionCleanupInterval: 24,   // 24小时
			SessionRetentionTime:   168,  // 7天
			MessageHistoryLimit:    1000, // 1000条
			CleanupInterval:        12,   // 12小时
		},
	}
}

// DisableMemory 返回禁用所有记忆功能的配置
func DisableMemory() *MemoryConfig {
	return &MemoryConfig{
		EnableConversationHistory: false,
		EnableUserMemories:        false,
		EnableSessionSummary:      false,
		EnableCheckpoint:          false,
		MemoryLimit:               0,
		AsyncWorkerPoolSize:       0,
	}
}

// EnableAllMemory 返回启用所有记忆功能的配置
func EnableAllMemory() *MemoryConfig {
	return &MemoryConfig{
		EnableConversationHistory: true,
		ConversationHistoryLimit:  100,
		EnableUserMemories:        true,
		EnableSessionSummary:      true,
		Retrieval:                 RetrievalLastN,
		MemoryLimit:               20,
		AsyncWorkerPoolSize:       5,
		EnableCheckpoint:          true,
		CheckpointRetentionTime:   24,
		SummaryTrigger: SummaryTriggerConfig{
			Strategy:         TriggerSmart,
			MessageThreshold: 10,
			MinInterval:      600,
		},
		Cleanup: CleanupConfig{
			SessionCleanupInterval: 24,
			SessionRetentionTime:   168,
			MessageHistoryLimit:    1000,
			CleanupInterval:        12,
		},
	}
}

// CleanupConfig 清理相关配置
type CleanupConfig struct {
	// 会话状态清理间隔（小时），默认24小时
	SessionCleanupInterval int `json:"sessionCleanupInterval"`
	// 会话状态保留时间（小时），默认168小时（7天）
	SessionRetentionTime int `json:"sessionRetentionTime"`
	// 消息历史保留数量限制，默认1000条
	MessageHistoryLimit int `json:"messageHistoryLimit"`
	// 定期清理间隔（小时），默认12小时
	CleanupInterval int `json:"cleanupInterval"`
}

// SummaryTriggerConfig 摘要触发配置
type SummaryTriggerConfig struct {
	// 触发策略类型
	Strategy SummaryTriggerStrategy `json:"strategy"`
	// 基于消息数量触发的阈值
	MessageThreshold int `json:"messageThreshold"`
	// 最小触发间隔（秒）
	MinInterval int `json:"minInterval"`
}

// SummaryTriggerStrategy 摘要触发策略
type SummaryTriggerStrategy string

const (
	TriggerAlways     SummaryTriggerStrategy = "always"      // 每次都触发
	TriggerByMessages SummaryTriggerStrategy = "by_messages" // 基于消息数量触发
	TriggerByTime     SummaryTriggerStrategy = "by_time"     // 基于时间间隔触发
	TriggerSmart      SummaryTriggerStrategy = "smart"       // 智能触发
)

// UserMemoryAnalyzerParam 用户记忆更新参数
type UserMemoryAnalyzerParam struct {
	Op     string `json:"op"`     // 操作类型: update(更新记忆)、noop(无需更新)
	Memory string `json:"memory"` // 记忆内容（完整 Markdown 文档，op 为 update 时有效）
}

// 用户记忆操作类型
type UserMemoryOp string

const (
	UserMemoryOpUpdate UserMemoryOp = "update" // 更新记忆
	UserMemoryOpNoop   UserMemoryOp = "noop"   // 无需更新
)

// TaskQueueStats 异步任务队列统计
type TaskQueueStats struct {
	QueueSize        int     `json:"queueSize"`        // 队列大小
	QueueCapacity    int     `json:"queueCapacity"`    // 队列容量
	ProcessedTasks   int64   `json:"processedTasks"`   // 已处理任务数
	DroppedTasks     int64   `json:"droppedTasks"`     // 丢弃任务数
	ActiveWorkers    int     `json:"activeWorkers"`    // 当前工作 goroutine 数
	QueueUtilization float64 `json:"queueUtilization"` // 队列使用率
}

// =============================================================================
// Prompt 模板
// =============================================================================

// DefaultUserMemoryPrompt 用户记忆提取和更新 Prompt
const DefaultUserMemoryPrompt = `你是一个用户记忆管理助手。你的任务是根据对话内容，提取或更新用户的记忆信息。

## 当前用户记忆
如果用户没有记忆，请创建一个新的记忆文档。
%s

## 新的对话内容
%s

## 输出要求
请分析对话内容，决定是更新记忆还是保持不变：

1. 如果需要更新记忆，请以 Markdown 格式输出完整的新记忆文档，格式如下：
---
[title: 用户记忆]

## 基本信息
- 用户名/昵称：xxx
- 偏好：xxx
- 习惯：xxx
- ...

## 重要信息
- xxx
- ...

## 其他
- xxx
---

2. 如果当前记忆已经完整且无需更新，请直接输出 "NO_UPDATE"

请开始分析：`

// DefaultSessionSummaryPrompt 会话摘要生成 Prompt
const DefaultSessionSummaryPrompt = `你是一个会话摘要生成助手。你的任务是对话历史生成简洁准确的摘要。

## 对话历史
%s

## 输出要求
请生成一个简洁的摘要，包含：
1. 对话的主要话题和目的
2. 关键信息和结论
3. 未完成的事项或下一步

请以 Markdown 格式输出：`

// DefaultIncrementalSessionSummaryPrompt 增量摘要更新 Prompt
const DefaultIncrementalSessionSummaryPrompt = `你是一个会话摘要更新助手。你的任务是基于新的对话内容，更新已有的会话摘要。

## 已有摘要
%s

## 新的对话内容
%s

## 输出要求
请基于新的对话内容，更新摘要：

1. 如果有新的关键信息需要添加到摘要中，请输出完整的新摘要：
---
## 会话摘要

[更新时间：%s]

### 话题和目的
xxx

### 关键信息
- 已有信息...
- 新增信息...

### 未完成事项
xxx
---

2. 如果对话内容没有新增关键信息，请直接输出 "NO_UPDATE"

请开始更新：`
