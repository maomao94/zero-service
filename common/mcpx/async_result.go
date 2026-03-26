package mcpx

import "context"

// AsyncToolResult 异步工具执行结果
type AsyncToolResult struct {
	TaskID    string  // 任务ID
	Status    string  // pending | completed | failed
	Result    string  // 成功时的结果
	Error     string  // 失败时的错误信息
	Progress  float64 // 进度 0-100
	CreatedAt int64   // 创建时间戳
	UpdatedAt int64   // 更新时间戳
}

// AsyncResultHandler 异步结果处理器接口
// 二开可实现 Redis/gRPC/MySQL 等多种存储方式
type AsyncResultHandler interface {
	// Save 保存异步结果
	Save(ctx context.Context, result *AsyncToolResult) error
	// Get 根据 task_id 获取结果
	Get(ctx context.Context, taskID string) (*AsyncToolResult, error)
	// UpdateProgress 更新进度
	UpdateProgress(ctx context.Context, taskID string, progress float64, status string) error
}

// CallToolAsyncRequest 异步工具调用请求
type CallToolAsyncRequest struct {
	Name          string             // 工具名称
	Args          map[string]any     // 工具参数
	ResultHandler AsyncResultHandler // 结果存储处理器
	OnProgress    ProgressCallback   // 进度回调（可选）
}
