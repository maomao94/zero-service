package resume

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"zero-service/common/einox/router"
)

// =============================================================================
// 错误定义
// =============================================================================

var (
	// ErrCheckPointNotFound 检查点不存在
	ErrCheckPointNotFound = errors.New("checkpoint not found")
	// ErrInvalidCheckPoint 无效的检查点
	ErrInvalidCheckPoint = errors.New("invalid checkpoint")
	// ErrResumeContextMissing 恢复上下文缺失
	ErrResumeContextMissing = errors.New("resume context missing")
)

// =============================================================================
// ResumeHandler 中断恢复处理器
// =============================================================================

// ResumeHandler 中断恢复处理器接口
type ResumeHandler interface {
	// Resume 恢复中断的请求
	Resume(ctx context.Context, req *ResumeRequest) (*ResumeResponse, error)
}

// ResumeRequest 恢复请求
type ResumeRequest struct {
	CheckPointID string      `json:"checkpoint_id"` // 检查点 ID
	UserID       string      `json:"user_id"`       // 用户 ID
	SessionID    string      `json:"session_id"`    // 会话 ID
	UserInput    string      `json:"user_input"`    // 用户输入（如批准/拒绝）
	Choice       UserChoice  `json:"choice"`        // 用户选择
	ExtraData    interface{} `json:"extra_data"`    // 额外数据
}

// ResumeResponse 恢复响应
type ResumeResponse struct {
	Success      bool               `json:"success"`
	Content      string             `json:"content"`
	ToolCalls    []*schema.ToolCall `json:"tool_calls"`
	Error        string             `json:"error,omitempty"`
	NeedsConfirm bool               `json:"needs_confirm"`
	ConfirmInfo  *ApprovalInfo      `json:"confirm_info,omitempty"`
}

// UserChoice 用户选择
type UserChoice string

const (
	ChoiceApprove  UserChoice = "approve"  // 批准
	ChoiceReject   UserChoice = "reject"   // 拒绝
	ChoiceCancel   UserChoice = "cancel"   // 取消
	ChoiceContinue UserChoice = "continue" // 继续
	ChoiceModify   UserChoice = "modify"   // 修改后继续
)

// =============================================================================
// ResumeService 恢复服务
// =============================================================================

// AgentRunner Agent 运行器接口
type AgentRunner interface {
	// Stream 运行 Agent 并返回原始事件流
	Stream(ctx context.Context, input string, opts ...RunOption) (*adk.AsyncIterator[*adk.AgentEvent], error)
}

// RunOption 运行选项
type RunOption func(*RunOptions)

// RunOptions 运行选项
type RunOptions struct {
	UserID    string
	SessionID string
	System    string
}

// WithUserID 设置用户 ID
func WithUserID(userID string) RunOption {
	return func(o *RunOptions) {
		o.UserID = userID
	}
}

// WithSessionID 设置会话 ID
func WithSessionID(sessionID string) RunOption {
	return func(o *RunOptions) {
		o.SessionID = sessionID
	}
}

// WithSystem 设置系统消息
func WithSystem(system string) RunOption {
	return func(o *RunOptions) {
		o.System = system
	}
}

// ResumeService 恢复服务
type ResumeService struct {
	store     router.CheckPointStore
	agent     AgentRunner
	interrupt InterruptHandler
}

// NewResumeService 创建恢复服务
func NewResumeService(store router.CheckPointStore, agent AgentRunner, interrupt InterruptHandler) *ResumeService {
	return &ResumeService{
		store:     store,
		agent:     agent,
		interrupt: interrupt,
	}
}

// Resume 恢复中断的请求
func (s *ResumeService) Resume(ctx context.Context, req *ResumeRequest) (*ResumeResponse, error) {
	// 1. 获取检查点
	data, exists, err := s.store.Get(ctx, req.CheckPointID)
	if err != nil {
		return nil, fmt.Errorf("get checkpoint: %w", err)
	}
	if !exists {
		return &ResumeResponse{
			Success: false,
			Error:   ErrCheckPointNotFound.Error(),
		}, nil
	}

	// 2. 解析检查点数据
	checkPoint, err := router.DecodeCheckPoint(data)
	if err != nil {
		return &ResumeResponse{
			Success: false,
			Error:   fmt.Errorf("decode checkpoint: %w", err).Error(),
		}, nil
	}

	// 3. 构建恢复上下文
	resumeCtx := s.buildResumeContext(checkPoint, req)

	// 4. 使用中断处理器恢复执行
	result, err := s.interrupt.ResumeWithContext(ctx, resumeCtx)
	if err != nil {
		return &ResumeResponse{
			Success: false,
			Error:   fmt.Errorf("resume: %w", err).Error(),
		}, nil
	}

	// 5. 删除检查点
	if err := s.store.Delete(ctx, req.CheckPointID); err != nil {
		// 日志记录，不影响返回
		fmt.Printf("failed to delete checkpoint: %v\n", err)
	}

	return result, nil
}

// buildResumeContext 构建恢复上下文
func (s *ResumeService) buildResumeContext(cp *router.CheckPointData, req *ResumeRequest) *ResumeContext {
	ctx := &ResumeContext{
		CheckPointID: req.CheckPointID,
		UserID:       cp.UserID,
		SessionID:    cp.SessionID,
		State:        cp.State,
		Messages:     cp.Messages,
		ResumeData:   cp.ResumeData,
	}

	// 根据用户选择设置恢复数据
	switch req.Choice {
	case ChoiceApprove:
		ctx.ApprovalResult = &ApprovalResult{
			Approved:   true,
			Choice:     ChoiceApprove,
			Input:      req.UserInput,
			ExtraData:  req.ExtraData,
			ApprovedAt: time.Now(),
		}
	case ChoiceReject:
		ctx.ApprovalResult = &ApprovalResult{
			Approved:   false,
			Choice:     ChoiceReject,
			Input:      req.UserInput,
			ExtraData:  req.ExtraData,
			ApprovedAt: time.Now(),
		}
	case ChoiceModify:
		ctx.ApprovalResult = &ApprovalResult{
			Approved:   true,
			Choice:     ChoiceModify,
			Input:      req.UserInput,
			ExtraData:  req.ExtraData,
			ApprovedAt: time.Now(),
		}
	}

	return ctx
}

