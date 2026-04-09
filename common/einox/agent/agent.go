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

// Ensure Agent implements einox.AgentInterface
var _ einox.AgentInterface = (*Agent)(nil)

// =============================================================================
// Agent
// =============================================================================

// Agent AI Agent
type Agent struct {
	name    string
	runner  *adk.Runner
	storage memory.Storage
	opts    options
}

// New 创建 Agent
func New(ctx context.Context, opts ...Option) (*Agent, error) {
	var cfg options
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.model == nil {
		return nil, fmt.Errorf("model is required")
	}

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

	// 3. 创建默认记忆存储
	if cfg.storage == nil {
		cfg.storage = memory.NewMemoryStorage()
	}

	return &Agent{
		name:    cfg.name,
		runner:  runner,
		storage: cfg.storage,
		opts:    cfg,
	}, nil
}

// Run 运行 Agent（单轮对话）
func (a *Agent) Run(ctx context.Context, input string, opts ...einox.RunOption) (*einox.AgentResult, error) {
	iter := a.runner.Query(ctx, input)
	return a.collectResult(iter)
}

// RunWithHistory 运行 Agent（带历史消息）
func (a *Agent) RunWithHistory(ctx context.Context, input string, opts ...einox.RunOption) (*einox.AgentResult, error) {
	var runCfg einox.RunOptions
	for _, opt := range opts {
		opt(&runCfg)
	}

	// 验证必填参数
	if runCfg.UserID == "" {
		return nil, einox.ErrUserIDRequired
	}
	if runCfg.SessionID == "" {
		return nil, einox.ErrSessionIDRequired
	}

	sessionID := runCfg.SessionID
	userID := runCfg.UserID

	// 1. 获取历史消息
	msgs, err := a.storage.GetMessages(ctx, userID, sessionID, 0)
	if err != nil {
		return nil, err
	}

	// 转换为 schema.Message
	var schemaMsgs []*schema.Message
	for _, msg := range msgs {
		schemaMsgs = append(schemaMsgs, msg.ToSchemaMessage())
	}

	// 2. 添加系统消息
	if runCfg.System != "" {
		schemaMsgs = append([]*schema.Message{
			{Role: schema.System, Content: runCfg.System},
		}, schemaMsgs...)
	}

	// 3. 添加用户消息
	userMsg := schema.UserMessage(input)
	schemaMsgs = append(schemaMsgs, userMsg)

	// 4. 运行
	iter := a.runner.Run(ctx, schemaMsgs)
	result, err := a.collectResult(iter)
	if err != nil {
		return nil, err
	}

	// 5. 运行成功后保存消息
	_ = a.saveMessage(ctx, userID, sessionID, "user", input)
	_ = a.saveMessage(ctx, userID, sessionID, "assistant", result.Response)

	return result, nil
}

// RunStream 流式运行
func (a *Agent) RunStream(ctx context.Context, input string, opts ...einox.RunOption) (<-chan *einox.AgentResult, error) {
	var runCfg einox.RunOptions
	for _, opt := range opts {
		opt(&runCfg)
	}

	// 验证必填参数
	if runCfg.UserID == "" {
		return nil, einox.ErrUserIDRequired
	}
	if runCfg.SessionID == "" {
		return nil, einox.ErrSessionIDRequired
	}

	sessionID := runCfg.SessionID
	userID := runCfg.UserID

	// 获取历史消息
	msgs, err := a.storage.GetMessages(ctx, userID, sessionID, 0)
	if err != nil {
		return nil, err
	}

	var schemaMsgs []*schema.Message
	for _, msg := range msgs {
		schemaMsgs = append(schemaMsgs, msg.ToSchemaMessage())
	}

	// 添加系统消息
	if runCfg.System != "" {
		schemaMsgs = append([]*schema.Message{
			{Role: schema.System, Content: runCfg.System},
		}, schemaMsgs...)
	}

	// 添加用户消息
	userMsg := schema.UserMessage(input)
	schemaMsgs = append(schemaMsgs, userMsg)

	ch := make(chan *einox.AgentResult, 1)

	go func() {
		defer close(ch)

		iter := a.runner.Run(ctx, schemaMsgs)
		var response string

		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				logx.Errorf("[Agent] stream error: %v", event.Err)
				ch <- &einox.AgentResult{Err: event.Err}
				return
			}
			if event.Output != nil {
				msg, err := event.Output.MessageOutput.GetMessage()
				if err != nil {
					continue
				}
				response = msg.Content
				ch <- &einox.AgentResult{Response: msg.Content}
			}
		}

		// 流式结束后保存消息
		if response != "" {
			_ = a.saveMessage(ctx, userID, sessionID, "user", input)
			_ = a.saveMessage(ctx, userID, sessionID, "assistant", response)
		}
	}()

	return ch, nil
}

