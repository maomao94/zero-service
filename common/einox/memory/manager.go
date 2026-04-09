package memory

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
)

// =============================================================================
// MemoryManager 记忆管理器
// =============================================================================

// MemoryManager 记忆管理器
//
// 负责管理用户记忆、会话摘要和对话历史。
// 支持异步处理、定期清理、任务去重等功能。
type MemoryManager struct {
	// 存储接口
	storage Storage
	// 记忆配置
	config *MemoryConfig

	// 模型（用于生成摘要和记忆）
	model model.BaseChatModel

	// 摘要触发管理
	summaryTrigger *SummaryTriggerManager

	// 异步处理相关
	taskChannel chan asyncTask
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc

	// 定期清理相关
	cleanupTicker *time.Ticker
	cleanupWg     sync.WaitGroup
	cleanupCtx    context.Context
	cleanupCancel context.CancelFunc

	// 异步任务队列统计
	taskQueueStats TaskQueueStats

	// 异步任务处理去重标记
	pendingTasks sync.Map

	// 外部注入的清理函数
	CleanupOldMessagesFunc     func(ctx context.Context) error
	CleanupMessagesByLimitFunc func(ctx context.Context) error
}

// asyncTask 异步任务结构
type asyncTask struct {
	taskType  string // "memory" 或 "summary"
	userID    string
	sessionID string
}

// NewMemoryManager 创建记忆管理器
//
// cm: ChatModel，用于生成摘要和记忆分析
// storage: 存储接口
// config: 配置，传 nil 使用默认配置
func NewMemoryManager(cm model.BaseChatModel, storage Storage, config *MemoryConfig) (*MemoryManager, error) {
	if config == nil {
		config = DefaultMemoryConfig()
	}

	// 填充零值字段的默认值
	defaults := DefaultMemoryConfig()
	if config.MemoryLimit <= 0 {
		config.MemoryLimit = defaults.MemoryLimit
	}
	if config.AsyncWorkerPoolSize <= 0 {
		config.AsyncWorkerPoolSize = defaults.AsyncWorkerPoolSize
	}
	if config.SummaryTrigger.MessageThreshold <= 0 {
		config.SummaryTrigger.MessageThreshold = defaults.SummaryTrigger.MessageThreshold
	}
	if config.Cleanup.CleanupInterval <= 0 {
		config.Cleanup.CleanupInterval = defaults.Cleanup.CleanupInterval
	}
	if config.Cleanup.SessionCleanupInterval <= 0 {
		config.Cleanup.SessionCleanupInterval = defaults.Cleanup.SessionCleanupInterval
	}
	if config.Cleanup.SessionRetentionTime <= 0 {
		config.Cleanup.SessionRetentionTime = defaults.Cleanup.SessionRetentionTime
	}
	if config.Cleanup.MessageHistoryLimit <= 0 {
		config.Cleanup.MessageHistoryLimit = defaults.Cleanup.MessageHistoryLimit
	}

	// 初始化存储
	if err := storage.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())

	manager := &MemoryManager{
		storage:        storage,
		config:         config,
		model:          cm,
		summaryTrigger: NewSummaryTriggerManager(config.SummaryTrigger),
		ctx:            ctx,
		cancel:         cancel,
		cleanupCtx:     cleanupCtx,
		cleanupCancel:  cleanupCancel,
	}

	// 初始化 goroutine 池
	queueCapacity := config.AsyncWorkerPoolSize * 10
	manager.taskChannel = make(chan asyncTask, queueCapacity)
	manager.taskQueueStats.QueueCapacity = queueCapacity
	manager.startAsyncWorkers()

	// 启动定期清理任务
	manager.startPeriodicCleanup()

	return manager, nil
}

// startAsyncWorkers 启动异步工作 goroutine 池
func (m *MemoryManager) startAsyncWorkers() {
	for i := 0; i < m.config.AsyncWorkerPoolSize; i++ {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			for {
				select {
				case <-m.ctx.Done():
					return
				case task, ok := <-m.taskChannel:
					if !ok {
						return
					}
					// 任务已取出，从去重标记中移除
					taskKey := fmt.Sprintf("%s:%s:%s", task.taskType, task.userID, task.sessionID)
					m.pendingTasks.Delete(taskKey)
					m.processAsyncTask(task)
					atomic.AddInt64(&m.taskQueueStats.ProcessedTasks, 1)
				}
			}
		}()
	}
	m.taskQueueStats.ActiveWorkers = m.config.AsyncWorkerPoolSize
}

