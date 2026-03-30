package mcpx

import "context"

// ProgressMessage 进度消息记录
// 记录 MCP 服务器发送的所有进度通知，用于前端展示消息历史
type ProgressMessage struct {
	Progress float64 `json:"progress"` // 当前进度百分比 0-100
	Total    float64 `json:"total"`    // 进度总值
	Message  string  `json:"message"`  // 消息内容
	Time     int64   `json:"time"`     // 时间戳
}

// AsyncToolResult 异步工具执行结果
// 记录任务从创建到完成的完整状态
type AsyncToolResult struct {
	TaskID    string            `json:"task_id"`    // 任务唯一标识符
	Status    string            `json:"status"`     // 任务状态：pending/completed/failed
	Result    string            `json:"result"`     // 任务成功完成时的执行结果
	Error     string            `json:"error"`      // 任务失败时的错误信息
	Progress  float64           `json:"progress"`   // 当前进度百分比 0-100
	Total     float64           `json:"total"`      // 进度总值
	Messages  []ProgressMessage `json:"messages"`   // 消息历史（MCP 进度通知列表）
	CreatedAt int64             `json:"created_at"` // 创建时间戳
	UpdatedAt int64             `json:"updated_at"` // 更新时间戳
}

// AsyncResultStore 异步结果存储接口
// 二开可实现 Redis/MySQL/文件等持久化存储
type AsyncResultStore interface {
	// Save 保存异步结果（全量保存，会合并消息历史）
	Save(ctx context.Context, result *AsyncToolResult) error
	// Get 根据 task_id 获取结果
	Get(ctx context.Context, taskID string) (*AsyncToolResult, error)
	// UpdateProgress 更新进度，追加消息到历史
	UpdateProgress(ctx context.Context, taskID string, progress, total float64, message string) error
	// Exists 检查任务是否存在
	Exists(ctx context.Context, taskID string) bool

	// List 分页列表查询
	List(ctx context.Context, req *ListAsyncResultsReq) (*ListAsyncResultsResp, error)
	// Stats 统计查询
	Stats(ctx context.Context) (*AsyncResultStats, error)
}

// ListAsyncResultsReq 分页查询请求
type ListAsyncResultsReq struct {
	Status    string // 过滤状态: pending/completed/failed（空字符串表示全部）
	StartTime int64  // 开始时间戳（毫秒），0 表示不限制
	EndTime   int64  // 结束时间戳（毫秒），0 表示不限制
	Page      int    // 页码 (从1开始)
	PageSize  int    // 每页数量
	SortField string // 排序字段: created_at/updated_at/progress
	SortOrder string // 排序方向: asc/desc
}

// ListAsyncResultsResp 分页查询响应
type ListAsyncResultsResp struct {
	Items      []*AsyncToolResult `json:"items"`       // 数据列表
	Total      int64              `json:"total"`       // 总数
	Page       int                `json:"page"`        // 当前页
	PageSize   int                `json:"page_size"`   // 每页数量
	TotalPages int                `json:"total_pages"` // 总页数
}

// AsyncResultStats 统计信息
type AsyncResultStats struct {
	Total       int64   `json:"total"`        // 任务总数
	Pending     int64   `json:"pending"`      // 待处理
	Completed   int64   `json:"completed"`    // 已完成
	Failed      int64   `json:"failed"`       // 失败
	SuccessRate float64 `json:"success_rate"` // 成功率
}

// TaskObserver 任务观察者接口
// 观察任务状态变化，触发后续逻辑（如 WebSocket 通知、业务回调等）
// 不负责存储，只负责处理状态变化事件
type TaskObserver interface {
	// OnProgress 进度更新回调
	// - taskID: 任务ID
	// - progress: 当前进度百分比 0-100
	// - total: 进度总值
	// - message: MCP 服务器返回的消息
	OnProgress(taskID string, progress, total float64, message string)

	// OnComplete 任务完成回调
	// - taskID: 任务ID
	// - message: MCP 服务器返回的最终消息
	// - result: 任务结果（包含最终状态、结果或错误）
	OnComplete(taskID string, message string, result *AsyncToolResult)
}

// CallToolAsyncRequest 异步工具调用请求
type CallToolAsyncRequest struct {
	Name         string           // 工具名称
	Args         map[string]any   // 工具参数
	ResultStore  AsyncResultStore // 异步结果存储（必填）
	TaskObserver TaskObserver     // 任务观察者（可选，用于实时通知）
}
