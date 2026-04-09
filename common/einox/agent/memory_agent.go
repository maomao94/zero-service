package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/common/einox"
	"zero-service/common/einox/memory"
)

// =============================================================================
// MemoryAgent - 带记忆功能的 Agent
// =============================================================================

// MemoryAgent 带记忆功能的 Agent
type MemoryAgent struct {
	name    string
	runner  *adk.Runner
	manager *memory.MemoryManager
	opts    options
	userID  string
}

// NewWithMemory 创建带记忆功能的 Agent
//
// ctx: 上下文
// cm: ChatModel，用于生成摘要和记忆分析
// opts: Agent 配置选项
// 注意：调用者需要设置 WithMemoryConfig 选项来配置记忆功能
func NewWithMemory(ctx context.Context, cm model.BaseChatModel, opts ...Option) (*MemoryAgent, error) {
	var cfg options
	for _, opt := range opts {
		opt(&cfg)
	}

	if cm == nil {
		return nil, fmt.Errorf("model is required")
	}

	// 设置 model 到 cfg
	cfg.model = cm

	// 1. 创建 Agent
	chatAgent, err := createChatAgent(ctx, &cfg)
	if err != nil {
		return nil, fmt.Errorf("create chat agent: %w", err)
	}

	// 2. 创建 Runner
	runnerConfig := adk.RunnerConfig{
		Agent:           chatAgent,
		EnableStreaming: cfg.stream,
	}
	runner := adk.NewRunner(ctx, runnerConfig)

	// 3. 创建默认存储
	var store memory.Storage
	if cfg.storage != nil {
		if s, ok := cfg.storage.(memory.Storage); ok {
			store = s
		} else {
			store = memory.NewMemoryStorage()
		}
	} else {
		store = memory.NewMemoryStorage()
	}

	// 4. 创建记忆管理器
	manager, err := memory.NewMemoryManager(cm, store, cfg.memoryConfig)
	if err != nil {
		return nil, fmt.Errorf("create memory manager: %w", err)
	}

	return &MemoryAgent{
		name:    cfg.name,
		runner:  runner,
		manager: manager,
		opts:    cfg,
	}, nil
}

// Run 运行 Agent（单轮对话）
func (a *MemoryAgent) Run(ctx context.Context, userID, sessionID, input string) (*MemoryResult, error) {
	// 1. 保存用户消息
	if err := a.manager.ProcessUserMessage(ctx, userID, sessionID, input, nil); err != nil {
		logx.Errorf("[MemoryAgent] process user message: %v", err)
	}

	// 2. 获取上下文（用户记忆 + 会话摘要）
	systemContext := a.buildSystemContext(ctx, userID, sessionID)

	// 3. 构建消息列表
	messages := []*schema.Message{
		{Role: schema.System, Content: systemContext},
	}

	// 添加历史消息（用于上下文）
	history, err := a.manager.GetRecentMessages(ctx, userID, sessionID, a.opts.memoryConfig.MemoryLimit)
	if err != nil {
		logx.Errorf("[MemoryAgent] get recent messages: %v", err)
	} else {
		for _, msg := range history {
			messages = append(messages, msg.ToSchemaMessage())
		}
	}

	// 添加当前用户消息
	messages = append(messages, schema.UserMessage(input))

	// 4. 运行 Agent
	iter := a.runner.Run(ctx, messages)
	result, err := a.collectResult(iter)
	if err != nil {
		return nil, err
	}

	// 5. 处理助手消息（异步保存，生成摘要/记忆）
	if err := a.manager.ProcessAssistantMessage(ctx, userID, sessionID, result.Response); err != nil {
		logx.Errorf("[MemoryAgent] process assistant message: %v", err)
	}

	return &MemoryResult{
		Response: result.Response,
	}, nil
}

// RunStream 流式运行
func (a *MemoryAgent) RunStream(ctx context.Context, userID, sessionID, input string) (<-chan *MemoryResult, error) {
	// 1. 保存用户消息
	if err := a.manager.ProcessUserMessage(ctx, userID, sessionID, input, nil); err != nil {
		logx.Errorf("[MemoryAgent] process user message: %v", err)
	}

	// 2. 获取上下文
	systemContext := a.buildSystemContext(ctx, userID, sessionID)

	// 3. 构建消息列表
	messages := []*schema.Message{
		{Role: schema.System, Content: systemContext},
	}

	history, err := a.manager.GetRecentMessages(ctx, userID, sessionID, a.opts.memoryConfig.MemoryLimit)
	if err != nil {
		logx.Errorf("[MemoryAgent] get recent messages: %v", err)
	} else {
		for _, msg := range history {
			messages = append(messages, msg.ToSchemaMessage())
		}
	}
	messages = append(messages, schema.UserMessage(input))

	ch := make(chan *MemoryResult, 1)

	go func() {
		defer close(ch)

		iter := a.runner.Run(ctx, messages)
		var response string

		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				logx.Errorf("[MemoryAgent] stream error: %v", event.Err)
				ch <- &MemoryResult{Err: event.Err}
				return
			}
			if event.Output != nil {
				msg, err := event.Output.MessageOutput.GetMessage()
				if err != nil {
					continue
				}
				response = msg.Content
				ch <- &MemoryResult{Response: msg.Content}
			}
		}

		// 流结束后处理助手消息
		if response != "" {
			if err := a.manager.ProcessAssistantMessage(ctx, userID, sessionID, response); err != nil {
				logx.Errorf("[MemoryAgent] process assistant message: %v", err)
			}
		}
	}()

	return ch, nil
}