// submitAsyncTask 提交异步任务（带去重）
func (m *MemoryManager) submitAsyncTask(task asyncTask) bool {
	taskKey := fmt.Sprintf("%s:%s:%s", task.taskType, task.userID, task.sessionID)

	// 如果相同签名的任务已在队列中，则丢弃重复提交
	if _, loaded := m.pendingTasks.LoadOrStore(taskKey, struct{}{}); loaded {
		logx.Debugf("异步任务去重: 已存在相同的待处理任务, 类型: %s, 用户: %s", task.taskType, task.userID)
		return true
	}

	select {
	case m.taskChannel <- task:
		return true
	default:
		// 队列满
		m.pendingTasks.Delete(taskKey)
		atomic.AddInt64(&m.taskQueueStats.DroppedTasks, 1)
		logx.Errorf("异步任务队列已满，丢弃任务. 队列: %d/%d", len(m.taskChannel), m.taskQueueStats.QueueCapacity)
		return false
	}
}

// GetTaskQueueStats 获取异步任务队列统计
func (m *MemoryManager) GetTaskQueueStats() TaskQueueStats {
	stats := TaskQueueStats{
		QueueCapacity:  m.taskQueueStats.QueueCapacity,
		ActiveWorkers:  m.taskQueueStats.ActiveWorkers,
		ProcessedTasks: atomic.LoadInt64(&m.taskQueueStats.ProcessedTasks),
		DroppedTasks:   atomic.LoadInt64(&m.taskQueueStats.DroppedTasks),
	}
	if m.taskChannel != nil {
		stats.QueueSize = len(m.taskChannel)
		if stats.QueueCapacity > 0 {
			stats.QueueUtilization = float64(stats.QueueSize) / float64(stats.QueueCapacity)
		}
	}
	return stats
}

// startPeriodicCleanup 启动定期清理任务
func (m *MemoryManager) startPeriodicCleanup() {
	m.cleanupTicker = time.NewTicker(time.Duration(m.config.Cleanup.CleanupInterval) * time.Hour)
	m.cleanupWg.Add(1)
	go func() {
		defer m.cleanupWg.Done()
		for {
			select {
			case <-m.cleanupCtx.Done():
				m.cleanupTicker.Stop()
				return
			case <-m.cleanupTicker.C:
				m.performPeriodicCleanup(m.cleanupCtx)
			}
		}
	}()
}

// performPeriodicCleanup 执行定期清理
func (m *MemoryManager) performPeriodicCleanup(parentCtx context.Context) {
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Minute)
	defer cancel()

	// 1. 清理旧的会话状态
	if m.config.Cleanup.SessionCleanupInterval > 0 {
		sessionRetention := time.Duration(m.config.Cleanup.SessionRetentionTime) * time.Hour
		if err := m.storage.CleanupOldSessions(ctx, sessionRetention); err != nil {
			logx.Errorf("清理旧会话失败: %v", err)
		}
	}

	// 2. 清理旧的消息历史
	if m.CleanupOldMessagesFunc != nil {
		if err := m.CleanupOldMessagesFunc(ctx); err != nil {
			logx.Errorf("清理旧消息失败: %v", err)
		}
	}

	// 3. 按数量限制清理消息
	if m.CleanupMessagesByLimitFunc != nil {
		if err := m.CleanupMessagesByLimitFunc(ctx); err != nil {
			logx.Errorf("按数量清理消息失败: %v", err)
		}
	}
}

// processAsyncTask 处理异步任务
func (m *MemoryManager) processAsyncTask(task asyncTask) {
	switch task.taskType {
	case "memory":
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		m.analyzeAndUpdateUserMemory(ctx, task.userID, task.sessionID)
	case "summary":
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := m.updateSessionSummary(ctx, task.userID, task.sessionID); err != nil {
			logx.Errorf("异步更新会话摘要失败: sessionID=%s, userID=%s, err=%v", task.sessionID, task.userID, err)
		} else {
			m.summaryTrigger.MarkSummaryUpdated(sessionKey(task.userID, task.sessionID))
		}
	}
}

