package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/common/einox"
	"zero-service/common/einox/a2ui"
	"zero-service/common/einox/memory"
)

type SoloRunner struct {
	agent  adk.Agent
	runner *adk.Runner
	config *RunnerConfig
	store  memory.Storage
	Logger logx.Logger
	mu     sync.RWMutex
}

type RunnerConfig struct {
	EnableStreaming bool
	EnableHistory   bool
	MaxHistory      int
	EnableInterrupt bool
	Timeout         int64 // 超时时间（秒）
}

type RunnerOption func(*RunnerConfig)

func WithEnableStreaming(enable bool) RunnerOption {
	return func(c *RunnerConfig) { c.EnableStreaming = enable }
}

func WithEnableHistory(enable bool) RunnerOption {
	return func(c *RunnerConfig) { c.EnableHistory = enable }
}

func WithMaxHistory(max int) RunnerOption {
	return func(c *RunnerConfig) { c.MaxHistory = max }
}

func WithEnableInterrupt(enable bool) RunnerOption {
	return func(c *RunnerConfig) { c.EnableInterrupt = enable }
}

func WithTimeout(timeout int64) RunnerOption {
	return func(c *RunnerConfig) { c.Timeout = timeout }
}

func NewSoloRunner(ctx context.Context, agent adk.Agent, opts ...RunnerOption) (*SoloRunner, error) {
	if agent == nil {
		return nil, fmt.Errorf("agent is required")
	}

	var cfg RunnerConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.MaxHistory <= 0 {
		cfg.MaxHistory = 20
	}

	runnerConfig := adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: cfg.EnableStreaming,
	}

	runner := adk.NewRunner(ctx, runnerConfig)

	return &SoloRunner{
		agent:  agent,
		runner: runner,
		config: &cfg,
		store:  memory.NewMemoryStorage(),
		Logger: logx.WithContext(ctx),
	}, nil
}

func (r *SoloRunner) Query(ctx context.Context, input string) (*adk.AsyncIterator[*adk.AgentEvent], error) {
	if input == "" {
		return nil, fmt.Errorf("input is required")
	}

	r.mu.RLock()
	runner := r.runner
	r.mu.RUnlock()

	if runner == nil {
		return nil, errors.New("runner not initialized")
	}

	iter := runner.Query(ctx, input)
	return iter, nil
}

func (r *SoloRunner) QueryStream(ctx context.Context, sessionID string, messages []*schema.Message) (*adk.AsyncIterator[*adk.AgentEvent], error) {
	if sessionID == "" {
		return nil, einox.ErrSessionIDRequired
	}

	r.mu.RLock()
	runner := r.runner
	r.mu.RUnlock()

	if runner == nil {
		return nil, errors.New("runner not initialized")
	}

	iter := runner.Run(ctx, messages)
	return iter, nil
}

func (r *SoloRunner) Chat(ctx context.Context, userID, sessionID, input string) (*ChatResult, error) {
	if userID == "" {
		return nil, einox.ErrUserIDRequired
	}
	if sessionID == "" {
		return nil, einox.ErrSessionIDRequired
	}
	if input == "" {
		return nil, fmt.Errorf("input is required")
	}

	var messages []*schema.Message

	if r.config.EnableHistory && r.store != nil {
		history, err := r.store.GetMessages(ctx, userID, sessionID, r.config.MaxHistory)
		if err != nil {
			r.Logger.Errorf("get history messages: %v", err)
		} else {
			for _, msg := range history {
				messages = append(messages, msg.ToSchemaMessage())
			}
		}
	}

	messages = append(messages, schema.UserMessage(input))

	r.mu.RLock()
	runner := r.runner
	r.mu.RUnlock()

	if runner == nil {
		return nil, errors.New("runner not initialized")
	}

	iter := runner.Run(ctx, messages)

	var result ChatResult
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return nil, fmt.Errorf("agent error: %w", event.Err)
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}
			result.Response = msg.Content
		}
	}

	if r.config.EnableHistory && r.store != nil {
		_ = r.store.SaveMessage(ctx, &memory.ConversationMessage{
			UserID:    userID,
			SessionID: sessionID,
			Role:      "user",
			Content:   input,
		})
		_ = r.store.SaveMessage(ctx, &memory.ConversationMessage{
			UserID:    userID,
			SessionID: sessionID,
			Role:      "assistant",
			Content:   result.Response,
		})
	}

	return &result, nil
}

func (r *SoloRunner) StreamToA2UI(ctx context.Context, w io.Writer, sessionID string, messages []*schema.Message) (string, string, error) {
	if w == nil {
		return "", "", fmt.Errorf("writer is required")
	}
	if sessionID == "" {
		return "", "", einox.ErrSessionIDRequired
	}

	r.mu.RLock()
	runner := r.runner
	r.mu.RUnlock()

	if runner == nil {
		return "", "", errors.New("runner not initialized")
	}

	iter := runner.Run(ctx, messages)

	response, interruptID, _, err := a2ui.StreamToWriter(w, sessionID, messages, iter)
	if err != nil {
		return response, "", fmt.Errorf("stream to a2ui: %w", err)
	}

	return response, interruptID, nil
}

func (r *SoloRunner) SetStore(store memory.Storage) {
	r.mu.Lock()
	r.store = store
	r.mu.Unlock()
}

func (r *SoloRunner) GetAgent() adk.Agent {
	r.mu.RLock()
	agent := r.agent
	r.mu.RUnlock()
	return agent
}

func (r *SoloRunner) GetRunner() *adk.Runner {
	r.mu.RLock()
	runner := r.runner
	r.mu.RUnlock()
	return runner
}

func (r *SoloRunner) GetConfig() *RunnerConfig {
	r.mu.RLock()
	config := r.config
	r.mu.RUnlock()
	return config
}

func (r *SoloRunner) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 这里可以添加清理逻辑
	r.runner = nil
	return nil
}

type ChatResult struct {
	Response  string
	Usage     *UsageInfo
	ToolCalls []schema.ToolCall
}

type UsageInfo struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type StreamResult struct {
	Content   string
	ToolCall  *schema.ToolCall
	Interrupt *InterruptInfo
	IsFinal   bool
}

type InterruptInfo struct {
	ID          string
	Type        string
	Description string
}