// buildSystemContext 构建系统上下文（包含用户记忆和会话摘要）
func (a *MemoryAgent) buildSystemContext(ctx context.Context, userID, sessionID string) string {
	var contextParts []string

	// 添加原始系统指令
	if a.opts.instruction != "" {
		contextParts = append(contextParts, a.opts.instruction)
	}

	// 添加用户记忆
	if a.opts.memoryConfig != nil && a.opts.memoryConfig.EnableUserMemories {
		mem, err := a.manager.GetUserMemory(ctx, userID)
		if err != nil {
			logx.Errorf("[MemoryAgent] get user memory: %v", err)
		} else if mem != nil && mem.Memory != "" {
			contextParts = append(contextParts, "", "## 用户记忆", mem.Memory)
		}
	}

	// 添加会话摘要
	if a.opts.memoryConfig != nil && a.opts.memoryConfig.EnableSessionSummary {
		summary, err := a.manager.GetSessionSummary(ctx, userID, sessionID)
		if err != nil {
			logx.Errorf("[MemoryAgent] get session summary: %v", err)
		} else if summary != nil && summary.Summary != "" {
			contextParts = append(contextParts, "", "## 当前会话摘要", summary.Summary)
		}
	}

	return joinContext(contextParts...)
}

// Stream 返回 Agent 事件的原始流
func (a *MemoryAgent) Stream(ctx context.Context, userID, sessionID, input string) (*adk.AsyncIterator[*adk.AgentEvent], error) {
	// 1. 保存用户消息
	if err := a.manager.ProcessUserMessage(ctx, userID, sessionID, input, nil); err != nil {
		logx.Errorf("[MemoryAgent] process user message: %v", err)
	}

	// 2. 获取上下文
	systemContext := a.buildSystemContext(ctx, userID, sessionID)

	// 3. 构建消息列表
	messages := []*schema.Message{
		{Role: schema.System, Content: systemContext},
	}

	history, err := a.manager.GetRecentMessages(ctx, userID, sessionID, a.opts.memoryConfig.MemoryLimit)
	if err != nil {
		logx.Errorf("[MemoryAgent] get recent messages: %v", err)
	} else {
		for _, msg := range history {
			messages = append(messages, msg.ToSchemaMessage())
		}
	}
	messages = append(messages, schema.UserMessage(input))

	// 4. 返回原始事件流
	return a.runner.Run(ctx, messages), nil
}

// CollectAndSaveStream 收集流式事件并保存结果
func (a *MemoryAgent) CollectAndSaveStream(ctx context.Context, userID, sessionID, input string) (<-chan *MemoryResult, error) {
	iter, err := a.Stream(ctx, userID, sessionID, input)
	if err != nil {
		return nil, err
	}

	ch := make(chan *MemoryResult, 1)

	go func() {
		defer close(ch)
		var response string

		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				ch <- &MemoryResult{Err: event.Err}
				return
			}
			if event.Output != nil {
				msg, err := event.Output.MessageOutput.GetMessage()
				if err != nil {
					continue
				}
				response = msg.Content
				ch <- &MemoryResult{Response: msg.Content}
			}
		}

		// 流结束后处理助手消息
		if response != "" {
			if err := a.manager.ProcessAssistantMessage(ctx, userID, sessionID, response); err != nil {
				logx.Errorf("[MemoryAgent] process assistant message: %v", err)
			}
		}
	}()

	return ch, nil
}

// collectResult 收集结果
func (a *MemoryAgent) collectResult(iter *adk.AsyncIterator[*adk.AgentEvent]) (*einox.AgentResult, error) {
	var result einox.AgentResult
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return nil, event.Err
		}
		if event.Output != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}
			result.Response = msg.Content
		}
	}
	return &result, nil
}

// GetMemoryManager 获取记忆管理器
func (a *MemoryAgent) GetMemoryManager() *memory.MemoryManager {
	return a.manager
}

// Close 关闭 Agent
func (a *MemoryAgent) Close() {
	if a.manager != nil {
		a.manager.Close()
	}
}

// =============================================================================
// MemoryResult - 带记忆的 Agent 结果
// =============================================================================

// MemoryResult 带记忆功能的 Agent 执行结果
type MemoryResult struct {
	Response string
	Err      error
}

// =============================================================================
// 辅助函数
// =============================================================================

func joinContext(parts ...string) string {
	result := ""
	for _, part := range parts {
		if part != "" {
			result += part + "\n"
		}
	}
	return result
}

// =============================================================================
// 内部创建函数（复用 createChatAgent 逻辑）
// =============================================================================

func createChatAgentWithMemory(ctx context.Context, cfg *options) (*adk.ChatModelAgent, error) {
	description := cfg.description
	if description == "" {
		description = cfg.name + " - AI Assistant"
	}

	var chatModel model.BaseChatModel
	if cModel, ok := cfg.model.(model.ChatModel); ok {
		chatModel = cModel
	} else if bModel, ok := cfg.model.(model.BaseChatModel); ok {
		chatModel = bModel
	} else {
		return nil, fmt.Errorf("model must implement BaseChatModel interface")
	}

	agentCfg := &adk.ChatModelAgentConfig{
		Name:        cfg.name,
		Description: description,
		Instruction: cfg.instruction,
		Model:       chatModel,
	}

	// 绑定工具到 Agent
	if len(cfg.tools) > 0 {
		agentCfg.ToolsConfig = adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: cfg.tools,
			},
		}
	}

	return adk.NewChatModelAgent(ctx, agentCfg)
}