// =============================================================================
// 对话消息处理
// =============================================================================

// ProcessUserMessage 处理用户消息
//
// 保存用户消息，并根据配置决定是否创建用户记忆或更新会话摘要。
func (m *MemoryManager) ProcessUserMessage(ctx context.Context, userID, sessionID, content string, parts []any) error {
	if userID == "" {
		return fmt.Errorf("用户ID不能为空")
	}
	if sessionID == "" {
		return fmt.Errorf("会话ID不能为空")
	}
	if content == "" && len(parts) == 0 {
		return fmt.Errorf("消息内容不能为空")
	}

	// 检查消息数量并可能清理旧消息
	if m.config.Cleanup.MessageHistoryLimit > 0 {
		count, err := m.storage.GetMessageCount(ctx, userID, sessionID)
		if err != nil {
			logx.Errorf("获取消息数量失败: %v", err)
		} else if count >= m.config.Cleanup.MessageHistoryLimit {
			err := m.storage.CleanupMessagesByLimit(ctx, userID, sessionID, m.config.Cleanup.MessageHistoryLimit-1)
			if err != nil {
				logx.Errorf("清理超限消息失败: %v", err)
			}
		}
	}

	// 保存用户消息
	msg := &ConversationMessage{
		SessionID: sessionID,
		UserID:    userID,
		Role:      "user",
		Content:   content,
	}
	return m.storage.SaveMessage(ctx, msg)
}

// ProcessAssistantMessage 处理助手回复消息
//
// 保存助手消息，并根据配置决定是否更新摘要或记忆。
func (m *MemoryManager) ProcessAssistantMessage(ctx context.Context, userID, sessionID, content string) error {
	if userID == "" {
		return fmt.Errorf("用户ID不能为空")
	}
	if sessionID == "" {
		return fmt.Errorf("会话ID不能为空")
	}
	if content == "" {
		return fmt.Errorf("消息内容不能为空")
	}

	// 保存助手消息
	msg := &ConversationMessage{
		SessionID: sessionID,
		UserID:    userID,
		Role:      "assistant",
		Content:   content,
	}
	if err := m.storage.SaveMessage(ctx, msg); err != nil {
		return fmt.Errorf("保存助手消息失败: %w", err)
	}

	// 如果启用了会话摘要，检查是否需要更新
	if m.config.EnableSessionSummary {
		shouldTrigger, err := m.shouldTriggerSummaryUpdate(ctx, userID, sessionID)
		if err != nil {
			logx.Errorf("检查摘要触发条件失败: %v", err)
		} else if shouldTrigger {
			m.submitAsyncTask(asyncTask{
				taskType:  "summary",
				userID:    userID,
				sessionID: sessionID,
			})
		}
	}

	// 如果启用了用户记忆，在 AI 回复后触发分析
	if m.config.EnableUserMemories {
		m.submitAsyncTask(asyncTask{
			taskType:  "memory",
			userID:    userID,
			sessionID: sessionID,
		})
	}

	return nil
}

// =============================================================================
// 用户记忆
// =============================================================================

// analyzeAndUpdateUserMemory 分析并更新用户记忆
func (m *MemoryManager) analyzeAndUpdateUserMemory(ctx context.Context, userID, sessionID string) {
	// 获取最近的对话
	messages, err := m.storage.GetMessages(ctx, userID, sessionID, m.config.MemoryLimit)
	if err != nil || len(messages) == 0 {
		return
	}

	// 获取现有记忆
	existingMemory, _ := m.storage.GetUserMemory(ctx, userID)
	existingText := "暂无记忆"
	if existingMemory != nil && existingMemory.Memory != "" {
		existingText = existingMemory.Memory
	}

	// 构建对话文本
	dialogueText := MessagesToText(messages)

	// 调用模型分析
	param, err := m.analyzeUserMemory(ctx, existingText, dialogueText)
	if err != nil {
		logx.Errorf("分析用户记忆失败: %v", err)
		return
	}

	// 如果需要更新
	if param.Op == string(UserMemoryOpUpdate) && param.Memory != "" {
		memory := &UserMemory{
			UserID: userID,
			Memory: param.Memory,
		}
		if existingMemory != nil {
			memory.ID = existingMemory.ID
			memory.CreatedAt = existingMemory.CreatedAt
		}
		if err := m.storage.SaveUserMemory(ctx, memory); err != nil {
			logx.Errorf("保存用户记忆失败: %v", err)
		}
	}
}

