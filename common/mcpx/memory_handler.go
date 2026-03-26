package mcpx

import (
	"context"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	defaultExpiration = time.Hour * 24 // 默认过期时间 24 小时
)

// MemoryAsyncResultHandler 内存版异步结果处理器
// 使用 go-zero 的 collection.Cache 实现，适合开发测试和小规模部署
type MemoryAsyncResultHandler struct {
	cache *collection.Cache
	mu    sync.RWMutex
}

// NewMemoryAsyncResultHandler 创建内存版异步结果处理器
func NewMemoryAsyncResultHandler() *MemoryAsyncResultHandler {
	cache, _ := collection.NewCache(defaultExpiration, collection.WithName("async-result"))
	return &MemoryAsyncResultHandler{
		cache: cache,
	}
}

// Save 保存异步结果到内存
func (h *MemoryAsyncResultHandler) Save(ctx context.Context, result *AsyncToolResult) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 设置创建和更新时间
	now := time.Now().Unix()
	if result.CreatedAt == 0 {
		result.CreatedAt = now
	}
	result.UpdatedAt = now
	logx.WithContext(ctx).Debugf("save async result: %+v", result)
	h.cache.Set(result.TaskID, result)
	return nil
}

// Get 根据 task_id 获取结果
func (h *MemoryAsyncResultHandler) Get(ctx context.Context, taskID string) (*AsyncToolResult, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	val, ok := h.cache.Get(taskID)
	if !ok {
		return nil, nil
	}
	result, ok := val.(*AsyncToolResult)
	if !ok {
		return nil, nil
	}
	return result, nil
}

// UpdateProgress 更新进度
func (h *MemoryAsyncResultHandler) UpdateProgress(ctx context.Context, taskID string, progress float64, status string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	val, ok := h.cache.Get(taskID)
	if !ok {
		return nil
	}
	result, ok := val.(*AsyncToolResult)
	if !ok {
		return nil
	}

	result.Progress = progress
	result.Status = status
	result.UpdatedAt = time.Now().Unix()

	h.cache.Set(taskID, result)
	return nil
}

// SetStatus 设置任务状态
func (h *MemoryAsyncResultHandler) SetStatus(ctx context.Context, taskID string, status string) error {
	return h.UpdateProgress(ctx, taskID, 0, status)
}

// SetResult 设置任务结果
func (h *MemoryAsyncResultHandler) SetResult(ctx context.Context, taskID string, result string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	val, ok := h.cache.Get(taskID)
	if !ok {
		return nil
	}
	r, ok := val.(*AsyncToolResult)
	if !ok {
		return nil
	}

	r.Result = result
	r.Status = "completed"
	r.Progress = 100
	r.UpdatedAt = time.Now().Unix()

	h.cache.Set(taskID, r)
	return nil
}

// SetError 设置任务错误
func (h *MemoryAsyncResultHandler) SetError(ctx context.Context, taskID string, errMsg string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	val, ok := h.cache.Get(taskID)
	if !ok {
		return nil
	}
	r, ok := val.(*AsyncToolResult)
	if !ok {
		return nil
	}

	r.Error = errMsg
	r.Status = "failed"
	r.UpdatedAt = time.Now().Unix()

	h.cache.Set(taskID, r)
	return nil
}

// Delete 删除任务结果
func (h *MemoryAsyncResultHandler) Delete(ctx context.Context, taskID string) error {
	h.cache.Del(taskID)
	return nil
}

// Exists 检查任务是否存在
func (h *MemoryAsyncResultHandler) Exists(ctx context.Context, taskID string) bool {
	_, ok := h.cache.Get(taskID)
	return ok
}
