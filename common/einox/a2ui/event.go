package a2ui

import (
	"encoding/json"
)

// Surface 渲染表面类型
type Surface string

const (
	SurfaceText     Surface = "text"     // 纯文本
	SurfaceMarkdown Surface = "markdown" // Markdown
	SurfaceHTML     Surface = "html"     // HTML
	SurfaceJSON     Surface = "json"     // JSON
	SurfaceImage    Surface = "image"    // 图片
	SurfaceCard     Surface = "card"     // 卡片
	SurfaceTable    Surface = "table"    // 表格
	SurfaceChart    Surface = "chart"    // 图表
)

// Component 组件类型
type Component string

const (
	ComponentText     Component = "text"
	ComponentCode     Component = "code"
	ComponentImage    Component = "image"
	ComponentFile     Component = "file"
	ComponentTable    Component = "table"
	ComponentChart    Component = "chart"
	ComponentToolCall Component = "tool_call"
	ComponentThinking Component = "thinking"
	ComponentError    Component = "error"
	ComponentLoading  Component = "loading"
)

// DataModel 数据模型类型
type DataModel string

const (
	DataModelNone   DataModel = "none"
	DataModelList   DataModel = "list"
	DataModelObject DataModel = "object"
	DataModelTree   DataModel = "tree"
	DataModelGraph  DataModel = "graph"
)

// Event A2UI 事件
type Event struct {
	Type      EventType `json:"type"`      // 事件类型
	ID        string    `json:"id"`        // 事件 ID
	Surface   Surface   `json:"surface"`   // 渲染表面
	Component Component `json:"component"` // 组件类型
	Data      any       `json:"data"`      // 数据
	Error     string    `json:"error"`     // 错误信息
}

// EventType 事件类型
type EventType string

const (
	EventStart      EventType = "start"       // 开始
	EventChunk      EventType = "chunk"       // 数据块
	EventEnd        EventType = "end"         // 结束
	EventError      EventType = "error"       // 错误
	EventComponent  EventType = "component"   // 组件
	EventDataModel  EventType = "data_model"  // 数据模型
	EventThinking   EventType = "thinking"    // 思考中
	EventToolCall   EventType = "tool_call"   // 工具调用
	EventToolResult EventType = "tool_result" // 工具结果
	EventComplete   EventType = "complete"    // 完成
)

// TextEvent 文本事件
type TextEvent struct {
	Content string `json:"content"` // 文本内容
	Finish  bool   `json:"finish"`  // 是否完成
}

// MarkdownEvent Markdown 事件
type MarkdownEvent struct {
	Content string `json:"content"` // Markdown 内容
	Finish  bool   `json:"finish"`  // 是否完成
}

// ToolCallEvent 工具调用事件
type ToolCallEvent struct {
	ID   string          `json:"id"`   // 调用 ID
	Name string          `json:"name"` // 工具名称
	Args json.RawMessage `json:"args"` // 参数
}

// ToolResultEvent 工具结果事件
type ToolResultEvent struct {
	ID     string `json:"id"`     // 调用 ID
	Name   string `json:"name"`   // 工具名称
	Result string `json:"result"` // 结果
	Error  string `json:"error"`  // 错误
}

// CodeEvent 代码块事件
type CodeEvent struct {
	Language string `json:"language"` // 语言
	Code     string `json:"code"`     // 代码
	Finish   bool   `json:"finish"`   // 是否完成
}

// TableEvent 表格事件
type TableEvent struct {
	Headers []string   `json:"headers"` // 表头
	Rows    [][]string `json:"rows"`    // 行数据
	Finish  bool       `json:"finish"`  // 是否完成
}

// ChartEvent 图表事件
type ChartEvent struct {
	Type   string `json:"type"`   // 图表类型: bar, line, pie, scatter
	Title  string `json:"title"`  // 标题
	XAxis  []any  `json:"x_axis"` // X 轴数据
	YAxis  []any  `json:"y_axis"` // Y 轴数据
	Finish bool   `json:"finish"` // 是否完成
}

// =============================================================================
// 事件构建工具
// =============================================================================

// NewTextEvent 创建文本事件
func NewTextEvent(content string, finish bool) *Event {
	return &Event{
		Type:      EventChunk,
		Surface:   SurfaceText,
		Component: ComponentText,
		Data: TextEvent{
			Content: content,
			Finish:  finish,
		},
	}
}

// NewMarkdownEvent 创建 Markdown 事件
func NewMarkdownEvent(content string, finish bool) *Event {
	return &Event{
		Type:      EventChunk,
		Surface:   SurfaceMarkdown,
		Component: ComponentText,
		Data: MarkdownEvent{
			Content: content,
			Finish:  finish,
		},
	}
}

// NewToolCallEvent 创建工具调用事件
func NewToolCallEvent(id, name string, args any) (*Event, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	return &Event{
		Type:      EventToolCall,
		ID:        id,
		Component: ComponentToolCall,
		Data: ToolCallEvent{
			ID:   id,
			Name: name,
			Args: argsJSON,
		},
	}, nil
}

// NewToolResultEvent 创建工具结果事件
func NewToolResultEvent(id, name, result, errMsg string) *Event {
	return &Event{
		Type:      EventToolResult,
		ID:        id,
		Component: ComponentToolCall,
		Data: ToolResultEvent{
			ID:     id,
			Name:   name,
			Result: result,
			Error:  errMsg,
		},
	}
}

// NewCodeEvent 创建代码块事件
func NewCodeEvent(language, code string, finish bool) *Event {
	return &Event{
		Type:      EventChunk,
		Surface:   SurfaceText,
		Component: ComponentCode,
		Data: CodeEvent{
			Language: language,
			Code:     code,
			Finish:   finish,
		},
	}
}

// NewStartEvent 创建开始事件
func NewStartEvent() *Event {
	return &Event{
		Type: EventStart,
	}
}

// NewEndEvent 创建结束事件
func NewEndEvent() *Event {
	return &Event{
		Type: EventEnd,
	}
}

// NewErrorEvent 创建错误事件
func NewErrorEvent(err string) *Event {
	return &Event{
		Type:      EventError,
		Component: ComponentError,
		Error:     err,
	}
}

// NewThinkingEvent 创建思考事件
func NewThinkingEvent(content string, finish bool) *Event {
	return &Event{
		Type:      EventThinking,
		Component: ComponentThinking,
		Data: map[string]any{
			"content": content,
			"finish":  finish,
		},
	}
}