// analyzeUserMemory 分析用户记忆
func (m *MemoryManager) analyzeUserMemory(ctx context.Context, existingMemory, newDialogue string) (*UserMemoryAnalyzerParam, error) {
	prompt := fmt.Sprintf(DefaultUserMemoryPrompt, existingMemory, newDialogue)
	msg := schema.UserMessage(prompt)

	resp, err := m.model.Generate(ctx, []*schema.Message{msg})
	if err != nil {
		return nil, err
	}

	content := resp.Content
	if content == "NO_UPDATE" {
		return &UserMemoryAnalyzerParam{Op: string(UserMemoryOpNoop)}, nil
	}

	return &UserMemoryAnalyzerParam{
		Op:     string(UserMemoryOpUpdate),
		Memory: content,
	}, nil
}

// GetUserMemory 获取用户记忆
func (m *MemoryManager) GetUserMemory(ctx context.Context, userID string) (*UserMemory, error) {
	return m.storage.GetUserMemory(ctx, userID)
}

// GetRecentMessages 获取最近的会话消息（用于 Agent 上下文）
func (m *MemoryManager) GetRecentMessages(ctx context.Context, userID, sessionID string, limit int) ([]*ConversationMessage, error) {
	if limit <= 0 {
		limit = m.config.MemoryLimit
	}
	return m.storage.GetMessages(ctx, userID, sessionID, limit)
}

// =============================================================================
// 会话摘要
// =============================================================================

// shouldTriggerSummaryUpdate 检查是否应该触发摘要更新
func (m *MemoryManager) shouldTriggerSummaryUpdate(ctx context.Context, userID, sessionID string) (bool, error) {
	return m.summaryTrigger.ShouldTrigger(ctx, sessionKey(userID, sessionID))
}

// updateSessionSummary 更新会话摘要
func (m *MemoryManager) updateSessionSummary(ctx context.Context, userID, sessionID string) error {
	// 获取最近的对话
	messages, err := m.storage.GetMessages(ctx, userID, sessionID, m.config.MemoryLimit)
	if err != nil {
		return fmt.Errorf("获取消息失败: %w", err)
	}
	if len(messages) == 0 {
		return nil
	}

	// 获取现有摘要
	existingSummary, _ := m.storage.GetSessionSummary(ctx, userID, sessionID)
	dialogueText := MessagesToText(messages)

	var summaryText string
	if existingSummary != nil && existingSummary.Summary != "" {
		// 增量更新
		summaryText, err = m.generateIncrementalSummary(ctx, existingSummary.Summary, dialogueText)
	} else {
		// 全新生成
		summaryText, err = m.generateSummary(ctx, dialogueText)
	}

	if err != nil {
		return fmt.Errorf("生成摘要失败: %w", err)
	}

	if summaryText == "NO_UPDATE" {
		return nil
	}

	summary := &SessionSummary{
		SessionID: sessionID,
		UserID:    userID,
		Summary:   summaryText,
	}
	if existingSummary != nil {
		summary.ID = existingSummary.ID
		summary.CreatedAt = existingSummary.CreatedAt
	}

	return m.storage.SaveSessionSummary(ctx, summary)
}