// =============================================================================
// InterruptHandler 中断处理器接口
// =============================================================================

// InterruptHandler 中断处理器
type InterruptHandler interface {
	// HandleInterrupt 处理中断
	HandleInterrupt(ctx context.Context, info *InterruptInfo) (*InterruptResponse, error)

	// ResumeWithContext 使用恢复上下文继续执行
	ResumeWithContext(ctx context.Context, resumeCtx *ResumeContext) (*ResumeResponse, error)
}

// InterruptInfo 中断信息
type InterruptInfo struct {
	Type         InterruptType          `json:"type"`
	Message      string                 `json:"message"`
	CheckPoint   *router.CheckPointData `json:"checkpoint,omitempty"`
	Choices      []ChoiceInfo           `json:"choices,omitempty"`
	NeedsConfirm bool                   `json:"needs_confirm"`
}

// InterruptType 中断类型
type InterruptType string

const (
	InterruptTypeApproval InterruptType = "approval" // 需要确认
	InterruptTypeConfirm  InterruptType = "confirm"  // 需要确认
	InterruptTypeSelect   InterruptType = "select"   // 需要选择
)

// InterruptResponse 中断响应
type InterruptResponse struct {
	Interrupted  bool          `json:"interrupted"`
	CheckPointID string        `json:"checkpoint_id,omitempty"`
	Message      string        `json:"message"`
	NeedsConfirm bool          `json:"needs_confirm"`
	ConfirmInfo  *ApprovalInfo `json:"confirm_info,omitempty"`
	SelectInfo   *SelectInfo   `json:"select_info,omitempty"`
}

// ApprovalInfo 审批信息
type ApprovalInfo struct {
	ToolName    string                 `json:"tool_name"`
	ToolArgs    map[string]interface{} `json:"tool_args"`
	Description string                 `json:"description"`
	Choices     []ChoiceInfo           `json:"choices"`
}

// ChoiceInfo 选择信息
type ChoiceInfo struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// SelectInfo 选择信息
type SelectInfo struct {
	Title    string         `json:"title"`
	Options  []SelectOption `json:"options"`
	Multiple bool           `json:"multiple"`
}

// SelectOption 选择选项
type SelectOption struct {
	Value    string `json:"value"`
	Label    string `json:"label"`
	Desc     string `json:"desc,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
}

// ResumeContext 恢复上下文
type ResumeContext struct {
	CheckPointID   string          `json:"checkpoint_id"`
	UserID         string          `json:"user_id"`
	SessionID      string          `json:"session_id"`
	State          json.RawMessage `json:"state"`
	Messages       json.RawMessage `json:"messages"`
	ResumeData     json.RawMessage `json:"resume_data"`
	ApprovalResult *ApprovalResult `json:"approval_result,omitempty"`
	SelectResult   *SelectResult   `json:"select_result,omitempty"`
}

// ApprovalResult 审批结果
type ApprovalResult struct {
	Approved   bool        `json:"approved"`
	Choice     UserChoice  `json:"choice"`
	Input      string      `json:"input,omitempty"`
	ExtraData  interface{} `json:"extra_data,omitempty"`
	ApprovedAt time.Time   `json:"approved_at"`
}

// SelectResult 选择结果
type SelectResult struct {
	Values     []string  `json:"values"`
	SelectedAt time.Time `json:"selected_at"`
}

// =============================================================================
// CheckPointManager 检查点管理器
// =============================================================================

// CheckPointManager 检查点管理器
type CheckPointManager struct {
	store router.CheckPointStore
}

// NewCheckPointManager 创建检查点管理器
func NewCheckPointManager(store router.CheckPointStore) *CheckPointManager {
	return &CheckPointManager{store: store}
}

// Save 保存检查点
func (m *CheckPointManager) Save(ctx context.Context, userID, sessionID string, data *router.CheckPointData) (string, error) {
	checkPointID := fmt.Sprintf("%s:%s:%d", userID, sessionID, time.Now().UnixNano())

	encoded, err := data.Encode()
	if err != nil {
		return "", fmt.Errorf("encode checkpoint: %w", err)
	}

	if err := m.store.Set(ctx, checkPointID, encoded); err != nil {
		return "", fmt.Errorf("set checkpoint: %w", err)
	}

	return checkPointID, nil
}

// Load 加载检查点
func (m *CheckPointManager) Load(ctx context.Context, checkPointID string) (*router.CheckPointData, error) {
	data, exists, err := m.store.Get(ctx, checkPointID)
	if err != nil {
		return nil, fmt.Errorf("get checkpoint: %w", err)
	}
	if !exists {
		return nil, ErrCheckPointNotFound
	}

	cp, err := router.DecodeCheckPoint(data)
	if err != nil {
		return nil, fmt.Errorf("decode checkpoint: %w", err)
	}

	return cp, nil
}

// Delete 删除检查点
func (m *CheckPointManager) Delete(ctx context.Context, checkPointID string) error {
	return m.store.Delete(ctx, checkPointID)
}

// Exists 检查检查点是否存在
func (m *CheckPointManager) Exists(ctx context.Context, checkPointID string) (bool, error) {
	return m.store.Exists(ctx, checkPointID)
}