// ClearMemory 清除记忆
func (a *Agent) ClearMemory(ctx context.Context, userID, sessionID string) error {
	if userID == "" {
		return einox.ErrUserIDRequired
	}
	if sessionID == "" {
		return einox.ErrSessionIDRequired
	}
	// 清空该会话的所有消息
	return a.storage.CleanupMessagesByLimit(ctx, userID, sessionID, 0)
}

// saveMessage 保存消息到存储
func (a *Agent) saveMessage(ctx context.Context, userID, sessionID, role, content string) error {
	return a.storage.SaveMessage(ctx, &memory.ConversationMessage{
		UserID:    userID,
		SessionID: sessionID,
		Role:      role,
		Content:   content,
	})
}

// collectResult 收集结果
func (a *Agent) collectResult(iter *adk.AsyncIterator[*adk.AgentEvent]) (*einox.AgentResult, error) {
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

// =============================================================================
// 内部创建函数
// =============================================================================

func createChatAgent(ctx context.Context, cfg *options) (*adk.ChatModelAgent, error) {
	description := cfg.description
	if description == "" {
		description = cfg.name + " - AI Assistant"
	}

	// 直接使用传入的模型
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

	// 绑定工具到 Agent（ToolsConfig 会自动将工具信息传递给 Model）
	if len(cfg.tools) > 0 {
		agentCfg.ToolsConfig = adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: cfg.tools,
			},
		}
	}

	// 应用 Handlers（ChatModelAgentMiddleware 接口实现）
	if len(cfg.handlers) > 0 {
		agentCfg.Handlers = cfg.handlers
	}

	// 应用 Middlewares（AgentMiddleware 结构体，简化场景）
	if len(cfg.middlewares) > 0 {
		agentCfg.Middlewares = cfg.middlewares
	}

	return adk.NewChatModelAgent(ctx, agentCfg)
}

// Stream 返回 Agent 事件的原始流
// 这个方法直接返回 adk.AsyncIterator，允许调用者自定义事件处理逻辑
// 与 RunStream 不同，Stream() 返回的是完整的 Agent 事件，而不是简化后的 Result
func (a *Agent) Stream(ctx context.Context, input string, opts ...einox.RunOption) (*adk.AsyncIterator[*adk.AgentEvent], error) {
	var runCfg einox.RunOptions
	for _, opt := range opts {
		opt(&runCfg)
	}

	// 验证必填参数
	if runCfg.UserID == "" {
		return nil, einox.ErrUserIDRequired
	}
	if runCfg.SessionID == "" {
		return nil, einox.ErrSessionIDRequired
	}

	sessionID := runCfg.SessionID
	userID := runCfg.UserID

	// 1. 获取历史消息
	msgs, err := a.storage.GetMessages(ctx, userID, sessionID, 0)
	if err != nil {
		return nil, err
	}

	// 转换为 schema.Message
	var schemaMsgs []*schema.Message
	for _, msg := range msgs {
		schemaMsgs = append(schemaMsgs, msg.ToSchemaMessage())
	}

	// 2. 添加系统消息
	if runCfg.System != "" {
		schemaMsgs = append([]*schema.Message{
			{Role: schema.System, Content: runCfg.System},
		}, schemaMsgs...)
	}

	// 3. 添加用户消息
	userMsg := schema.UserMessage(input)
	schemaMsgs = append(schemaMsgs, userMsg)

	// 4. 保存用户消息（异步）
	go func() {
		_ = a.saveMessage(ctx, userID, sessionID, "user", input)
	}()

	// 5. 返回原始事件流
	return a.runner.Run(ctx, schemaMsgs), nil
}

// CollectAndSaveStream 收集流式事件并保存结果
// 这是一个便捷方法，结合了 Stream() 和消息保存逻辑
func (a *Agent) CollectAndSaveStream(ctx context.Context, input string, opts ...einox.RunOption) (<-chan *einox.AgentResult, error) {
	iter, err := a.Stream(ctx, input, opts...)
	if err != nil {
		return nil, err
	}

	var runCfg einox.RunOptions
	for _, opt := range opts {
		opt(&runCfg)
	}
	userID := runCfg.UserID
	sessionID := runCfg.SessionID

	ch := make(chan *einox.AgentResult, 1)

	go func() {
		defer close(ch)
		var response string

		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				ch <- &einox.AgentResult{Err: event.Err}
				return
			}
			if event.Output != nil {
				msg, err := event.Output.MessageOutput.GetMessage()
				if err != nil {
					continue
				}
				response = msg.Content
				ch <- &einox.AgentResult{Response: msg.Content}
			}
		}

		// 流结束后保存助手消息
		if response != "" {
			_ = a.saveMessage(ctx, userID, sessionID, "assistant", response)
		}
	}()

	return ch, nil
}