// generateSummary 生成会话摘要
func (m *MemoryManager) generateSummary(ctx context.Context, dialogue string) (string, error) {
	prompt := fmt.Sprintf(DefaultSessionSummaryPrompt, dialogue)
	msg := schema.UserMessage(prompt)

	resp, err := m.model.Generate(ctx, []*schema.Message{msg})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// generateIncrementalSummary 增量更新摘要
func (m *MemoryManager) generateIncrementalSummary(ctx context.Context, existingSummary, newDialogue string) (string, error) {
	updateTime := time.Now().Format("2006-01-02 15:04:05")
	prompt := fmt.Sprintf(DefaultIncrementalSessionSummaryPrompt, existingSummary, newDialogue, updateTime)
	msg := schema.UserMessage(prompt)

	resp, err := m.model.Generate(ctx, []*schema.Message{msg})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// GetSessionSummary 获取会话摘要
func (m *MemoryManager) GetSessionSummary(ctx context.Context, userID, sessionID string) (*SessionSummary, error) {
	return m.storage.GetSessionSummary(ctx, userID, sessionID)
}

// =============================================================================
// 辅助函数
// =============================================================================

// sessionKey 生成会话 key
func sessionKey(userID, sessionID string) string {
	return userID + ":" + sessionID
}

// =============================================================================
// SummaryTriggerManager 摘要触发管理器
// =============================================================================

// SummaryTriggerManager 管理摘要更新触发逻辑
type SummaryTriggerManager struct {
	config SummaryTriggerConfig
	// 会话状态：key 为 sessionKey(userID, sessionID)
	sessions sync.Map // map[string]*sessionState
}

// sessionState 会话状态
type sessionState struct {
	lastMessageCount int       // 上次触发时的消息数量
	lastUpdateTime   time.Time // 上次更新时间
}

// NewSummaryTriggerManager 创建摘要触发管理器
func NewSummaryTriggerManager(config SummaryTriggerConfig) *SummaryTriggerManager {
	return &SummaryTriggerManager{
		config: config,
	}
}

// ShouldTrigger 检查是否应该触发摘要更新
func (s *SummaryTriggerManager) ShouldTrigger(ctx context.Context, key string) (bool, error) {
	state := s.getOrCreateState(key)

	// 检查最小间隔
	if s.config.MinInterval > 0 {
		minInterval := time.Duration(s.config.MinInterval) * time.Second
		if time.Since(state.lastUpdateTime) < minInterval {
			return false, nil
		}
	}

	switch s.config.Strategy {
	case TriggerAlways:
		return true, nil

	case TriggerByMessages:
		return state.lastMessageCount >= s.config.MessageThreshold, nil

	case TriggerByTime:
		return time.Since(state.lastUpdateTime) >= time.Duration(s.config.MinInterval)*time.Second, nil

	case TriggerSmart:
		// 智能策略：结合消息数量和时间间隔
		if state.lastMessageCount >= s.config.MessageThreshold {
			return true, nil
		}
		// 长时间会话后也触发
		longInterval := time.Duration(s.config.MinInterval*10) * time.Second
		if time.Since(state.lastUpdateTime) >= longInterval && state.lastMessageCount > 0 {
			return true, nil
		}
		return false, nil

	default:
		return false, nil
	}
}

// MarkMessageProcessed 标记消息已处理
func (s *SummaryTriggerManager) MarkMessageProcessed(key string) {
	state := s.getOrCreateState(key)
	state.lastMessageCount++
}

// MarkSummaryUpdated 标记摘要已更新
func (s *SummaryTriggerManager) MarkSummaryUpdated(key string) {
	state := s.getOrCreateState(key)
	state.lastMessageCount = 0
	state.lastUpdateTime = time.Now()
}

// getOrCreateState 获取或创建会话状态
func (s *SummaryTriggerManager) getOrCreateState(key string) *sessionState {
	actual, _ := s.sessions.LoadOrStore(key, &sessionState{
		lastMessageCount: 0,
		lastUpdateTime:   time.Time{},
	})
	return actual.(*sessionState)
}

// ResetSession 重置会话状态
func (s *SummaryTriggerManager) ResetSession(key string) {
	s.sessions.Delete(key)
}

// Close 关闭记忆管理器
func (m *MemoryManager) Close() {
	// 取消清理任务
	if m.cleanupCancel != nil {
		m.cleanupCancel()
	}
	m.cleanupWg.Wait()

	// 取消异步任务
	if m.cancel != nil {
		m.cancel()
	}
	close(m.taskChannel)
	m.wg.Wait()
}
