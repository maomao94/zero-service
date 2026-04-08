package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/common/einox/memory"
)

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
func (a *Agent) Run(ctx context.Context, input string, opts ...RunOption) (*Result, error) {
	iter := a.runner.Query(ctx, input)
	return a.collectResult(iter)
}

// RunWithHistory 运行 Agent（带历史消息）
func (a *Agent) RunWithHistory(ctx context.Context, input string, opts ...RunOption) (*Result, error) {
	var runCfg runOptions
	for _, opt := range opts {
		opt(&runCfg)
	}

	sessionID := runCfg.sessionID
	if sessionID == "" {
		sessionID = "default"
	}

	// 1. 获取历史消息
	msgs, err := a.storage.Get(ctx, sessionID, 0)
	if err != nil {
		return nil, err
	}

	// 2. 添加系统消息
	if runCfg.system != "" {
		msgs = append([]*schema.Message{
			{Role: schema.System, Content: runCfg.system},
		}, msgs...)
	}

	// 3. 添加用户消息
	userMsg := schema.UserMessage(input)
	msgs = append(msgs, userMsg)

	// 4. 运行（先不保存用户消息，确保 runner 成功后再保存）
	iter := a.runner.Run(ctx, msgs)
	result, err := a.collectResult(iter)
	if err != nil {
		return nil, err
	}

	// 5. 运行成功后保存消息
	_ = a.storage.Save(ctx, sessionID, userMsg)
	assistantMsg := &schema.Message{
		Role:    schema.Assistant,
		Content: result.Response,
	}
	_ = a.storage.Save(ctx, sessionID, assistantMsg)

	return result, nil
}

// RunStream 流式运行
func (a *Agent) RunStream(ctx context.Context, input string, opts ...RunOption) (<-chan *Result, error) {
	var runCfg runOptions
	for _, opt := range opts {
		opt(&runCfg)
	}

	sessionID := runCfg.sessionID
	if sessionID == "" {
		sessionID = "default"
	}

	// 获取历史消息
	msgs, err := a.storage.Get(ctx, sessionID, 0)
	if err != nil {
		return nil, err
	}

	// 添加系统消息
	if runCfg.system != "" {
		msgs = append([]*schema.Message{
			{Role: schema.System, Content: runCfg.system},
		}, msgs...)
	}

	// 添加用户消息
	userMsg := schema.UserMessage(input)
	msgs = append(msgs, userMsg)

	ch := make(chan *Result, 1)

	go func() {
		defer close(ch)

		iter := a.runner.Run(ctx, msgs)
		var response string

		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				logx.Errorf("[Agent] stream error: %v", event.Err)
				ch <- &Result{Err: event.Err}
				return
			}
			if event.Output != nil {
				msg, err := event.Output.MessageOutput.GetMessage()
				if err != nil {
					continue
				}
				response = msg.Content
				ch <- &Result{Response: msg.Content}
			}
		}

		// 流式结束后保存消息
		if response != "" {
			_ = a.storage.Save(ctx, sessionID, userMsg)
			assistantMsg := &schema.Message{
				Role:    schema.Assistant,
				Content: response,
			}
			_ = a.storage.Save(ctx, sessionID, assistantMsg)
		}
	}()

	return ch, nil
}

// ClearMemory 清除记忆
func (a *Agent) ClearMemory(ctx context.Context, sessionID string) error {
	return a.storage.Clear(ctx, sessionID)
}

// collectResult 收集结果
func (a *Agent) collectResult(iter *adk.AsyncIterator[*adk.AgentEvent]) (*Result, error) {
	var result Result
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

	// 获取 ChatModel（可能是 BaseChatModel, ChatModel 或 ToolCallingChatModel）
	var chatModel model.BaseChatModel

	// 尝试转换为 ToolCallingChatModel（支持安全的 WithTools）
	if len(cfg.tools) > 0 {
		if tcModel, ok := cfg.model.(model.ToolCallingChatModel); ok {
			// 使用 WithTools 创建带有工具的模型实例（并发安全）
			toolInfos := make([]*schema.ToolInfo, 0, len(cfg.tools))
			for _, t := range cfg.tools {
				info, err := t.Info(ctx)
				if err != nil {
					return nil, fmt.Errorf("get tool info: %w", err)
				}
				toolInfos = append(toolInfos, info)
			}

			var err error
			chatModel, err = tcModel.WithTools(toolInfos)
			if err != nil {
				return nil, fmt.Errorf("bind tools to model: %w", err)
			}
		} else if cModel, ok := cfg.model.(model.ChatModel); ok {
			// 降级到 ChatModel（deprecated，但仍然支持）
			chatModel = cModel
		} else if bModel, ok := cfg.model.(model.BaseChatModel); ok {
			// 直接使用 BaseChatModel
			chatModel = bModel
		} else {
			return nil, fmt.Errorf("model must implement BaseChatModel interface")
		}
	} else {
		// 没有工具，直接使用传入的模型
		if cModel, ok := cfg.model.(model.ChatModel); ok {
			chatModel = cModel
		} else if bModel, ok := cfg.model.(model.BaseChatModel); ok {
			chatModel = bModel
		} else {
			return nil, fmt.Errorf("model must implement BaseChatModel interface")
		}
	}

	agentCfg := &adk.ChatModelAgentConfig{
		Name:        cfg.name,
		Description: description,
		Instruction: cfg.instruction,
		Model:       chatModel,
	}

	return adk.NewChatModelAgent(ctx, agentCfg)
}

// =============================================================================
// 类型定义
// =============================================================================

// Result Agent 执行结果
type Result struct {
	Response string // 响应内容
	Err      error  // 错误信息（仅在发生错误时设置）
}
