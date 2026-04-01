package mcpx

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// MemoryAsyncResultStore 内存版异步结果存储
// 使用 sync.RWMutex + map 实现，适合开发测试和小规模部署
// 支持遍历查询，支持 TTL 过期清理
type MemoryAsyncResultStore struct {
	mu       sync.RWMutex
	data     map[string]*AsyncToolResult
	expiries map[string]int64 // 过期时间戳，0 表示永不过期
}

// NewMemoryAsyncResultStore 创建内存版异步结果存储
func NewMemoryAsyncResultStore() *MemoryAsyncResultStore {
	store := &MemoryAsyncResultStore{
		data:     make(map[string]*AsyncToolResult),
		expiries: make(map[string]int64),
	}
	// 启动过期清理 goroutine
	go store.cleanupLoop()
	return store
}

// cleanupLoop 定期清理过期数据
func (h *MemoryAsyncResultStore) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		h.cleanup()
	}
}

// cleanup 清理过期数据
func (h *MemoryAsyncResultStore) cleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now().UnixMilli()
	for taskID, expiry := range h.expiries {
		if expiry > 0 && expiry < now {
			delete(h.data, taskID)
			delete(h.expiries, taskID)
		}
	}
}

// Save 保存异步结果到内存
// 如果已存在，则合并更新（保留 Messages 历史）
func (h *MemoryAsyncResultStore) Save(ctx context.Context, result *AsyncToolResult) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 先获取现有记录，用于合并
	if existing, ok := h.data[result.TaskID]; ok {
		// 合并：保留 Messages 历史
		if len(existing.Messages) > 0 && len(result.Messages) == 0 {
			result.Messages = existing.Messages
		}
	}

	// 设置创建和更新时间
	now := time.Now().UnixMilli()
	if result.CreatedAt == 0 {
		result.CreatedAt = now
	}
	result.UpdatedAt = now
	logx.WithContext(ctx).Debugf("[MemoryAsyncResultStore] Save: taskID=%s, messages=%d", result.TaskID, len(result.Messages))
	h.data[result.TaskID] = result
	// 默认 24 小时过期
	if h.expiries[result.TaskID] == 0 {
		h.expiries[result.TaskID] = now + 86400000 // 默认 24 小时过期（毫秒）
	}
	return nil
}

// Get 根据 task_id 获取结果
func (h *MemoryAsyncResultStore) Get(ctx context.Context, taskID string) (*AsyncToolResult, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result, ok := h.data[taskID]
	if !ok {
		return nil, errors.New("async result not found")
	}
	return result, nil
}

// UpdateProgress 更新进度，追加消息到历史
// 如果任务不存在，会自动创建（幂等操作）
func (h *MemoryAsyncResultStore) UpdateProgress(ctx context.Context, taskID string, progress, total float64, message string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	result, ok := h.data[taskID]
	if !ok {
		// 自动创建任务（幂等）
		result = &AsyncToolResult{
			TaskID:    taskID,
			Status:    "pending",
			Progress:  progress,
			Total:     total,
			CreatedAt: time.Now().UnixMilli(),
			UpdatedAt: time.Now().UnixMilli(),
		}
		h.data[taskID] = result
	}

	result.Progress = progress
	result.Total = total
	result.UpdatedAt = time.Now().UnixMilli()

	// 保存消息到历史
	result.Messages = append(result.Messages, ProgressMessage{
		Progress: progress,
		Total:    total,
		Message:  message,
		Time:     time.Now().UnixMilli(),
	})

	logx.WithContext(ctx).Debugf("[MemoryAsyncResultStore] UpdateProgress: taskID=%s, progress=%.2f, message=%s, totalMessages=%d",
		taskID, progress, message, len(result.Messages))

	return nil
}

// Exists 检查任务是否存在
func (h *MemoryAsyncResultStore) Exists(ctx context.Context, taskID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.data[taskID]
	return ok
}

// Delete 删除任务结果
func (h *MemoryAsyncResultStore) Delete(ctx context.Context, taskID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.data, taskID)
	delete(h.expiries, taskID)
	return nil
}

