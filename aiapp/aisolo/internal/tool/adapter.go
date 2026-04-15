package tool

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// =============================================================================
// ToolFunc -> tool.BaseTool 适配器
// =============================================================================

// ToolAdapter 将 ToolFunc 适配为 eino 的 tool.BaseTool
type ToolAdapter struct {
	name string
	fn   ToolFunc
}

// NewToolAdapter 创建工具适配器
func NewToolAdapter(name string, fn ToolFunc) tool.BaseTool {
	return &ToolAdapter{
		name: name,
		fn:   fn,
	}
}

// Info 返回工具信息
func (t *ToolAdapter) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return t.fn.Info(ctx)
}

// InvokableRun 执行工具
func (t *ToolAdapter) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.fn.Invoke(ctx, argumentsInJSON)
}

// Ensure ToolAdapter 实现 tool.InvokableTool
var _ tool.InvokableTool = (*ToolAdapter)(nil)

// =============================================================================
// 批量转换工具
// =============================================================================

// ToEinoTools 将 Registry 中的所有工具转换为 eino 的 tool.BaseTool 列表
func (r *Registry) ToEinoTools() []tool.BaseTool {
	tools := make([]tool.BaseTool, 0, len(r.tools))
	for name, fn := range r.tools {
		tools = append(tools, NewToolAdapter(name, fn))
	}
	return tools
}

// GetEinoTool 获取单个 eino 工具
func (r *Registry) GetEinoTool(name string) (tool.BaseTool, bool) {
	fn, ok := r.Get(name)
	if !ok {
		return nil, false
	}
	return NewToolAdapter(name, fn), true
}
