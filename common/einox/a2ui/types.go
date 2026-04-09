/*
 * A2UI v0.8 协议实现
 * Agent to UI 协议：定义 Agent 输出如何映射到 UI 组件
 *
 * 特性：
 * - 支持流式渲染：组件可以实时更新，无需等待完整响应
 * - 声明式组件树：Text、Column、Row、Card 等组件通过 ID 引用形成树
 * - Data Binding：文本内容通过 dataKey 绑定到数据模型，实现增量更新
 */

package a2ui

import "encoding/json"

// =============================================================================
// 消息信封（Message Envelope）
// 每行 SSE (data: {...}) 承载一个 Message，Message 是信封结构，每次只会出现一个字段
// =============================================================================

// Message 是 A2UI 的顶层信封结构
type Message struct {
	BeginRendering   *BeginRenderingMsg   `json:"beginRendering,omitempty"`
	SurfaceUpdate    *SurfaceUpdateMsg    `json:"surfaceUpdate,omitempty"`
	DataModelUpdate  *DataModelUpdateMsg  `json:"dataModelUpdate,omitempty"`
	DeleteSurface    *DeleteSurfaceMsg    `json:"deleteSurface,omitempty"`
	InterruptRequest *InterruptRequestMsg `json:"interruptRequest,omitempty"`
}

// =============================================================================
// 会话渲染消息
// =============================================================================

// BeginRenderingMsg 告诉前端"开始渲染一个 surface（会话）"，并指定根节点 ID
type BeginRenderingMsg struct {
	SurfaceID string `json:"surfaceId"` // 会话 ID
	Root      string `json:"root"`      // 根组件 ID
}

// DeleteSurfaceMsg 移除一个 surface 从渲染器
type DeleteSurfaceMsg struct {
	SurfaceID string `json:"surfaceId"`
}

// =============================================================================
// 组件更新消息
// =============================================================================

// SurfaceUpdateMsg 新增或更新一批组件（组件是一个树，用 id 互相引用）
type SurfaceUpdateMsg struct {
	SurfaceID  string      `json:"surfaceId"`
	Components []Component `json:"components"`
}

// Component 是一个命名的 UI 组件定义
type Component struct {
	ID        string         `json:"id"`
	Component ComponentValue `json:"component"`
}

// ComponentValue 持有一种组件类型（只能有一种类型被设置）
type ComponentValue struct {
	Text   *TextComp   `json:"Text,omitempty"`
	Column *ColumnComp `json:"Column,omitempty"`
	Card   *CardComp   `json:"Card,omitempty"`
	Row    *RowComp    `json:"Row,omitempty"`
}

// TextComp 渲染文本。如果设置了 DataKey，则从 data model 读取值
type TextComp struct {
	Value     string `json:"value,omitempty"`     // 静态值
	DataKey   string `json:"dataKey,omitempty"`   // 数据模型绑定（流式更新时使用）
	UsageHint string `json:"usageHint,omitempty"` // 用途提示: "caption" | "body" | "title"
}

// ColumnComp 垂直布局子组件
type ColumnComp struct {
	Children []string `json:"children"` // 子组件 ID 列表
}

// RowComp 水平布局子组件
type RowComp struct {
	Children []string `json:"children"` // 子组件 ID 列表
}

// CardComp 包装子组件的卡片容器
type CardComp struct {
	Children []string `json:"children"` // 子组件 ID 列表
}

// =============================================================================
// 数据模型更新消息
// =============================================================================

// DataModelUpdateMsg 更新 data bindings（用于把流式文本增量更新到某个 Text 组件）
type DataModelUpdateMsg struct {
	SurfaceID string        `json:"surfaceId"`
	Contents  []DataContent `json:"contents"`
}

// DataContent 是用于 DataModelUpdateMsg 的键值绑定
type DataContent struct {
	Key         string `json:"key"`
	ValueString string `json:"valueString,omitempty"`
}

// =============================================================================
// 中断请求消息
// =============================================================================

// InterruptRequestMsg 当 Agent 触发 interrupt（例如审批）时，通知前端展示批准/拒绝入口
type InterruptRequestMsg struct {
	InterruptID string            `json:"interruptId"`       // 中断 ID
	Type        InterruptType     `json:"type"`              // 中断类型: approval | confirm | select
	Description string            `json:"description"`       // 人类可读的原因描述
	Details     *InterruptDetails `json:"details,omitempty"` // 详细信息
}

// InterruptType 中断类型
type InterruptType string

const (
	InterruptTypeApproval InterruptType = "approval" // 工具调用审批
	InterruptTypeConfirm  InterruptType = "confirm"  // 确认提示
	InterruptTypeSelect   InterruptType = "select"   // 选项选择
)

// InterruptDetails 中断详细信息
type InterruptDetails struct {
	// 工具调用信息（用于 approval 类型）
	ToolName string                 `json:"toolName,omitempty"` // 工具名称
	ToolArgs map[string]interface{} `json:"toolArgs,omitempty"` // 工具参数
	ToolDesc string                 `json:"toolDesc,omitempty"` // 工具描述

	// 选择信息（用于 select 类型）
	Options    []InterruptOption `json:"options,omitempty"`    // 选项列表
	Multiple   bool              `json:"multiple,omitempty"`   // 是否多选
	DefaultVal string            `json:"defaultVal,omitempty"` // 默认值

	// 自定义数据
	Extra map[string]interface{} `json:"extra,omitempty"` // 额外数据
}

