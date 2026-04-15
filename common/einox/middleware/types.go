package middleware

import (
	"github.com/cloudwego/eino/schema"
)

// =============================================================================
// 中断相关类型定义
// =============================================================================

// ApprovalInfo 审批信息，用于工具调用审批
type ApprovalInfo struct {
	ToolName        string `json:"tool_name"`
	ArgumentsInJSON string `json:"arguments_in_json"`
	Question        string `json:"question,omitempty"`
	Required        bool   `json:"required,omitempty"`
}

// ApprovalResult 审批结果
type ApprovalResult struct {
	Approved         bool    `json:"approved"`
	DisapproveReason *string `json:"disapprove_reason,omitempty"`
}

// ChoiceInfo 选择信息，用于单选/多选
type ChoiceInfo struct {
	ToolName        string         `json:"tool_name"`
	ArgumentsInJSON string         `json:"arguments_in_json"`
	Question        string         `json:"question"`
	Options         []ChoiceOption `json:"options"`
	Required        bool           `json:"required,omitempty"`
	MinSelect       int            `json:"min_select,omitempty"`
	MaxSelect       int            `json:"max_select,omitempty"`
}

// HumanConfirmInfo 人工确认信息，用于需要用户确认的操作
type HumanConfirmInfo struct {
	ToolName        string `json:"tool_name"`
	ArgumentsInJSON string `json:"arguments_in_json"`
	Message         string `json:"message"`
}

// HumanConfirmResult 确认结果
type HumanConfirmResult struct {
	Confirmed bool    `json:"confirmed"`
	Comment   *string `json:"comment,omitempty"`
}

func init() {
	schema.Register[*ApprovalInfo]()
	schema.Register[*ApprovalResult]()
	schema.Register[*ChoiceInfo]()
	schema.Register[*ChoiceResult]()
	schema.Register[*HumanConfirmInfo]()
	schema.Register[*HumanConfirmResult]()
}
