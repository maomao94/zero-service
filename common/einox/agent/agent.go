package agent

import (
	"context"
	"fmt"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"time"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/skill"
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

// AgentStatus 智能体状态
type AgentStatus string

const (
	AgentStatusInit     AgentStatus = "init"     // 初始化中
	AgentStatusRunning  AgentStatus = "running"  // 运行中
	AgentStatusPaused   AgentStatus = "paused"   // 已暂停
	AgentStatusStopping AgentStatus = "stopping" // 关闭中
	AgentStatusStopped  AgentStatus = "stopped"  // 已关闭
	AgentStatusError    AgentStatus = "error"    // 错误状态
)

// Agent AI Agent
type Agent struct {
	name           string
	adkAgent       adk.Agent // 底层的 adk.Agent 实例
	runner         *adk.Runner
	storage        memory.Storage
	opts           options
	status         AgentStatus
	statusMu       sync.RWMutex
	startTime      time.Time
	lastActiveTime time.Time
	requestCount   atomic.Int64
	errorCount     atomic.Int64
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

	agent := &Agent{
		name:      cfg.name,
		adkAgent:  chatAgent,
		runner:    runner,
		storage:   cfg.storage,
		opts:      cfg,
		status:    AgentStatusInit,
		startTime: time.Now(),
	}
	agent.lastActiveTime = agent.startTime
	return agent, nil
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

	// 配置 skill 中间件（通用功能，所有 Agent 类型支持）
	skillHandlers := buildSkillHandlers(ctx, cfg)
	cfg.handlers = append(skillHandlers, cfg.handlers...)

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

// buildSkillHandlers 根据配置构建 skill 中间件
func buildSkillHandlers(ctx context.Context, cfg *options) []adk.ChatModelAgentMiddleware {
	if cfg.skillsDir == "" {
		// 尝试从环境变量获取
		cfg.skillsDir = os.Getenv("EINO_EXT_SKILLS_DIR")
	}

	if cfg.skillsDir == "" {
		return nil
	}

	// 验证目录存在
	fi, err := os.Stat(cfg.skillsDir)
	if err != nil || !fi.IsDir() {
		return nil
	}

	// 使用 local backend
	backend, err := localbk.NewBackend(ctx, &localbk.Config{})
	if err != nil {
		logx.Errorf("[Agent] create local backend for skill: %v", err)
		return nil
	}

	// 创建 skill backend
	skillBackend, err := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{
		Backend: backend,
		BaseDir: cfg.skillsDir,
	})
	if err != nil {
		logx.Errorf("[Agent] create skill backend: %v", err)
		return nil
	}

	// 创建 skill 中间件
	skillMiddleware, err := skill.NewMiddleware(ctx, &skill.Config{
		Backend: skillBackend,
	})
	if err != nil {
		logx.Errorf("[Agent] create skill middleware: %v", err)
		return nil
	}

	logx.Infof("[Agent] skill middleware loaded: dir=%s", cfg.skillsDir)
	return []adk.ChatModelAgentMiddleware{skillMiddleware}
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

// Runner 获取底层的 adk.Runner
func (a *Agent) Runner() *adk.Runner {
	return a.runner
}

// GetAdkAgent 获取底层 ADK Agent
func (a *Agent) GetAgent() adk.Agent {
	return a.adkAgent
}

// GetStatus 获取智能体状态
func (a *Agent) GetStatus() AgentStatus {
	a.statusMu.RLock()
	defer a.statusMu.RUnlock()
	return a.status
}

// setStatus 设置智能体状态（内部方法）
func (a *Agent) setStatus(status AgentStatus) {
	a.statusMu.Lock()
	defer a.statusMu.Unlock()
	a.status = status
}

// Start 启动智能体
func (a *Agent) Start(ctx context.Context) error {
	if a.GetStatus() == AgentStatusRunning {
		return nil
	}
	a.setStatus(AgentStatusRunning)
	logx.Infof("[Agent] %s started", a.name)
	return nil
}

// Stop 优雅关闭智能体
func (a *Agent) Stop(ctx context.Context) error {
	currentStatus := a.GetStatus()
	if currentStatus == AgentStatusStopped || currentStatus == AgentStatusStopping {
		return nil
	}

	a.setStatus(AgentStatusStopping)
	logx.Infof("[Agent] %s stopping...", a.name)

	// 等待正在处理的请求完成（超时5秒）
	_, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// TODO: 等待Runner处理完成

	a.setStatus(AgentStatusStopped)
	logx.Infof("[Agent] %s stopped", a.name)
	return nil
}

// HealthCheck 健康检查
func (a *Agent) HealthCheck(ctx context.Context) bool {
	status := a.GetStatus()
	if status == AgentStatusError || status == AgentStatusStopped {
		return false
	}
	// 检查最后活跃时间是否超过阈值（默认5分钟）
	if time.Since(a.lastActiveTime) > 5*time.Minute {
		return false
	}
	return true
}

// GetMetrics 获取监控指标
func (a *Agent) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"name":           a.name,
		"status":         a.GetStatus(),
		"start_time":     a.startTime.Format(time.RFC3339),
		"last_active":    a.lastActiveTime.Format(time.RFC3339),
		"uptime_seconds": time.Since(a.startTime).Seconds(),
		"request_count":  a.requestCount.Load(),
		"error_count":    a.errorCount.Load(),
		"error_rate":     float64(a.errorCount.Load()) / math.Max(float64(a.requestCount.Load()), 1),
	}
}

// recordRequest 记录请求（内部方法）
func (a *Agent) recordRequest(success bool) {
	a.requestCount.Add(1)
	if !success {
		a.errorCount.Add(1)
	}
	a.lastActiveTime = time.Now()
}
