package middleware

import (
	"github.com/cloudwego/eino/schema"
)

// =============================================================================
// Approval 类中断 —— 两态（approve / deny）
// =============================================================================

// ApprovalInfo 审批信息，用于工具调用审批。
type ApprovalInfo struct {
	ToolName        string `json:"tool_name"`
	ArgumentsInJSON string `json:"arguments_in_json"`
	Question        string `json:"question,omitempty"`
	Detail          string `json:"detail,omitempty"`
	Required        bool   `json:"required,omitempty"`
}

// ApprovalResult 审批结果。
type ApprovalResult struct {
	Approved         bool    `json:"approved"`
	DisapproveReason *string `json:"disapprove_reason,omitempty"`
}

// =============================================================================
// Select 类中断 —— 单选 / 多选
// =============================================================================

// InterruptOption 选项定义，供单选 / 多选类中断使用。
type InterruptOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Desc  string `json:"desc,omitempty"`
}

// SelectInfo 选择类中断。Multi=false 单选，Multi=true 多选。
// MinSelect / MaxSelect 仅多选时生效；MinSelect=0 表示可以不选。
type SelectInfo struct {
	ToolName        string            `json:"tool_name"`
	ArgumentsInJSON string            `json:"arguments_in_json,omitempty"`
	Question        string            `json:"question"`
	Options         []InterruptOption `json:"options"`
	Multi           bool              `json:"multi"`
	MinSelect       int               `json:"min_select,omitempty"`
	MaxSelect       int               `json:"max_select,omitempty"`
	Required        bool              `json:"required,omitempty"`
}

// SelectResult 选择类中断的恢复结果。
type SelectResult struct {
	SelectedIDs []string `json:"selected_ids"`
	Cancelled   bool     `json:"cancelled,omitempty"`
	Reason      string   `json:"reason,omitempty"`
}

// =============================================================================
// TextInput 类中断 —— 自由文本输入
// =============================================================================

// TextInputInfo 向用户征集一段自由文本。
type TextInputInfo struct {
	ToolName        string `json:"tool_name"`
	ArgumentsInJSON string `json:"arguments_in_json,omitempty"`
	Question        string `json:"question"`
	Placeholder     string `json:"placeholder,omitempty"`
	Multiline       bool   `json:"multiline,omitempty"`
	Required        bool   `json:"required,omitempty"`
}

// TextInputResult 文本输入结果。Cancelled=true 时 Text 可能为空。
type TextInputResult struct {
	Text      string `json:"text"`
	Cancelled bool   `json:"cancelled,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// =============================================================================
// FormInput 类中断 —— 结构化表单
// =============================================================================

// FormField 表单字段定义。
// Type: string | number | boolean
type FormField struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Required    bool   `json:"required,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Default     string `json:"default,omitempty"`
}

// FormInputInfo 向用户征集结构化表单。
type FormInputInfo struct {
	ToolName        string      `json:"tool_name"`
	ArgumentsInJSON string      `json:"arguments_in_json,omitempty"`
	Question        string      `json:"question"`
	Fields          []FormField `json:"fields"`
	Required        bool        `json:"required,omitempty"`
}

// FormInputResult 表单提交结果。Values 的 key 是 Field.Name，value 是字符串形式
// （数值/布尔由工具层自己 parse，避免 map[string]any 反序列化歧义）。
type FormInputResult struct {
	Values    map[string]string `json:"values"`
	Cancelled bool              `json:"cancelled,omitempty"`
	Reason    string            `json:"reason,omitempty"`
}

// =============================================================================
// InfoAck 类中断 —— 展示信息 + 点击确认继续
// =============================================================================

// InfoAckInfo 向用户展示一段信息，要求点击确认继续。
type InfoAckInfo struct {
	ToolName        string `json:"tool_name"`
	ArgumentsInJSON string `json:"arguments_in_json,omitempty"`
	Title           string `json:"title"`
	Body            string `json:"body"` // 支持 markdown
}

// InfoAckResult 用户是否确认。Ack=false 视为取消。
type InfoAckResult struct {
	Ack    bool   `json:"ack"`
	Reason string `json:"reason,omitempty"`
}

func init() {
	schema.Register[*ApprovalInfo]()
	schema.Register[*ApprovalResult]()
	schema.Register[*SelectInfo]()
	schema.Register[*SelectResult]()
	schema.Register[*TextInputInfo]()
	schema.Register[*TextInputResult]()
	schema.Register[*FormInputInfo]()
	schema.Register[*FormInputResult]()
	schema.Register[*InfoAckInfo]()
	schema.Register[*InfoAckResult]()
}