// InterruptOption 中断选项
type InterruptOption struct {
	Value    string `json:"value"`              // 选项值
	Label    string `json:"label"`              // 显示标签
	Desc     string `json:"desc,omitempty"`     // 选项描述
	Disabled bool   `json:"disabled,omitempty"` // 是否禁用
}

// ApprovalDecision 审批决定
type ApprovalDecision string

const (
	ApprovalDecisionApprove ApprovalDecision = "approve" // 批准
	ApprovalDecisionReject  ApprovalDecision = "reject"  // 拒绝
	ApprovalDecisionCancel  ApprovalDecision = "cancel"  // 取消
	ApprovalDecisionModify  ApprovalDecision = "modify"  // 修改后继续
)

// =============================================================================
// 辅助函数
// =============================================================================

// Encode 序列化 Message 为 JSON 并追加换行符
func Encode(msg Message) ([]byte, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// EncodeCompact 序列化 Message 为 JSON（紧凑格式，用于 SSE data 字段）
func EncodeCompact(msg Message) (string, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// =============================================================================
// 组件构建工具
// =============================================================================

// NewTextComponent 创建文本组件（静态值）
func NewTextComponent(id, value, usageHint string) Component {
	return Component{
		ID: id,
		Component: ComponentValue{
			Text: &TextComp{
				Value:     value,
				UsageHint: usageHint,
			},
		},
	}
}

// NewTextComponentWithDataKey 创建文本组件（数据绑定）
func NewTextComponentWithDataKey(id, dataKey, usageHint string) Component {
	return Component{
		ID: id,
		Component: ComponentValue{
			Text: &TextComp{
				DataKey:   dataKey,
				UsageHint: usageHint,
			},
		},
	}
}

// NewColumnComponent 创建列布局组件
func NewColumnComponent(id string, children []string) Component {
	return Component{
		ID: id,
		Component: ComponentValue{
			Column: &ColumnComp{
				Children: children,
			},
		},
	}
}

// NewRowComponent 创建行布局组件
func NewRowComponent(id string, children []string) Component {
	return Component{
		ID: id,
		Component: ComponentValue{
			Row: &RowComp{
				Children: children,
			},
		},
	}
}

// NewCardComponent 创建卡片组件
func NewCardComponent(id string, children []string) Component {
	return Component{
		ID: id,
		Component: ComponentValue{
			Card: &CardComp{
				Children: children,
			},
		},
	}
}

// =============================================================================
// 消息构建工具
// =============================================================================

// NewBeginRendering 创建开始渲染消息
func NewBeginRendering(surfaceID, rootID string) Message {
	return Message{
		BeginRendering: &BeginRenderingMsg{
			SurfaceID: surfaceID,
			Root:      rootID,
		},
	}
}

// NewSurfaceUpdate 创建组件更新消息
func NewSurfaceUpdate(surfaceID string, components []Component) Message {
	return Message{
		SurfaceUpdate: &SurfaceUpdateMsg{
			SurfaceID:  surfaceID,
			Components: components,
		},
	}
}

// NewDataModelUpdate 创建数据模型更新消息
func NewDataModelUpdate(surfaceID, key, value string) Message {
	return Message{
		DataModelUpdate: &DataModelUpdateMsg{
			SurfaceID: surfaceID,
			Contents: []DataContent{
				{Key: key, ValueString: value},
			},
		},
	}
}

// NewDeleteSurface 创建删除会话消息
func NewDeleteSurface(surfaceID string) Message {
	return Message{
		DeleteSurface: &DeleteSurfaceMsg{
			SurfaceID: surfaceID,
		},
	}
}

// NewInterruptRequest 创建中断请求消息（基础版）
func NewInterruptRequest(interruptID, description string) Message {
	return Message{
		InterruptRequest: &InterruptRequestMsg{
			InterruptID: interruptID,
			Description: description,
		},
	}
}

// NewApprovalInterruptRequest 创建工具审批中断请求
func NewApprovalInterruptRequest(interruptID, toolName string, toolArgs map[string]interface{}, desc string) Message {
	return Message{
		InterruptRequest: &InterruptRequestMsg{
			InterruptID: interruptID,
			Type:        InterruptTypeApproval,
			Description: desc,
			Details: &InterruptDetails{
				ToolName: toolName,
				ToolArgs: toolArgs,
				ToolDesc: desc,
			},
		},
	}
}

// NewConfirmInterruptRequest 创建确认中断请求
func NewConfirmInterruptRequest(interruptID, description string, options []InterruptOption) Message {
	return Message{
		InterruptRequest: &InterruptRequestMsg{
			InterruptID: interruptID,
			Type:        InterruptTypeConfirm,
			Description: description,
			Details: &InterruptDetails{
				Options: options,
			},
		},
	}
}

// NewSelectInterruptRequest 创建选择中断请求
func NewSelectInterruptRequest(interruptID, description string, options []InterruptOption, multiple bool) Message {
	return Message{
		InterruptRequest: &InterruptRequestMsg{
			InterruptID: interruptID,
			Type:        InterruptTypeSelect,
			Description: description,
			Details: &InterruptDetails{
				Options:  options,
				Multiple: multiple,
			},
		},
	}
}