// List 分页列表查询
func (h *MemoryAsyncResultStore) List(ctx context.Context, req *ListAsyncResultsReq) (*ListAsyncResultsResp, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 设置默认值
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100 // 最大每页100条
	}
	if req.SortField == "" {
		req.SortField = "created_at"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}

	// 收集所有结果
	allResults := make([]*AsyncToolResult, 0, len(h.data))
	for _, result := range h.data {
		// 状态过滤
		if req.Status != "" && result.Status != req.Status {
			continue
		}
		// 时间范围过滤（使用创建时间）
		if req.StartTime > 0 && result.CreatedAt < req.StartTime {
			continue
		}
		if req.EndTime > 0 && result.CreatedAt > req.EndTime {
			continue
		}
		allResults = append(allResults, result)
	}

	// 排序
	sortFunc := getSortFunc(req.SortField, req.SortOrder == "asc")
	sort.Slice(allResults, func(i, j int) bool {
		return sortFunc(allResults[i], allResults[j])
	})

	// 计算分页
	total := int64(len(allResults))
	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start >= len(allResults) {
		start = len(allResults)
	}
	if end > len(allResults) {
		end = len(allResults)
	}

	items := make([]*AsyncToolResult, 0)
	if start < end {
		items = allResults[start:end]
	}

	return &ListAsyncResultsResp{
		Items:      items,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// Stats 统计查询
func (h *MemoryAsyncResultStore) Stats(ctx context.Context) (*AsyncResultStats, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := &AsyncResultStats{}

	for _, result := range h.data {
		stats.Total++
		switch result.Status {
		case "pending":
			stats.Pending++
		case "completed":
			stats.Completed++
		case "failed":
			stats.Failed++
		}
	}

	// 计算成功率
	if stats.Total > 0 {
		stats.SuccessRate = float64(stats.Completed) / float64(stats.Total) * 100
	}

	return stats, nil
}

// getSortFunc 根据排序字段和方向返回排序函数
func getSortFunc(field string, ascending bool) func(a, b *AsyncToolResult) bool {
	return func(a, b *AsyncToolResult) bool {
		var less bool
		switch field {
		case "created_at":
			less = a.CreatedAt < b.CreatedAt
		case "updated_at":
			less = a.UpdatedAt < b.UpdatedAt
		case "progress":
			less = a.Progress < b.Progress
		default:
			less = a.CreatedAt < b.CreatedAt
		}
		if ascending {
			return less
		}
		return !less
	}
}

// SetStatus 设置任务状态
func (h *MemoryAsyncResultStore) SetStatus(ctx context.Context, taskID string, status string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	result, ok := h.data[taskID]
	if !ok {
		return errors.New("async result not found")
	}

	result.Status = status
	result.UpdatedAt = time.Now().UnixMilli()

	return nil
}

// SetResult 设置任务结果
func (h *MemoryAsyncResultStore) SetResult(ctx context.Context, taskID string, result string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	r, ok := h.data[taskID]
	if !ok {
		return errors.New("async result not found")
	}

	r.Result = result
	r.Status = "completed"
	r.Progress = 100
	r.UpdatedAt = time.Now().UnixMilli()

	return nil
}

// SetError 设置任务错误
func (h *MemoryAsyncResultStore) SetError(ctx context.Context, taskID string, errMsg string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	r, ok := h.data[taskID]
	if !ok {
		return errors.New("async result not found")
	}

	r.Error = errMsg
	r.Status = "failed"
	r.UpdatedAt = time.Now().UnixMilli()

	return nil
}

// DefaultTaskObserver 默认任务观察者
// 只负责将任务状态变化通知到外部（如 WebSocket、Server-Sent Events 等）
// 不负责存储，存储由 Client 内部管理
type DefaultTaskObserver struct {
	callback ProgressCallback // 外部回调
}

// NewDefaultTaskObserver 创建默认任务观察者
// - callback: 外部回调，用于实时通知（如 WebSocket 推送）
func NewDefaultTaskObserver(callback ProgressCallback) *DefaultTaskObserver {
	return &DefaultTaskObserver{
		callback: callback,
	}
}

// OnProgress 进度更新回调，只触发外部回调
func (h *DefaultTaskObserver) OnProgress(taskID string, progress, total float64, message string) {
	if h.callback != nil {
		h.callback(&ProgressInfo{
			Token:    taskID,
			Progress: progress,
			Total:    total,
			Message:  message,
		})
	}
}

// OnComplete 任务完成回调，只触发外部回调
// 存储已在 Client.CallToolAsyncAwait 中处理
func (h *DefaultTaskObserver) OnComplete(taskID string, message string, result *AsyncToolResult) {
	if h.callback != nil {
		h.callback(&ProgressInfo{
			Token:    taskID,
			Progress: 100,
			Total:    100,
			Message:  message,
		})
	}
}

// emptyAsyncResultStore 空实现，不做任何操作
// 用于 Client 默认值，避免空指针
type emptyAsyncResultStore struct{}

func (e *emptyAsyncResultStore) Save(ctx context.Context, result *AsyncToolResult) error {
	return nil
}

func (e *emptyAsyncResultStore) Get(ctx context.Context, taskID string) (*AsyncToolResult, error) {
	return nil, nil
}

func (e *emptyAsyncResultStore) UpdateProgress(ctx context.Context, taskID string, progress, total float64, message string) error {
	return nil
}

func (e *emptyAsyncResultStore) Exists(ctx context.Context, taskID string) bool {
	return false
}

func (e *emptyAsyncResultStore) Delete(ctx context.Context, taskID string) error {
	return nil
}

func (e *emptyAsyncResultStore) List(ctx context.Context, req *ListAsyncResultsReq) (*ListAsyncResultsResp, error) {
	return &ListAsyncResultsResp{}, nil
}

func (e *emptyAsyncResultStore) Stats(ctx context.Context) (*AsyncResultStats, error) {
	return &AsyncResultStats{}, nil
}

func (e *emptyAsyncResultStore) SetStatus(ctx context.Context, taskID string, status string) error {
	return nil
}

func (e *emptyAsyncResultStore) SetResult(ctx context.Context, taskID string, result string) error {
	return nil
}

func (e *emptyAsyncResultStore) SetError(ctx context.Context, taskID string, errMsg string) error {
	return nil
}

// NewEmptyAsyncResultStore 创建空实现，用于 Client 默认值
func NewEmptyAsyncResultStore() AsyncResultStore {
	return &emptyAsyncResultStore{}
}
